package gonet

import (
	"bufio"
	"net"
)

type Request interface {
	Handle()
	WriteResponse(writer *bufio.Writer) error
}

type RequestHandler interface {
	ReadRequest(reader *bufio.Reader) (Request, error)
}

type Server struct {
	handler RequestHandler
	conn    net.Conn
	done    <-chan struct{}

	requests chan *PendingRequest
}

type PendingRequest struct {
	request   Request
	completed chan struct{}
}

func NewServer(handler RequestHandler, conn net.Conn, done <-chan struct{}) *Server {
	s := &Server{
		handler: handler,
		conn:    conn,
		done:    done,

		requests: make(chan *PendingRequest),
	}
	return s
}

func (s *Server) Run() {
	go s.requestLoop()
	s.responseLoop()
	s.close()
}

func (s *Server) requestLoop() {
	defer close(s.requests)

	reader := bufio.NewReaderSize(s.conn, 1024)
	for {
		request, err := s.handler.ReadRequest(reader)
		if err != nil {
			return
		}
		pending := &PendingRequest{request: request, completed: make(chan struct{})}
		s.requests <- pending
		go s.handle(pending)
	}
}

func (s *Server) handle(pending *PendingRequest) {
	pending.request.Handle()
	close(pending.completed)
}

func (s *Server) responseLoop() {
	writer := bufio.NewWriter(s.conn)
	for pending := range s.requests {
		<-pending.completed
		err := pending.request.WriteResponse(writer)
		if err != nil {
			break
		}
		err = writer.Flush()
		if err != nil {
			break
		}
	}

	for _ = range s.requests {
		// discarding any remaining requests
	}
}

// close is called after:
//   - all requests were handled or
//   - there was a write error and responses cannot be sent anymore.
//
// In either case the responseLoop has completed, but the requestLoop may need to be interrupted and let finish
func (s *Server) close() {
	// todo: log errors
	_ = s.conn.Close()

	// discarding any remaining requests and waiting for the requestLoop to close the channel
	for _ = range s.requests {
	}

	s.conn = nil
}

type ServerFactory struct {
	handler RequestHandler
}

func NewServerFactory(handler RequestHandler) *ServerFactory {
	s := &ServerFactory{
		handler: handler,
	}
	return s
}

func (s *ServerFactory) New(conn net.Conn, done <-chan struct{}) {
	svr := NewServer(s.handler, conn, done)
	// No goroutine, per `ConnectionHandler.New` contract
	svr.Run()
}
