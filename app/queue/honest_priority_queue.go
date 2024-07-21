package queue

import (
	"container/heap"
	"github.com/pkg/errors"
	"sync"
	"time"
)

// HonestJobQueue It ain't much, but it's an honest job.jpg
// Wraps PriorityQueue to make it thread-safe. Manages priorities.
// Extremely inefficient, but works for my use-case (very slow jobs and small queue sizes)
type HonestJobQueue struct {
	mu          *sync.RWMutex
	queue       PriorityQueue
	users       map[int64]int // Tracks the amount of job per-user currently in the queue. Used to calculate priority
	banned      map[int64]any // Drop jobs from these users
	maintenance bool
}

func NewHonestJobQueue(initialCapacity int) *HonestJobQueue {
	return &HonestJobQueue{
		mu:     &sync.RWMutex{},
		queue:  make(PriorityQueue, 0, initialCapacity),
		users:  make(map[int64]int),
		banned: make(map[int64]any),
	}
}

// BanUser This will "ban" the user (if they were impatient and banned the bot first)
// causing their jobs to be dropped when they pop up. Once all the jobs have been popped
// the ban will be lifted
func (hjq *HonestJobQueue) BanUser(userID int64) {
	hjq.mu.Lock()
	defer hjq.mu.Unlock()

	hjq.banned[userID] = nil
}

func (hjq *HonestJobQueue) updatePriorities(userID int64) {
	for i, job := range hjq.queue {
		if job.userID != userID {
			continue
		}
		job.priority--
		// I don't want very active users to get stuck forever with lower priority, but I DO want them to "re-enter" the queue
		job.insertionTime = time.Now()
		// It's fine to do Fix here, the job will always get moved to the _left_, we won't see the same job twice
		heap.Fix(&hjq.queue, i)
	}
}

func (hjq *HonestJobQueue) Len() int {
	hjq.mu.RLock()
	defer hjq.mu.RUnlock()

	return len(hjq.queue)
}

func (hjq *HonestJobQueue) Stats() (int, int) {
	hjq.mu.RLock()
	defer hjq.mu.RUnlock()

	return len(hjq.queue), len(hjq.users)
}

func (hjq *HonestJobQueue) Pop() *Job {
	hjq.mu.Lock()
	defer hjq.mu.Unlock()

	job := heap.Pop(&hjq.queue).(*Job)

	if _, ok := hjq.banned[job.userID]; ok {
		allBannedJobsExhausted := true
		for _, queued := range hjq.queue {
			if queued.userID == job.userID {
				allBannedJobsExhausted = false
				break
			}
		}

		if allBannedJobsExhausted {
			delete(hjq.banned, job.userID)
		}

		return hjq.Pop()
	}

	hjq.users[job.userID]--

	if hjq.users[job.userID] == 0 {
		delete(hjq.users, job.userID)
	}

	hjq.updatePriorities(job.userID)

	return job
}

func (hjq *HonestJobQueue) ToggleMaintenance() bool {
	hjq.mu.Lock()
	defer hjq.mu.Unlock()

	hjq.maintenance = !hjq.maintenance

	return hjq.maintenance
}

func (hjq *HonestJobQueue) Push(userID int64, runnable func()) error {
	hjq.mu.Lock()
	defer hjq.mu.Unlock()

	if hjq.maintenance {
		return errors.New("The server is on temporary maintenance, no new videos are being processed at the moment, try again later")
	}

	if hjq.queue.Len() > 2000 {
		return errors.New("There are too many items queued already, try again later")
	}

	priority := hjq.users[userID]

	if priority > 2 {
		hjq.users[userID]--
		return errors.New("You're distorting videos too often, wait until the previous ones have been processed")
	}

	hjq.users[userID] = priority + 1

	job := newJob(userID, priority, runnable)
	heap.Push(&hjq.queue, &job)

	return nil
}
