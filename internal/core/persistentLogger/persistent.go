package persistentLogger

import (
	"context"
	"go-cache-server-mini/internal/config"
	"go-cache-server-mini/internal/core/data"
	"log"
	"maps"
	"sync"
	"sync/atomic"
)

type Command struct {
	Action string
	Key    string
	Item   data.CacheItem
}

type cacheChannel struct {
	aofControl chan string
	aofData    chan string
	snapData   chan map[string]data.CacheItem
	snapDone   chan bool
}

type PersistentLogger struct {
	ctx        context.Context
	aofLogger  *AOF
	snapLogger *Snap
	parser     *Parser
	cacheChan  cacheChannel
	closeOnce  sync.Once      // to ensure Close is only called once
	closed     int32          // atomic flag to indicate if closed
	ops        sync.WaitGroup // to track ongoing operations
}

func NewPersistentLogger(ctx context.Context, config *config.Config) *PersistentLogger {
	// initialize cache channels
	cacheChan := cacheChannel{
		aofControl: make(chan string),
		aofData:    make(chan string, 1000),
		snapData:   make(chan map[string]data.CacheItem),
		snapDone:   make(chan bool),
	}

	snapLogger := NewSnap(cacheChan.snapData, cacheChan.snapDone, config.Persistent.Path)
	aofLogger := NewAOF(cacheChan.aofData, cacheChan.aofControl, config.Persistent.Path)

	persistentLogger := &PersistentLogger{
		ctx:        ctx,
		aofLogger:  aofLogger,
		snapLogger: snapLogger,
		parser:     NewParser(),
		cacheChan:  cacheChan,
	}
	return persistentLogger
}

// Close gracefully shuts down the PersistentLogger, ensuring all pending operations are completed.
func (p *PersistentLogger) Close() {
	p.closeOnce.Do(func() {
		atomic.StoreInt32(&p.closed, 1)
		p.ops.Wait()

		close(p.cacheChan.aofData)
		close(p.cacheChan.snapData)
		close(p.cacheChan.aofControl)

		p.aofLogger.Wait()
		p.snapLogger.Wait()
	})
}

func (p *PersistentLogger) Load(data map[string]data.CacheItem) (map[string]data.CacheItem, error) {
	var snapLoadErr error
	data, snapLoadErr = p.snapLogger.Load(data)
	if snapLoadErr != nil {
		return data, snapLoadErr
	}

	var aofLoadErr error
	data, aofLoadErr = p.aofLogger.Load(data)
	if aofLoadErr != nil {
		return data, aofLoadErr
	}
	return data, nil
}

func (p *PersistentLogger) WriteAOF(command Command) {
	if atomic.LoadInt32(&p.closed) == 1 {
		return
	}
	p.ops.Add(1)
	defer p.ops.Done()

	cmd, err := p.parser.ConvertCMDToString(command.Action, command.Key, command.Item)
	if err != nil {
		return
	}
	select {
	case <-p.ctx.Done():
		return
	case p.cacheChan.aofData <- cmd:
	}
}

func (p *PersistentLogger) TriggerSnap(kvmap map[string]data.CacheItem, lock *sync.RWMutex) {
	if atomic.LoadInt32(&p.closed) == 1 {
		return
	}
	p.ops.Add(1)
	defer p.ops.Done()

	log.Println("Triggering snapshot...")
	select {
	case <-p.ctx.Done():
		return
	case p.cacheChan.aofControl <- "PAUSE":
	}

	lock.RLock()
	duplicatedData := make(map[string]data.CacheItem, len(kvmap))
	maps.Copy(duplicatedData, kvmap)
	lock.RUnlock()

	select {
	case <-p.ctx.Done():
		return
	case p.cacheChan.snapData <- duplicatedData:
	}

	if !p.waitForSnapDoneAck() {
		return
	}

	select {
	case <-p.ctx.Done():
		return
	case p.cacheChan.aofControl <- "RESUME":
	}
	log.Println("Snapshot completed.")
}

func (p *PersistentLogger) waitForSnapDoneAck() bool {
	for {
		select {
		case _, ok := <-p.cacheChan.snapDone:
			return ok
		case <-p.ctx.Done():
			_, ok := <-p.cacheChan.snapDone
			return ok
		}
	}
}
