package common

import "sync"

type SyncMap[K comparable, V any] struct {
	mutex *sync.Mutex
	data  map[K]V
}

func NewSyncMap[K comparable, V any]() *SyncMap[K, V] {
	return &SyncMap[K, V]{
		mutex: &sync.Mutex{},
		data:  make(map[K]V),
	}
}

func (smap *SyncMap[K, V]) Set(key K, value V) {
	smap.mutex.Lock()
	defer smap.mutex.Unlock()
	smap.data[key] = value
}

func (smap *SyncMap[K, V]) Get(key K) (value V, ok bool) {
	smap.mutex.Lock()
	defer smap.mutex.Unlock()
	value, ok = smap.data[key]
	return
}

func (smap *SyncMap[K, V]) Del(key K) {
	smap.mutex.Lock()
	defer smap.mutex.Unlock()
	delete(smap.data, key)
}

func (smap *SyncMap[K, V]) ForEach(action func(K, V)) {
	smap.mutex.Lock()
	defer smap.mutex.Unlock()
	for k, v := range smap.data {
		action(k, v)
	}
}

func (smap *SyncMap[K, V]) Len() int {
	smap.mutex.Lock()
	defer smap.mutex.Unlock()
	return len(smap.data)
}
