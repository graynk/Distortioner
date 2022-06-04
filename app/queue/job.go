package queue

import "time"

type Job struct {
	runnable     func()    // The job itself
	userID       int64     // ID of the user. Used to calculate priority
	priority     int       // The priority of the item in the queue. Lesser numbers mean bigger priority. Calculated by the HonestJobQueue
	creationTime time.Time // Needed to maintain insertion-order for items with equal priority.
}

func newJob(userID int64, priority int, runnable func()) Job {
	return Job{
		runnable:     runnable,
		userID:       userID,
		priority:     priority,
		creationTime: time.Now(),
	}
}

func (j Job) Run() {
	j.runnable()
}
