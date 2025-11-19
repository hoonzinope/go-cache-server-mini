package router

import (
	"context"
	"go-cache-server-mini/internal/distributed/adapter"
	"sync"
)

type NodeRouter struct {
	adapterMap   sync.Map // node-ip to adapter
	localAdapter adapter.AdapterInterface
}

func NewNodeRouter(ctx context.Context, localAdapter adapter.AdapterInterface) *NodeRouter {
	return &NodeRouter{
		adapterMap:   sync.Map{},
		localAdapter: localAdapter,
	}
}

func (nr *NodeRouter) GetLocalAdapter() adapter.AdapterInterface {
	return nr.localAdapter
}

func (nr *NodeRouter) GetAdapters(key string) ([]adapter.AdapterInterface, error) {
	adapters := []adapter.AdapterInterface{}
	return adapters, nil
}

func (nr *NodeRouter) GetAllAdapters() ([]adapter.AdapterInterface, error) {
	adapters := []adapter.AdapterInterface{}
	adapters = append(adapters, nr.localAdapter)
	return adapters, nil
}
