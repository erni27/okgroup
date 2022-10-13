// Package okgroup provides a synchronisation mechanism for group of goroutines
// executing functions having the same signature.
package okgroup

import (
	"context"
	"errors"
	"sync"
)

// An Error is a group's error containing errors from all goroutines if a group fails.
type Error struct {
	errors []error
}

func (e Error) Error() string {
	var msg string
	for _, err := range e.errors {
		msg += err.Error() + ";"
	}
	return msg[:len(msg)-1]
}

func (e Error) Is(target error) bool {
	for _, err := range e.errors {
		if errors.Is(err, target) {
			return true
		}
	}
	return false
}

// A Group is a collection of goroutines executing functions
// having the same signature func() (T, error) where T is any type.
type Group[T any] struct {
	cancel func()
	wg     sync.WaitGroup
	errCh  chan error
	okCh   chan T
}

// WithContext returns a new Group and a derived Context from a given ctx.
//
// The derived Context is canceled if a function passed to Go returns
// an ok response or the first time Wait returns.
func WithContext[T any](ctx context.Context) (*Group[T], context.Context) {
	ctx, cancel := context.WithCancel(ctx)
	return &Group[T]{cancel: cancel, errCh: make(chan error), okCh: make(chan T, 1)}, ctx
}

// Go executes a given function in a new goroutine.
//
// The first function returning an ok response cancel the group's context,
// if the group was created by calling WithContext.
// The ok response is returned by Wait.
func (g *Group[T]) Go(f func() (T, error)) {
	g.wg.Add(1)
	go func() {
		defer g.wg.Done()
		ok, err := f()
		if err != nil {
			g.errCh <- err
			return
		}
		select {
		case g.okCh <- ok:
			if g.cancel != nil {
				g.cancel()
			}
		default:
		}
	}()
}

// Wait blocks until all function calls from the Go method have returned.
//
// If there is an ok response then Wait returns the ok response and a nil error,
// otherwise a T zero value is returned along with the group's error.
func (g *Group[T]) Wait() (T, error) {
	go func() {
		g.wg.Wait()
		if g.cancel != nil {
			g.cancel()
		}
		close(g.errCh)
	}()
	var grouperr Error
	for err := range g.errCh {
		grouperr.errors = append(grouperr.errors, err)
	}
	select {
	case ok := <-g.okCh:
		return ok, nil
	default:
		var ok T
		return ok, grouperr
	}
}
