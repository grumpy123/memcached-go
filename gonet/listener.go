package gonet

import (
	"context"
	"fmt"
	"net"
	"sync"
)

type ConnectionHandler interface {
	// New handles new connection, it should return only when the connection has been closed
	New(conn net.Conn, done <-chan struct{})
}

type Listener struct {
	handler ConnectionHandler
	addr    string

	listener net.Listener
	done     chan struct{}
}

func NewListener(port int, handler ConnectionHandler) *Listener {
	return NewListenerForAddr(fmt.Sprintf(":%d", port), handler)
}

func NewListenerForAddr(addr string, handler ConnectionHandler) *Listener {
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

type TrackingConnectionHandler struct {
	inner   ConnectionHandler
	tracker sync.WaitGroup
}

func WithTracking(handler ConnectionHandler) *TrackingConnectionHandler {
	return &TrackingConnectionHandler{
		inner:   handler,
		tracker: sync.WaitGroup{},
	}
}

func (tc *TrackingConnectionHandler) New(conn net.Conn, done <-chan struct{}) {
	tc.tracker.Add(1)
	tc.inner.New(conn, done)
	tc.tracker.Done()
}

func (tc *TrackingConnectionHandler) Wait() {
	tc.tracker.Wait()
}

func (tc *TrackingConnectionHandler) Done() <-chan struct{} {
	done := make(chan struct{})
	go func() {
		tc.tracker.Wait()
		close(done)
	}()

	return done
}
