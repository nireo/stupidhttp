package stupidhttp

import (
	"bytes"
	"io"
	"log"
	"net"
)

type MethodType int

type HandleFunc func(*Request) (*Response, error)

const (
	MethodGet MethodType = iota
	MethodPost
	MethodPut
	MethodDelete
)

type Request struct {
	Headers map[string]string
	Body    io.ReadCloser
}

type Response struct {
	Headers    map[string]string
	StatusCode int
	Body       io.ReadCloser
}

type Route struct {
	Method     MethodType
	HandleFunc func(*Request) (*Response, error)
}

type Server struct {
	Routes  map[string]Route
	Address string
}

func (s *Server) AddHandler(path string, method MethodType, handlerFunc HandleFunc) {
	s.Routes[path] = Route{
		Method:     method,
		HandleFunc: handlerFunc,
	}
}

func (s *Server) Start() error {
	l, err := net.Listen("tcp", s.Address)
	if err != nil {
		return err
	}
	defer l.Close()

	for {
		c, err := l.Accept()
		if err != nil {
			log.Printf("error accepting connection: %s\n", err)
			continue
		}

		go s.handleConn(c)
	}
}

func parseRequest(c net.Conn) (*Request, error) {
	// TODO: make this more efficient since this is going too nicely with the name otherwise

	// Basically the header and content are separated by \r\n\r\n and if the request is valid
	// it should be the first instance.
	sepIndex := bytes.Index()
}

func (s *Server) handleConn(c net.Conn) error {
	// TODO: Construct Request object aka parse the request
	// TODO: Decide which route should handle given request
	// TODO: Execute the handler func for the given request and write the response
	return nil
}
