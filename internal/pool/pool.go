package pool

import "sync"

type Resetter interface {
	Reset()
}

type Pool[T Resetter] struct {
	freeObjects []T

	mu            sync.Mutex
	newObjectFunc func() T
}

func New[T Resetter](
	newObjectFunc func() T,
) *Pool[T] {
	result := &Pool[T]{
		newObjectFunc: newObjectFunc,
		freeObjects:   make([]T, 0),
	}

	return result
}

func (p *Pool[T]) Put(object T) {
	object.Reset()
	p.mu.Lock()
	defer p.mu.Unlock()

	p.freeObjects = append(p.freeObjects, object)
}

func (p *Pool[T]) Get() T {
	p.mu.Lock()

	if len(p.freeObjects) == 0 {
		p.mu.Unlock()
		return p.newObjectFunc()
	}

	lastIndex := len(p.freeObjects) - 1

	object := p.freeObjects[lastIndex]
	var zero T
	p.freeObjects[lastIndex] = zero
	p.freeObjects = p.freeObjects[:lastIndex]

	p.mu.Unlock()

	return object
}
