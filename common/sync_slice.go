package common

import "sync"

type SyncSlice[T any] struct {
	mutex *sync.Mutex
	data  []T
}

func NewSyncSlice[T any]() *SyncSlice[T] {
	return &SyncSlice[T]{
		mutex: &sync.Mutex{},
		data:  make([]T, 0),
	}
}

func SyncSliceFrom[T any](slice []T) *SyncSlice[T] {
	return &SyncSlice[T]{
		mutex: &sync.Mutex{},
		data:  slice,
	}
}

func (slice *SyncSlice[T]) Get(idx int) T {
	slice.mutex.Lock()
	defer slice.mutex.Unlock()
	return slice.data[idx]
}

func (slice *SyncSlice[T]) Set(idx int, value T) {
	slice.mutex.Lock()
	defer slice.mutex.Unlock()
	slice.data[idx] = value
}

func (slice *SyncSlice[T]) Append(values ...T) {
	slice.mutex.Lock()
	defer slice.mutex.Unlock()
	slice.data = append(slice.data, values...)
}

func (slice *SyncSlice[T]) ForEach(action func(int, T)) {
	slice.mutex.Lock()
	defer slice.mutex.Unlock()
	for i, val := range slice.data {
		action(i, val)
	}
}

func (slice *SyncSlice[T]) SliceUpper(from int) {
	slice.mutex.Lock()
	defer slice.mutex.Unlock()
	slice.data = slice.data[from:]
}

func (slice *SyncSlice[T]) Len() int {
	slice.mutex.Lock()
	defer slice.mutex.Unlock()
	return len(slice.data)
}
