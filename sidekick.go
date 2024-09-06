package unison

import (
	"context"
	"errors"
	"sync"
)

type SidekickGroup[R any] struct {
	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup
	res    chan result[R]
}

func NewSidekickGroup[R any](ctx context.Context) *SidekickGroup[R] {
	g := &SidekickGroup[R]{
		wg:  sync.WaitGroup{},
		res: make(chan result[R], 1),
	}
	g.ctx, g.cancel = context.WithCancel(ctx)
	return g
}

func (g *SidekickGroup[R]) putResult(value R, err error) {
	select {
	case g.res <- result[R]{value, err}:
	default:
	}
}

func (g *SidekickGroup[R]) putErr(err error) {
	var zero R
	g.putResult(zero, err)
}

func (g *SidekickGroup[R]) Main(fn func(context.Context) (R, error)) {
	g.wg.Add(1)
	go func() {
		defer g.wg.Done()
		defer func() {
			if r := recover(); r != nil {
				g.putErr(panicError{
					panic: r,
				})
				g.cancel()
			}
		}()

		value, err := fn(g.ctx)
		g.putResult(value, err)
		g.cancel() // Stop sidekicks
	}()
}

func (g *SidekickGroup[R]) Sidekick(fn func(context.Context) error) {
	g.wg.Add(1)
	go func() {
		defer g.wg.Done()
		defer func() {
			// Sidekick panic, stop other side tasks and the main task.
			if r := recover(); r != nil {
				g.putErr(panicError{
					panic: r,
				})
				g.cancel()
			}
		}()

		if err := fn(g.ctx); err != nil {
			g.putErr(err)
			g.cancel()
		}
	}()
}

func (g *SidekickGroup[R]) Wait() (R, error) {
	g.wg.Wait()
	res := <-g.res
	var pe panicError
	if errors.As(res.err, &pe) {
		panic(pe.panic)
	}
	return res.value, res.err
}
