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
		go nodeRouter.updateRemoteNodes()
	}
	return nodeRouter, nil
}

func (nr *NodeRouter) updateRemoteNodes() {
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
			nr._updateRemoteNodes(remoteIPs)
		}
	}
}

func (nr *NodeRouter) _updateRemoteNodes(remoteIPs []string) {
	nr.mu.Lock()
	defer nr.mu.Unlock()

	// Add new remote nodes
	remoteIPsSet := make(map[string]struct{})
	for _, ip := range remoteIPs {
		remoteIPsSet[ip] = struct{}{}
	}

	for _, ip := range remoteIPs {
		if _, exists := nr.nodeIpSet[ip]; !exists {
			err := nr.addRemoteAdapter(ip)
			if err != nil {
				log.Printf("failed to add remote adapter for IP %s: %v\n", ip, err)
			}
		}
	}

	for ip := range nr.nodeIpSet {
		if ip == nr.localIp {
			continue
		}
		_, found := remoteIPsSet[ip]
		if !found {
			err := nr.removeRemoteAdapter(ip)
			if err != nil {
				log.Printf("failed to remove adapter for IP %s: %v\n", ip, err)
			}
		}
	}
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
	return nr.addNode(nodeIP, adapter)
}

func (nr *NodeRouter) addRemoteAdapter(nodeIP string) error {
	if _, exists := nr.nodeIpSet[nodeIP]; !exists {
		remoteAdapter := adapter.NewRemoteAdapter(nodeIP)
		return nr.addNode(nodeIP, remoteAdapter)
	}

	return nil
}

func (nr *NodeRouter) addNode(nodeIP string, adapter adapter.AdapterInterface) error {
	nr.mu.Lock()
	defer nr.mu.Unlock()

	for i := 0; i < nr.replicas; i++ {
		hash := util.Fnv32aHash(fmt.Sprintf("%s-%d", nodeIP, i))
		nr.nodeMap.Store(hash, adapter)
		nr.hashes = append(nr.hashes, hash)
	}
	slices.Sort(nr.hashes)
	nr.nodeIpSet[nodeIP] = struct{}{}
	return nil
}

func (nr *NodeRouter) removeRemoteAdapter(nodeIP string) error {
	if _, exists := nr.nodeIpSet[nodeIP]; !exists {
		return fmt.Errorf("node IP %s not found", nodeIP)
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
	return nil
}
