package pool

import (
	"errors"
	"sync"
)

var ErrFuncNil = errors.New("pool function is nil")

type hasReset interface {
	Reset()
}

type Pool[T hasReset] struct {
	pool *sync.Pool
}

func New[T hasReset](f func() T) (*Pool[T], error) {
	if f == nil {
		return nil, ErrFuncNil
	}
	return &Pool[T]{
		pool: &sync.Pool{
			New: func() any {
				return f()
			},
		},
	}, nil
}

func (p *Pool[T]) Get() T {
	return p.pool.Get().(T)
}

func (p *Pool[T]) Put(obj T) {
	obj.Reset()
	p.pool.Put(obj)
}
