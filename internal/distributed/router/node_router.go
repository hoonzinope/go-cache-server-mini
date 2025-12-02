package router

import (
	"context"
	"fmt"
	"go-cache-server-mini/internal/config"
	"go-cache-server-mini/internal/distributed/adapter"
	"go-cache-server-mini/internal/util"
	"log"
	"slices"
	"sync"
	"time"
)

type NodeRouter struct {
	ctx            context.Context
	replicas       int                      // number of virtual nodes per physical node
	backupNodes    int                      // number of backup nodes
	nodeIpSet      map[string]struct{}      // set of node IPs
	nodeMap        sync.Map                 // hash to adapter mapping
	hashes         []uint32                 // sorted hash ring
	localIp        string                   // local node IP
	localAdapter   adapter.AdapterInterface // local adapter
	mu             sync.RWMutex
	clusterManager *ClusterManager
	updateInterval int64 // in seconds
}

func NewNodeRouter(ctx context.Context, localAdapter adapter.AdapterInterface, clusterManager *ClusterManager, config config.Config) (*NodeRouter, error) {
	distributed_enabled := config.Distributed.Enabled
	remote_port := config.Distributed.GRPCPort
	update_interval := config.Distributed.UpdateInterval
	replication_factor := config.Distributed.ReplicationFactor
	backup_nodes := config.Distributed.BackupNodes

	localIp, localIpReturnErr := clusterManager.GetLocalNodeIP()
	if localIpReturnErr != nil {
		return nil, fmt.Errorf("failed to get local node IP: %v", localIpReturnErr)
	}

	nodeRouter := &NodeRouter{
		ctx:            ctx,
		replicas:       replication_factor,        // number of virtual nodes per physical node
		backupNodes:    backup_nodes,              // backup nodes, 0 means no backup
		nodeMap:        sync.Map{},                // node-ip to adapter mapping
		localIp:        localIp,                   // local node IP
		localAdapter:   localAdapter,              // local adapter
		nodeIpSet:      make(map[string]struct{}), // set of node IPs
		hashes:         []uint32{},                // sorted hash ring
		clusterManager: clusterManager,            // cluster manager
		updateInterval: update_interval,           // in seconds
	}
	nodeRouter.addLocalAdapter(localIp, localAdapter)
	if distributed_enabled {
		go nodeRouter.updateRemoteNodes(remote_port)
	}
	return nodeRouter, nil
}

func (nr *NodeRouter) updateRemoteNodes(remotePort int) {
	ticker := time.NewTicker(time.Duration(nr.updateInterval) * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-nr.ctx.Done():
			return
		case <-ticker.C:
			remoteIPs, err := nr.clusterManager.GetRemoteNodeIPs()
			if err != nil {
				log.Printf("failed to get remote node IPs: %v\n", err)
				continue
			}
			nr._updateRemoteNodes(remoteIPs, remotePort)
		}
	}
}

func (nr *NodeRouter) _updateRemoteNodes(remoteIPs []string, remotePort int) {
	nr.mu.RLock()
	currentIPs := make(map[string]struct{}, len(nr.nodeIpSet))
	for ip := range nr.nodeIpSet {
		currentIPs[ip] = struct{}{}
	}
	nr.mu.RUnlock()

	remoteIPsSet := make(map[string]struct{})
	for _, ip := range remoteIPs {
		remoteIPsSet[ip] = struct{}{}
	}

	ipsToAdd := []string{}
	ipsToRemove := []string{}

	for ip := range remoteIPsSet {
		if ip == nr.localIp {
			continue
		}
		if _, exists := currentIPs[ip]; !exists {
			ipsToAdd = append(ipsToAdd, ip)
		}
	}

	for ip := range currentIPs {
		if ip == nr.localIp {
			continue
		}
		if _, exists := remoteIPsSet[ip]; !exists {
			ipsToRemove = append(ipsToRemove, ip)
		}
	}

	// --- 2단계: 락 외부에서 시간이 오래 걸리는 작업 수행 ---

	// 새 어댑터 생성 (네트워크 I/O)
	newAdapters := make(map[string]adapter.AdapterInterface)
	for _, ip := range ipsToAdd {
		// NewRemoteAdapter는 내부적으로 gRPC 클라이언트를 생성하므로 락 외부에서 호출
		remoteAdapter, err := adapter.NewRemoteAdapter(ip, remotePort)
		if err != nil {
			log.Printf("failed to create remote adapter for IP %s: %v\n", ip, err)
			continue
		}
		newAdapters[ip] = remoteAdapter
	}

	// 제거할 어댑터 목록 가져오기 및 Close 호출 (네트워크 I/O)
	adaptersToClose := nr.collectAdaptersToClose(ipsToRemove)
	for _, adp := range adaptersToClose {
		adp.Close() // Close는 블로킹될 수 있으므로 락 외부에서 호출
	}

	// --- 3단계: 락을 짧게 잡고 메모리 구조만 갱신 ---

	nr.mu.Lock()
	defer nr.mu.Unlock()

	// 새 노드 추가
	for ip, adapter := range newAdapters {
		// addNode는 hashes, nodeMap, nodeIpSet을 수정
		nr.addNode(ip, adapter)
	}

	// 기존 노드 제거
	for _, ip := range ipsToRemove {
		// removeNodeHashes는 hashes, nodeMap, nodeIpSet을 수정
		nr.removeNodeHashes(ip)
	}

	// addNode에서 hashes 슬라이스가 정렬이 필요하므로 다시 정렬
	slices.Sort(nr.hashes)
}

// removeRemoteAdapter를 대체할 헬퍼 함수들

// 제거할 노드 IP들에 해당하는 어댑터들을 수집하는 함수 (읽기 락 사용)
func (nr *NodeRouter) collectAdaptersToClose(ipsToRemove []string) []*adapter.RemoteAdapter {
	adaptersToClose := []*adapter.RemoteAdapter{}
	uniqueAdapters := map[*adapter.RemoteAdapter]struct{}{}

	nr.mu.RLock()
	defer nr.mu.RUnlock()

	for _, ip := range ipsToRemove {
		for i := 0; i < nr.replicas; i++ {
			hash := util.Fnv32aHash(fmt.Sprintf("%s-%d", ip, i))
			if adapterInterface, ok := nr.nodeMap.Load(hash); ok {
				if remoteAdapter, ok := adapterInterface.(*adapter.RemoteAdapter); ok {
					if _, exists := uniqueAdapters[remoteAdapter]; !exists {
						uniqueAdapters[remoteAdapter] = struct{}{}
						adaptersToClose = append(adaptersToClose, remoteAdapter)
					}
				}
			}
		}
	}
	return adaptersToClose
}

// 메모리에서만 노드 정보를 제거하는 함수 (외부에서 이미 락을 잡고 호출해야 함)
func (nr *NodeRouter) removeNodeHashes(nodeIP string) {
	if _, exists := nr.nodeIpSet[nodeIP]; !exists {
		return
	}

	hashToRemove := make(map[uint32]struct{}, nr.replicas)
	for i := 0; i < nr.replicas; i++ {
		hash := util.Fnv32aHash(fmt.Sprintf("%s-%d", nodeIP, i))
		nr.nodeMap.Delete(hash)
		hashToRemove[hash] = struct{}{}
	}

	newHashes := make([]uint32, 0, len(nr.hashes)-len(hashToRemove))
	for _, hash := range nr.hashes {
		if _, found := hashToRemove[hash]; !found {
			newHashes = append(newHashes, hash)
		}
	}
	nr.hashes = newHashes
	delete(nr.nodeIpSet, nodeIP)
}

func (nr *NodeRouter) GetLocalAdapter() adapter.AdapterInterface {
	return nr.localAdapter
}

func (nr *NodeRouter) GetAdapters(key string) ([]adapter.AdapterInterface, error) {
	nr.mu.RLock()
	defer nr.mu.RUnlock()

	adapters := []adapter.AdapterInterface{}
	if len(nr.hashes) == 0 {
		return adapters, nil
	}
	hash := util.Fnv32aHash(key)
	// Find the nodes >= hash
	idx, _ := slices.BinarySearch(nr.hashes, hash)
	uniqueNode := make(map[adapter.AdapterInterface]struct{})
	resultAdapters := make([]adapter.AdapterInterface, 0, nr.backupNodes+1)
	// Get primary + backup nodes
	for i := 0; i < len(nr.hashes) && len(resultAdapters) < nr.backupNodes+1; i++ {
		current_idx := (idx + i) % len(nr.hashes)
		nodeHash := nr.hashes[current_idx]
		adapterInterface, ok := nr.nodeMap.Load(nodeHash)
		if !ok {
			continue
		}
		node, ok := adapterInterface.(adapter.AdapterInterface)
		if !ok {
			continue
		}
		if _, exists := uniqueNode[node]; !exists {
			uniqueNode[node] = struct{}{}
			resultAdapters = append(resultAdapters, node)
		}
	}
	return resultAdapters, nil
}

func (nr *NodeRouter) GetAllAdapters() ([]adapter.AdapterInterface, error) {
	nr.mu.RLock()
	defer nr.mu.RUnlock()

	uniqueAdapters := make(map[adapter.AdapterInterface]struct{})
	for _, hash := range nr.hashes {
		adapterInterface, ok := nr.nodeMap.Load(hash)
		if !ok {
			continue
		}
		adapterInst := adapterInterface.(adapter.AdapterInterface)
		uniqueAdapters[adapterInst] = struct{}{}
	}
	adapters := make([]adapter.AdapterInterface, 0, len(uniqueAdapters))
	for adapterInst := range uniqueAdapters {
		adapters = append(adapters, adapterInst)
	}
	return adapters, nil
}

func (nr *NodeRouter) addLocalAdapter(nodeIP string, adapter adapter.AdapterInterface) error {
	nr.mu.Lock()
	defer nr.mu.Unlock()
	return nr.addNode(nodeIP, adapter)
}

// func (nr *NodeRouter) addRemoteAdapter(nodeIP string, remotePort int) error {
// 	if _, exists := nr.nodeIpSet[nodeIP]; !exists {
// 		remoteAdapter, err := adapter.NewRemoteAdapter(nr.ctx, nodeIP, remotePort)
// 		if err != nil {
// 			log.Printf("failed to create remote adapter for IP %s: %v\n", nodeIP, err)
// 			return err
// 		}
// 		return nr.addNode(nodeIP, remoteAdapter)
// 	}

// 	return nil
// }

func (nr *NodeRouter) addNode(nodeIP string, adapter adapter.AdapterInterface) error {
	for i := 0; i < nr.replicas; i++ {
		hash := util.Fnv32aHash(fmt.Sprintf("%s-%d", nodeIP, i))
		nr.nodeMap.Store(hash, adapter)
		nr.hashes = append(nr.hashes, hash)
	}
	nr.nodeIpSet[nodeIP] = struct{}{}
	return nil
}

// func (nr *NodeRouter) removeRemoteAdapter(nodeIP string) error {
// 	if _, exists := nr.nodeIpSet[nodeIP]; !exists {
// 		return fmt.Errorf("node IP %s not found", nodeIP)
// 	}

// 	closeAdapters := map[*adapter.RemoteAdapter]struct{}{}

// 	hashToRemove := make(map[uint32]struct{}, nr.replicas)
// 	for i := 0; i < nr.replicas; i++ {
// 		hash := util.Fnv32aHash(fmt.Sprintf("%s-%d", nodeIP, i))
// 		adapterInterface, ok := nr.nodeMap.Load(hash)
// 		if ok {
// 			if remoteAdapter, ok := adapterInterface.(*adapter.RemoteAdapter); ok {
// 				closeAdapters[remoteAdapter] = struct{}{}
// 			}
// 		}
// 		nr.nodeMap.Delete(hash)
// 		hashToRemove[hash] = struct{}{}
// 	}
// 	newHashes := make([]uint32, 0, len(nr.hashes)-len(hashToRemove))
// 	for _, hash := range nr.hashes {
// 		if _, found := hashToRemove[hash]; !found {
// 			newHashes = append(newHashes, hash)
// 		}
// 	}
// 	nr.hashes = newHashes
// 	delete(nr.nodeIpSet, nodeIP)
// 	for adapterInst := range closeAdapters {
// 		adapterInst.Close()
// 	}
// 	return nil
// }
