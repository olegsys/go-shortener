package service

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"
)

type Deleter interface {
	DeleteURLs(ctx context.Context, userID string, ids []string) error
}

type deleteTask struct {
	userID  string
	shortID string
}

type DeletionService struct {
	deleter Deleter
	ch      chan deleteTask
	ctx     context.Context
	cancel  context.CancelFunc
	stopped atomic.Bool
	wg      sync.WaitGroup
}

func NewDeletionService(deleter Deleter) *DeletionService {
	ctx, cancel := context.WithCancel(context.Background())
	ds := &DeletionService{
		deleter: deleter,
		ch:      make(chan deleteTask, 100),
		ctx:     ctx,
		cancel:  cancel,
	}
	go ds.runWorker(ctx)
	return ds
}

func (ds *DeletionService) Enqueue(userID, shortID string) error {
	if ds.stopped.Load() {
		return fmt.Errorf("deletion service is shutting down")
	}
	ds.ch <- deleteTask{userID: userID, shortID: shortID}
	return nil
}

func (ds *DeletionService) Shutdown() {
	ds.stopped.Store(true)
	ds.cancel()
	close(ds.ch)
	ds.wg.Wait()
}

func (ds *DeletionService) runWorker(ctx context.Context) {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()
	buffer := make([]deleteTask, 0, 50)

	for {
		select {
		case task, ok := <-ds.ch:
			if !ok {
				if len(buffer) > 0 {
					ds.flush(buffer)
				}
				return
			}
			buffer = append(buffer, task)
			if len(buffer) >= 50 {
				ds.flush(buffer)
				buffer = buffer[:0]
			}
		case <-ticker.C:
			if len(buffer) > 0 {
				ds.flush(buffer)
				buffer = buffer[:0]
			}
		case <-ctx.Done():
			if len(buffer) > 0 {
				ds.flush(buffer)
			}
			return
		}
	}
}

func (ds *DeletionService) flush(tasks []deleteTask) {
	groups := make(map[string][]string)
	for _, t := range tasks {
		groups[t.userID] = append(groups[t.userID], t.shortID)
	}
	for userID, ids := range groups {
		ds.wg.Add(1)
		go func(uid string, shortIDs []string) {
			defer ds.wg.Done()
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			_ = ds.deleter.DeleteURLs(ctx, uid, shortIDs)
		}(userID, ids)
	}
}
