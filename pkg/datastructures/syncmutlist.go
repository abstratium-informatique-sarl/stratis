package datastructures

import "sync"

type SyncMutList[T any] struct {
	items *MutList[T]
	mutex sync.Mutex
}

func NewSyncMutList[T any]() *SyncMutList[T] {
	l := make([]T, 0, 10)
	return &SyncMutList[T]{
		items: &MutList[T]{
			items: l,
		},
		mutex: sync.Mutex{},
	}
}

func NewSyncMutListFromArray[T any](initial []T) *SyncMutList[T] {
	lst := NewMutListFromArray[T](initial)
	return &SyncMutList[T]{
		items: lst,
		mutex: sync.Mutex{},
	}
}

func (l *SyncMutList[T]) Add(item T) {
	l.mutex.Lock()
	defer l.mutex.Unlock()
	l.items.Add(item)
}

func (l *SyncMutList[T]) AddAll(items []T) {
	l.mutex.Lock()
	defer l.mutex.Unlock()
	l.items.AddAll(items)
}

func (l *SyncMutList[T]) Remove(item T) {
	l.mutex.Lock()
	defer l.mutex.Unlock()
	l.items.Remove(item)
}

func (l *SyncMutList[T]) RemoveIf(f func(T) bool) {
	l.mutex.Lock()
	defer l.mutex.Unlock()
	l.items.RemoveIf(f)
}

func (l *SyncMutList[T]) RemoveAfter(index int) {
	l.mutex.Lock()
	defer l.mutex.Unlock()
	l.items.RemoveAfter(index)
}

func (l *SyncMutList[T]) Contains(item T) bool {
	l.mutex.Lock()
	defer l.mutex.Unlock()
	return l.items.Contains(item)
}

func (l *SyncMutList[T]) Get(index int) T {
	l.mutex.Lock()
	defer l.mutex.Unlock()
	return l.items.Get(index)
}

func (l *SyncMutList[T]) Find(f func(t T) bool) (T, error) {
	l.mutex.Lock()
	defer l.mutex.Unlock()
	return l.items.Find(f)
}

func (l *SyncMutList[T]) Len() int {
	l.mutex.Lock()
	defer l.mutex.Unlock()
	return l.items.Len()
}

func (l *SyncMutList[T]) Clear() {
	l.mutex.Lock()
	defer l.mutex.Unlock()
	l.items.Clear()
}

func (l *SyncMutList[T]) Items() []T {
	l.mutex.Lock()
	defer l.mutex.Unlock()
	return l.items.Items()
}

func (l *SyncMutList[T]) SortFunc(f func(a, b T) int) {
	l.mutex.Lock()
	defer l.mutex.Unlock()
	l.items.SortFunc(f)
}

func (l *SyncMutList[T]) Filter(f func(T) bool) *SyncMutList[T] {
	l.mutex.Lock()
	defer l.mutex.Unlock()
	return NewSyncMutListFromArray[T](l.items.Filter(f).Items())
}

func (l *SyncMutList[T]) Head(n int) *SyncMutList[T] {
	l.mutex.Lock()
	defer l.mutex.Unlock()
	items := l.items.Head(n).Items()
	return NewSyncMutListFromArray[T](items)
}
