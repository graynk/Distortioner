package queue

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestHonestJobQueue_InsertionOrder(t *testing.T) {
	hjq := NewHonestJobQueue(50)
	for id := int64(1); id < 4; id++ {
		hjq.Push(id, func() {})
	}
	assert.Equal(t, 3, hjq.Len())
	for id := int64(1); id < 4; id++ {
		job := hjq.Pop()
		assert.Equal(t, id, job.userID)
	}
	assert.Equal(t, 0, hjq.Len())
}

func TestHonestJobQueue_RepeatUsers(t *testing.T) {
	hjq := NewHonestJobQueue(50)

	// three jobs by user 1
	for i := 0; i < 3; i++ {
		hjq.Push(1, func() {})
	}
	// one job from user 3
	hjq.Push(3, func() {})
	// two jobs from user 2
	for i := 0; i < 2; i++ {
		hjq.Push(2, func() {})
	}

	assert.Equal(t, 6, hjq.Len())

	poppedIDs := make([]int64, 0, 6)
	for i := 0; i < 6; i++ {
		poppedIDs = append(poppedIDs, hjq.Pop().userID)
	}

	assert.Equal(t, []int64{1, 3, 2, 1, 2, 1}, poppedIDs)
}
