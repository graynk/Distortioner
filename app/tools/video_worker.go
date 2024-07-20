package tools

import (
	"github.com/graynk/distortioner/queue"
)

type VideoWorker struct {
	queue       *queue.HonestJobQueue // the queue itself. separate from the channel, since we can't sort stuff in channels
	messenger   chan interface{}      // if there's something in the channel - there's something in the queue.
	workerCount int
}

func NewVideoWorker(workerCount int) *VideoWorker {
	capacity := 300
	worker := VideoWorker{
		queue:       queue.NewHonestJobQueue(capacity),
		messenger:   make(chan interface{}, capacity),
		workerCount: workerCount,
	}
	for i := 0; i < workerCount; i++ {
		go worker.run()
	}
	return &worker
}

func (vw *VideoWorker) BanUser(userID int64) {
	vw.queue.BanUser(userID)
}

func (vw *VideoWorker) run() {
	for range vw.messenger {
		vw.queue.Pop().Run()
	}
}

func (vw *VideoWorker) Submit(userID int64, runnable func()) error {
	err := vw.queue.Push(userID, runnable)
	if err != nil {
		return err
	}
	vw.messenger <- nil // let goroutines know that there's something in the queue
	return nil
}

func (vw *VideoWorker) Shutdown() {
	close(vw.messenger)
}

func (vw *VideoWorker) QueueStats() (int, int) {
	return vw.queue.Stats()
}

func (vw *VideoWorker) IsBusy() bool {
	return vw.queue.Len() > 0
}
