package datastructures

import (
	"cmp"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMutList_Add(t *testing.T) {
	assert := assert.New(t)
	list := NewMutList[int]()
	list.Add(1)
	list.Add(2)
	list.Add(3)
	assert.Equal(3, list.Len())
	assert.Equal([]int{1, 2, 3}, list.Items())
}

func TestMutList_Remove(t *testing.T) {
	assert := assert.New(t)
	list := NewMutList[int]()
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

func TestMutList_RemoveIf(t *testing.T) {
	assert := assert.New(t)
	list := NewMutList[int]()
	list.Add(1)
	list.Add(2)
	list.Add(3)
	list.RemoveIf(func(i int) bool { return i%2 == 0 })
	assert.Equal(2, list.Len())
	assert.Equal([]int{1, 3}, list.Items())
}

func TestMutList_RemoveIf_Empty(t *testing.T) {
	assert := assert.New(t)
	list := NewMutList[int]()
	list.RemoveIf(func(i int) bool { return true })
	assert.Equal(0, list.Len())
	assert.Equal([]int{}, list.Items())
}

func TestMutList_RemoveIf_OneLeft(t *testing.T) {
	assert := assert.New(t)
	list := NewMutList[int]()
	list.Add(1)
	list.RemoveIf(func(i int) bool { return false })
	assert.Equal(1, list.Len())
	assert.Equal([]int{1}, list.Items())
}

func TestMutList_RemoveIf_NoneLeft(t *testing.T) {
	assert := assert.New(t)
	list := NewMutList[int]()
	list.Add(1)
	list.RemoveIf(func(i int) bool { return true })
	assert.Equal(0, list.Len())
	assert.Equal([]int{}, list.Items())
}

func TestMutList_RemoveIf_Last(t *testing.T) {
	assert := assert.New(t)
	list := NewMutList[int]()
	list.Add(1)
	list.Add(2)
	list.RemoveIf(func(i int) bool { return i == 2 })
	assert.Equal(1, list.Len())
	assert.Equal([]int{1}, list.Items())
}

func TestMutList_RemoveIf_First(t *testing.T) {
	assert := assert.New(t)
	list := NewMutList[int]()
	list.Add(1)
	list.Add(2)
	list.RemoveIf(func(i int) bool { return i == 1 })
	assert.Equal(1, list.Len())
	assert.Equal([]int{2}, list.Items())
}

func TestMutList_Contains(t *testing.T) {
	assert := assert.New(t)
	list := NewMutList[int]()
	list.Add(1)
	list.Add(2)
	list.Add(3)
	assert.True(list.Contains(2))
	assert.False(list.Contains(4))
}

func TestMutList_Get(t *testing.T) {
	assert := assert.New(t)
	list := NewMutList[int]()
	list.Add(1)
	list.Add(2)
	list.Add(3)
	assert.Equal(1, list.Get(0))
	assert.Equal(2, list.Get(1))
	assert.Equal(3, list.Get(2))
}

func TestMutList_Len(t *testing.T) {
	assert := assert.New(t)
	list := NewMutList[int]()
	assert.Equal(0, list.Len())
	list.Add(1)
	assert.Equal(1, list.Len())
	list.Add(2)
	assert.Equal(2, list.Len())
}

func TestMutList_Clear(t *testing.T) {
	assert := assert.New(t)
	list := NewMutList[int]()
	list.Add(1)
	list.Add(2)
	list.Add(3)
	list.Clear()
	assert.Equal(0, list.Len())
	assert.Equal([]int{}, list.Items())
}

func TestMutList_Items(t *testing.T) {
	assert := assert.New(t)
	list := NewMutList[int]()
	list.Add(1)
	list.Add(2)
	list.Add(3)
	items := list.Items()
	assert.Equal([]int{1, 2, 3}, items)
	// Modify the returned slice and ensure the original list is not modified
	items[0] = 99
	assert.Equal(1, list.Get(0))
}

func TestMutList_NewFromArray(t *testing.T) {
	assert := assert.New(t)
	list := NewMutListFromArray[int]([]int{4, 5, 6})
	assert.Equal(3, list.Len())
	assert.Equal([]int{4, 5, 6}, list.Items())
}

// a test for the SortFunc method
func TestMutList_SortFunc(t *testing.T) {
	assert := assert.New(t)
	list := NewMutList[int]()
	list.Add(3)
	list.Add(1)
	list.Add(2)
	list.SortFunc(func(a, b int) int {
		return cmp.Compare(a, b)
	})
	assert.Equal([]int{1, 2, 3}, list.Items())
}

func TestMutList_Filter(t *testing.T) {
	assert := assert.New(t)
	list := NewMutList[int]()
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

func TestMutList_Head(t *testing.T) {
	assert := assert.New(t)
	list := NewMutList[int]()
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
