package main

import "sync"

type hasReset interface {
	Reset()
}

type Pool[T hasReset] struct {
	pool *sync.Pool
}

func New[T hasReset](f func() T) *Pool[T] {
	return &Pool[T]{
		pool: &sync.Pool{
			New: func() any {
				return f()
			},
		},
	}
}

func (p *Pool[T]) Get() T {
	return p.pool.Get().(T)
}

func (p *Pool[T]) Put(obj T) {
	obj.Reset()
	p.pool.Put(obj)
}
