package queue

import (
	"container/heap"
	"sync"
	"time"
)

// HonestJobQueue It ain't much, but it's an honest job.jpg
// Wraps PriorityQueue to make it thread-safe. Manages priorities.
// Extremely inefficient, but works for my use-case (very slow jobs and small queue sizes)
type HonestJobQueue struct {
	mu    *sync.RWMutex
	queue PriorityQueue
	users map[int64]int // Tracks the amount of job per-user currently in the queue. Used to calculate priority
}

func NewHonestJobQueue(initialCapacity int) *HonestJobQueue {
	return &HonestJobQueue{
		mu:    &sync.RWMutex{},
		queue: make(PriorityQueue, 0, initialCapacity),
		users: make(map[int64]int),
	}
}

func (hjq *HonestJobQueue) updatePriorities(userID int64) {
	for i, job := range hjq.queue {
		if job.userID != userID {
			continue
		}
		job.priority--
		// I don't want very active users to get stuck forever with lower priority, but I DO want them to "re-enter" the queue
		job.insertionTime = time.Now()
		heap.Fix(&hjq.queue, i)
	}
}
func (hjq *HonestJobQueue) Len() int {
	hjq.mu.RLock()
	defer hjq.mu.RUnlock()

	return len(hjq.queue)
}

func (hjq *HonestJobQueue) Pop() *Job {
	hjq.mu.Lock()
	defer hjq.mu.Unlock()

	job := heap.Pop(&hjq.queue).(*Job)
	hjq.users[job.userID]--

	hjq.updatePriorities(job.userID)

	return job
}

func (hjq *HonestJobQueue) Push(userID int64, runnable func()) {
	hjq.mu.Lock()
	defer hjq.mu.Unlock()

	hjq.users[userID]++
	priority := hjq.users[userID]

	job := newJob(userID, priority, runnable)
	heap.Push(&hjq.queue, &job)
}
