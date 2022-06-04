package queue

//PriorityQueue taken pretty much as-is from https://pkg.go.dev/container/heap.
//
//Only change is that smaller numbers == bigger priority, and it maintains insertion order for items with equal priority
type PriorityQueue []*Job

func (pq PriorityQueue) Len() int { return len(pq) }

func (pq PriorityQueue) Less(i, j int) bool {
	if pq[i].priority == pq[j].priority {
		return pq[i].insertionTime.Before(pq[j].insertionTime)
	}
	return pq[i].priority < pq[j].priority
}

func (pq PriorityQueue) Swap(i, j int) {
	pq[i], pq[j] = pq[j], pq[i]
}

func (pq *PriorityQueue) Push(x any) {
	item := x.(*Job)
	*pq = append(*pq, item)
}

func (pq *PriorityQueue) Pop() any {
	old := *pq
	last := len(old) - 1
	item := old[last]
	old[last] = nil // avoid memory leak
	*pq = old[:last]
	return item
}
