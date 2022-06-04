package tools

import (
	"sync"
)

const AllowedOverTime = 3
const TimePeriodSeconds = 300

type MessageRange struct {
	StartUtc int64
	Count    int
}

type RateLimiter struct {
	usersRate map[int64]MessageRange
	mu        sync.Mutex
}

func NewRateLimiter() *RateLimiter {
	return &RateLimiter{
		usersRate: make(map[int64]MessageRange),
		mu:        sync.Mutex{},
	}
}

func (r *RateLimiter) GetRateOverPeriod(userId int64, utc int64) (int, int64) {
	r.mu.Lock()
	defer r.mu.Unlock()
	messageRange, ok := r.usersRate[userId]
	if !ok {
		r.usersRate[userId] = MessageRange{
			StartUtc: utc,
			Count:    1,
		}
		return 1, 0
	}
	var updatedStamp MessageRange
	// crude, but will do
	diff := utc - messageRange.StartUtc
	if diff < TimePeriodSeconds {
		updatedStamp = MessageRange{
			StartUtc: messageRange.StartUtc, // keep the original range
			Count:    messageRange.Count + 1,
		}
	} else {
		updatedStamp = MessageRange{
			StartUtc: utc, // set up a new range
			Count:    1,
		}
	}
	r.usersRate[userId] = updatedStamp
	return updatedStamp.Count, TimePeriodSeconds - diff
}
