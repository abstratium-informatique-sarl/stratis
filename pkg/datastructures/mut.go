package datastructures

// an OO class for a mutable object, using generics

type Mut[T any] struct {
	value T
}

func (m *Mut[T]) SetValue(value T) {
	m.value = value
}

func (m *Mut[T]) GetValue() T {
	return m.value
}

func NewMut[T any](value T) *Mut[T] {
	return &Mut[T]{
		value: value,
	}
}
