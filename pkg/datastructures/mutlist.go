package datastructures

import (
	"fmt"
	"slices"
)

type MutList[T any] struct {
	items []T
}

func NewMutList[T any]() *MutList[T] {
	l := make([]T, 0, 10)
	return &MutList[T]{items: l}
}

func NewMutListFromArray[T any](initial []T) *MutList[T] {
	l := make([]T, 0, 10)
	l = append(l, initial...)
	return &MutList[T]{items: l}
}

func (l *MutList[T]) Add(item T) {
	l.items = append(l.items, item)
}

func (l *MutList[T]) AddAll(items []T) {
	l.items = append(l.items, items...)
}

func (l *MutList[T]) Remove(item T) {
	for i, v := range l.items {
		var a any = v
		var b any = item
		if a == b {
			l.items = append(l.items[:i], l.items[i+1:]...)
			return
		}
	}
}

func (l *MutList[T]) RemoveIf(f func(T) bool) {
	copyList := make([]T, 0, l.Len())
	for _, v := range l.items {
		if !f(v) {
			copyList = append(copyList, v)
		}
	}
	l.items = copyList
}

func (l *MutList[T]) RemoveAfter(index int) {
	l.items = l.items[:index]
}

func (l *MutList[T]) Contains(item T) bool {
	for _, v := range l.items {
		var a any = v
		var b any = item
		if a == b {
			return true
		}
	}
	return false
}

var NotFound = fmt.Errorf("not found")

func (l *MutList[T]) Find(f func(t T) bool) (T, error) {
	for _, v := range l.items {
		if f(v) {
			return v, nil
		}
	}
	var nf T;
	return nf, NotFound
}

func (l *MutList[T]) Get(index int) T {
	return l.items[index]
}

func (l *MutList[T]) Len() int {
	return len(l.items)
}

func (l *MutList[T]) Clear() {
	l.items = make([]T, 0, 10)
}

func (l *MutList[T]) Items() []T {
	list := make([]T, len(l.items))
	copy(list, l.items)
	return list
}

func (l *MutList[T]) SortFunc(f func(a, b T) int) {
	slices.SortFunc(l.items, f)
}

func (l *MutList[T]) Filter(f func(T) bool) *MutList[T] {
	filtered := make([]T, 0, len(l.items))
	for _, v := range l.items {
		if f(v) {
			filtered = append(filtered, v)
		}
	}
	return NewMutListFromArray(filtered)
}

func (l *MutList[T]) Head(n int) *MutList[T] {
	if n > len(l.items) {
		n = len(l.items)
	}
	return NewMutListFromArray(l.items[:n])
}
