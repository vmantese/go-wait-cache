package lmap

import "sync"

type WaitCache struct {
	m *sync.Mutex
	locks *sync.Map //map[string]*sync.Mutex
	cache *sync.Map
}

type CacheWaiter interface {
	LoadOrWait(key interface{}) (interface{}, bool)
	Store(key interface{}, value interface{})
}

func NewWaitCache() *WaitCache {
	return &WaitCache{
		m: &sync.Mutex{},
		locks: &sync.Map{},
		cache: &sync.Map{},
	}
}

func (l *WaitCache) LoadOrWait(key interface{}) (v interface{}, ok bool) {
	// early exit for routines that come in well after value is loaded
	if v, ok = l.cache.Load(key); ok {
		return v, ok
	}
	if existingMutex := l.lock(key); existingMutex != nil {
		//todo add timeout
		existingMutex.Lock() // will block until l.Store(key) is called

		defer existingMutex.Unlock()
	}
	//exit for routines that called load while value is locked
	return l.cache.Load(key)
}

func (l *WaitCache) lock(key interface{}) (existingMutex *sync.Mutex) {
	l.m.Lock()

	defer l.m.Unlock()

	var mutex *sync.Mutex
	if m, ok := l.locks.Load(key); ok {
		return m.(*sync.Mutex)
	}
	mutex = &sync.Mutex{}
	mutex.Lock()
	l.locks.Store(key, mutex)
	return nil // there was no existing mutex
}

func (l *WaitCache) Store(key interface{}, value interface{}) {
	l.m.Lock()

	defer l.m.Unlock()

	mutex, _ := l.locks.Load(key) // assumed to be locked if present
	l.cache.Store(key, value)
	//if this is present, should always be locked
	if mutex != nil {
		mutex.(*sync.Mutex).Unlock()
		l.locks.Delete(key)
	}
}
