package datastructures

import (
	"cmp"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSyncMutList_Add(t *testing.T) {
	assert := assert.New(t)
	list := NewSyncMutList[int]()
	list.Add(1)
	list.Add(2)
	list.Add(3)
	assert.Equal(3, list.Len())
	assert.Equal([]int{1, 2, 3}, list.Items())
}

func TestSyncMutList_Remove(t *testing.T) {
	assert := assert.New(t)
	list := NewSyncMutList[int]()
	list.Add(1)
	list.Add(2)
	list.Add(3)
	list.Remove(2)
	assert.Equal(2, list.Len())
	assert.Equal([]int{1, 3}, list.Items())

	list.Remove(1)
	assert.Equal(1, list.Len())
	assert.Equal([]int{3}, list.Items())

	list.Remove(3)
	assert.Equal(0, list.Len())
	assert.Equal([]int{}, list.Items())

	// Try removing an item that doesn't exist
	list.Remove(4)
	assert.Equal(0, list.Len())
	assert.Equal([]int{}, list.Items())
}

func TestSyncMutList_RemoveIf(t *testing.T) {
	assert := assert.New(t)
	list := NewSyncMutList[int]()
	list.Add(1)
	list.Add(2)
	list.Add(3)
	list.RemoveIf(func(i int) bool { return i%2 == 0 })
	assert.Equal(2, list.Len())
	assert.Equal([]int{1, 3}, list.Items())
}

func TestSyncMutList_Contains(t *testing.T) {
	assert := assert.New(t)
	list := NewSyncMutList[int]()
	list.Add(1)
	list.Add(2)
	list.Add(3)
	assert.True(list.Contains(2))
	assert.False(list.Contains(4))
}

func TestSyncMutList_Get(t *testing.T) {
	assert := assert.New(t)
	list := NewSyncMutList[int]()
	list.Add(1)
	list.Add(2)
	list.Add(3)
	assert.Equal(1, list.Get(0))
	assert.Equal(2, list.Get(1))
	assert.Equal(3, list.Get(2))
}

func TestSyncMutList_Len(t *testing.T) {
	assert := assert.New(t)
	list := NewSyncMutList[int]()
	assert.Equal(0, list.Len())
	list.Add(1)
	assert.Equal(1, list.Len())
	list.Add(2)
	assert.Equal(2, list.Len())
}

func TestSyncMutList_Clear(t *testing.T) {
	assert := assert.New(t)
	list := NewSyncMutList[int]()
	list.Add(1)
	list.Add(2)
	list.Add(3)
	list.Clear()
	assert.Equal(0, list.Len())
	assert.Equal([]int{}, list.Items())
}

func TestSyncMutList_Items(t *testing.T) {
	assert := assert.New(t)
	list := NewSyncMutList[int]()
	list.Add(1)
	list.Add(2)
	list.Add(3)
	items := list.Items()
	assert.Equal([]int{1, 2, 3}, items)
	// Modify the returned slice and ensure the original list is not modified
	items[0] = 99
	assert.Equal(1, list.Get(0))
}

// Test that it's safe to use in different threads (e.g. concurrency)
func TestSyncMutList_Concurrent(t *testing.T) {
	assert := assert.New(t)
	list := NewSyncMutList[int]()
	
	// use lots of threads, so that it's more likely to fail if there is a bug
	numGoroutines := 100
	numOperations := 1000
	
	// start lots of threads
	done := make(chan struct{}, numGoroutines)
	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer func(){
				done <- struct{}{}
			}()
			for j := 0; j < numOperations; j++ {
				list.Add(id*100 + j)
			}
		}(i)
	}

	// wait for them all to finish
	for i := 0; i < numGoroutines; i++ {
		<-done
	}

	// check the length
	assert.Equal(numGoroutines * numOperations, list.Len())
}

// a test for the SortFunc method
func TestSyncMutList_SortFunc(t *testing.T) {
	assert := assert.New(t)
	list := NewSyncMutList[int]()
	list.Add(3)
	list.Add(1)
	list.Add(2)
	list.SortFunc(func(a, b int) int {
		return cmp.Compare(a, b)
	})
	assert.Equal([]int{1, 2, 3}, list.Items())
}

func TestSyncMutList_Filter(t *testing.T) {
	assert := assert.New(t)
	list := NewSyncMutList[int]()
	list.Add(1)
	list.Add(2)
	list.Add(3)
	list.Add(4)
	filtered := list.Filter(func(i int) bool { return i%2 == 0 })
	assert.Equal(2, filtered.Len())
	assert.Equal([]int{2, 4}, filtered.Items())
	assert.Equal(4, list.Len())
	assert.Equal([]int{1, 2, 3, 4}, list.Items())
}

func TestSyncMutList_Head(t *testing.T) {
	assert := assert.New(t)
	list := NewSyncMutList[int]()
	list.Add(1)
	list.Add(2)
	list.Add(3)
	list.Add(4)
	list.Add(5)
	assert.Equal(5, list.Len())
	assert.Equal([]int{1, 2, 3, 4, 5}, list.Items())
	assert.Equal(3, list.Head(3).Len())
	assert.Equal([]int{1, 2, 3}, list.Head(3).Items())

	list = list.Head(3)
	assert.Equal(3, list.Len())
	assert.Equal([]int{1, 2, 3}, list.Items())
}
