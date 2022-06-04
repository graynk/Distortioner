package tools

import (
	"github.com/graynk/distortioner/queue"
)

type VideoWorker struct {
	queue     *queue.HonestJobQueue // the queue itself. separate from the channel, since we can't sort stuff in channels
	messenger chan interface{}      // if there's something in the channel - there's something in the queue.
}

func NewVideoWorker(workerCount int) *VideoWorker {
	capacity := 50
	worker := VideoWorker{
		queue:     queue.NewHonestJobQueue(capacity),
		messenger: make(chan interface{}, capacity),
	}
	for i := 0; i < workerCount; i++ {
		go worker.run()
	}
	return &worker
}

func (vw *VideoWorker) run() {
	for range vw.messenger {
		job := vw.queue.Pop()
		job.Run()
	}
}

func (vw *VideoWorker) Submit(userID int64, runnable func()) {
	vw.queue.Push(userID, runnable)
	vw.messenger <- nil // let goroutines know that there's something in the queue
}
