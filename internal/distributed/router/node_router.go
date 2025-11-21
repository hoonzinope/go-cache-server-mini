package router

import (
	"context"
	"fmt"
	"go-cache-server-mini/internal/distributed/adapter"
	"go-cache-server-mini/internal/util"
	"slices"
	"sync"
)

type NodeRouter struct {
	replicas    int      // number of virtual nodes per physical node
	backupNodes int      // number of backup nodes
	nodeMap     sync.Map // hash to adapter mapping
	hashes      []uint32 // sorted hash ring
	// localAdapter adapter.AdapterInterface
	mu sync.RWMutex
}

func NewNodeRouter(ctx context.Context, localAdapter adapter.AdapterInterface) *NodeRouter {
	nodeRouter := &NodeRouter{
		replicas:    3,          // number of virtual nodes per physical node, TODO : make it configurable
		backupNodes: 0,          // No backup nodes for now, TODO: implement later
		nodeMap:     sync.Map{}, // node-ip to adapter mapping
		// localAdapter: localAdapter,
		hashes: []uint32{},
	}
	nodeRouter.AddAdapter("local-node", localAdapter)
	return nodeRouter
}

func (nr *NodeRouter) GetLocalAdapter() adapter.AdapterInterface {
	nr.mu.RLock()
	defer nr.mu.RUnlock()

	node_name := fmt.Sprintf("%s-%d", "local-node", 0)
	hash := util.Fnv32aHash(node_name)
	local_adapter, ok := nr.nodeMap.Load(hash)
	if !ok {
		return nil
	}
	return local_adapter.(adapter.AdapterInterface)
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
	uniqueAdapters := make(map[adapter.AdapterInterface]struct{})
	resultAdapters := make([]adapter.AdapterInterface, 0, nr.backupNodes+1)
	// Get primary + backup nodes
	for i := 0; i < len(nr.hashes) && len(resultAdapters) < nr.backupNodes+1; i++ {
		current_idx := (idx + i) % len(nr.hashes)
		nodeHash := nr.hashes[current_idx]
		adapterInterface, ok := nr.nodeMap.Load(nodeHash)
		if !ok {
			continue
		}
		adapterInst := adapterInterface.(adapter.AdapterInterface)
		if _, visited := uniqueAdapters[adapterInst]; visited {
			continue
		}
		uniqueAdapters[adapterInst] = struct{}{}
		resultAdapters = append(resultAdapters, adapterInst)
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

func (nr *NodeRouter) AddAdapter(nodeIP string, adapter adapter.AdapterInterface) error {
	nr.mu.Lock()
	defer nr.mu.Unlock()

	for i := 0; i < nr.replicas; i++ {
		hash := util.Fnv32aHash(fmt.Sprintf("%s-%d", nodeIP, i))
		nr.nodeMap.Store(hash, adapter)
		nr.hashes = append(nr.hashes, hash)
	}
	slices.Sort(nr.hashes)
	return nil
}

func (nr *NodeRouter) RemoveAdapter(nodeIP string) error {
	nr.mu.Lock()
	defer nr.mu.Unlock()

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
	return nil
}
