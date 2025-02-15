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

	requests chan *pendingRequest
}

type pendingRequest struct {
	request   Request
	completed chan struct{}
}

func NewServer(handler RequestHandler, conn net.Conn, done <-chan struct{}) *Server {
	s := &Server{
		handler: handler,
		conn:    conn,
		done:    done,

		requests: make(chan *pendingRequest, 1024), // todo: parameter
	}
	return s
}

func (s *Server) Run() {
	go s.requestLoop()
	s.responseLoop()
}

func (s *Server) requestLoop() {
	defer close(s.requests)

	reader := bufio.NewReaderSize(s.conn, 1024)
	for {
		request, err := s.handler.ReadRequest(reader)
		if err != nil {
			return
		}
		pending := &pendingRequest{request: request, completed: make(chan struct{})}
		go s.handle(pending)
		s.requests <- pending
	}
}

func (s *Server) handle(pending *pendingRequest) {
	// todo: if we ever use this in production, we'd want to handle panics and have a timeout here
	pending.request.Handle()
	close(pending.completed)
}

func (s *Server) responseLoop() {
	defer s.close()

	writer := bufio.NewWriter(s.conn)
	for {
		select {
		case pending, ok := <-s.requests:
			if !ok {
				return
			}
			<-pending.completed
			err := pending.request.WriteResponse(writer)
			if err != nil {
				return
			}
			err = writer.Flush()
			if err != nil {
				return
			}
		case <-s.done:
			// drain ready work and exit
			for {
				select {
				case pending, ok := <-s.requests:
					if !ok {
						return
					}
					<-pending.completed // todo: have to be careful here, add a timeout to not block forever
					err := pending.request.WriteResponse(writer)
					if err != nil {
						return
					}
					err = writer.Flush()
					if err != nil {
						return
					}
				default:
					// No more ready work
					return
				}
			}
		}
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
