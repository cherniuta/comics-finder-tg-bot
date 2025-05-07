package core

import "sync"

type ServiceStatus string

const (
	StatusRunning ServiceStatus = "running"
	StatusIdle    ServiceStatus = "idle"
)

type SearchQuery struct {
	Keywords string
	Limit    int
}

type Comics struct {
	ID    int
	URL   string
	Words []string
}

type NormQuery struct {
	Words []string
	Limit int
}

type Index struct {
	index map[string][]int
	lock  sync.RWMutex
}

func NewIndex() *Index {
	return &Index{
		index: make(map[string][]int),
	}
}

func (i *Index) Drop() {
	i.lock.Lock()
	i.index = make(map[string][]int)
	i.lock.Unlock()
}

func (i *Index) Add(id int, words []string) {
	i.lock.Lock()
	for _, word := range words {
		i.index[word] = append(i.index[word], id)
	}
	i.lock.Unlock()
}

func (i *Index) Get(word string) []int {
	i.lock.RLock()
	defer i.lock.RUnlock()
	ids := make([]int, 0, 1000)
	ids = append(ids, i.index[word]...)
	return ids
}
