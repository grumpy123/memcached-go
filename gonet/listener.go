package gonet

import (
	"context"
	"fmt"
	"net"
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
		l.handler.New(conn, l.done)
	}
}

func (l *Listener) Address() net.Addr {
	return l.listener.Addr()
}

func (l *Listener) Close() error {
	close(l.done)
	return l.listener.Close()
}
