package gonet

import (
	"context"
	"fmt"
	"net"
	"sync"
)

type HandlerFactory interface {
	New(c net.Conn, done <-chan struct{})
}

type Listener struct {
	handler HandlerFactory
	addr    string

	listener net.Listener
	done     chan struct{}
}

func NewListener(port int, handler HandlerFactory) *Listener {
	return NewListenerForAddr(fmt.Sprintf(":%d", port), handler)
}

func NewListenerForAddr(addr string, handler HandlerFactory) *Listener {
	l := &Listener{
		handler: handler,
		addr:    addr,

		done: make(chan struct{}),
	}
	return l
}

func (l *Listener) Start(ctx context.Context) error {
	var lc net.ListenConfig
	listener, err := lc.Listen(ctx, "tcp", l.addr)
	if err != nil {
		return err
	}

	l.listener = listener
	go l.listen()
	return nil
}

func (l *Listener) listen() {
	for {
		conn, err := l.listener.Accept()
		if err != nil {
			return
		}
		go l.handler.New(conn, l.done)
	}
}

func (l *Listener) Address() net.Addr {
	return l.listener.Addr()
}

func (l *Listener) Close() error {
	close(l.done)
	return l.listener.Close()
}

type TrackingHandlerFactory struct {
	inner   HandlerFactory
	tracker sync.WaitGroup
}

func WithTracking(handler HandlerFactory) *TrackingHandlerFactory {
	return &TrackingHandlerFactory{
		inner:   handler,
		tracker: sync.WaitGroup{},
	}
}

func (hf *TrackingHandlerFactory) New(c net.Conn, done <-chan struct{}) {
	hf.tracker.Add(1)
	hf.inner.New(c, done)
	hf.tracker.Done()
}

func (hf *TrackingHandlerFactory) Wait() {
	hf.tracker.Wait()
}

func (hf *TrackingHandlerFactory) Done() <-chan struct{} {
	ch := make(chan struct{})
	go func() {
		hf.tracker.Wait()
		close(ch)
	}()

	return ch
}
