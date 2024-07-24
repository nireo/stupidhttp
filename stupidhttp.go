package stupidhttp

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"net"
	"strconv"
	"strings"
)

type MethodType int

type HandleFunc func(*Request) (*Response, error)

const (
	MethodGet MethodType = iota
	MethodPost
	MethodPut
	MethodDelete
	MethodUnrecognized
)

type Request struct {
	Headers    map[string]string
	Body       io.ReadCloser
	Method     MethodType
	ProtoMajor int
	ProtoMinor int
	Path       string
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

type Config struct {
	MaxHeaderSize int
}

type Server struct {
	Routes  map[string]Route
	Address string
	Config  Config
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

func strToMethodType(method string) MethodType {
	switch method {
	case "GET":
		return MethodGet
	case "POST":
		return MethodGet
	case "DELETE":
		return MethodGet
	case "PUT":
		return MethodGet
	default:
		return MethodUnrecognized
	}
}

func (s *Server) parseRequest(c net.Conn) (*Request, error) {
	// TODO: make this more efficient since this is going too nicely with the name otherwise
	reader := bufio.NewReader(c)

	reqLine, err := reader.ReadString('\n')
	if err != nil {
		return nil, fmt.Errorf("error reading request line: %w", err)
	}
	reqLine = strings.TrimSpace(reqLine)
	splitted := strings.Split(reqLine, " ")
	if len(splitted) != 3 {
		return nil, fmt.Errorf("invalid request line")
	}

	major, minor, err := parseHTTPProto(splitted[2])
	if err != nil {
		return nil, fmt.Errorf("invalid proto: %w", err)
	}

	method := strToMethodType(splitted[0])
	if method == MethodUnrecognized {
		return nil, fmt.Errorf("invalid method: %w", err)
	}

	headers := make(map[string]string)
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			return nil, fmt.Errorf("error reading header: %w", err)
		}

		line = strings.TrimSpace(line)
		// the headers end and content starts
		if line == "" {
			break
		}

		splitted := strings.SplitN(line, ":", 2)
		if len(splitted) != 2 {
			return nil, fmt.Errorf("invalid header: %s", line)
		}

		headers[strings.TrimSpace(splitted[0])] = strings.TrimSpace(splitted[1])
	}

	var body io.ReadCloser
	if clen, ok := headers["Content-Length"]; ok {
		l, err := strconv.Atoi(clen)
		if err != nil {
			return nil, fmt.Errorf("invalid content-length: %w", err)
		}

		body = io.NopCloser(io.LimitReader(reader, int64(l)))
	} else {
		body = io.NopCloser(reader)
	}

	return &Request{
		Body:       body,
		Headers:    headers,
		Method:     method,
		ProtoMajor: major,
		ProtoMinor: minor,
	}, nil
}

// parseHTTPProto also validates that the protocol is valid
func parseHTTPProto(proto string) (int, int, error) {
	if !strings.HasPrefix(proto, "HTTP/") {
		return 0, 0, fmt.Errorf("unsupported protocol")
	}

	version := strings.TrimPrefix(proto, "HTTP/")
	parts := strings.Split(version, ".")
	if len(parts) != 2 {
		return 0, 0, fmt.Errorf("invalid http version format")
	}

	major, err := strconv.Atoi(parts[0])
	if err != nil {
		return 0, 0, fmt.Errorf("invalid major version: %w", err)
	}

	minor, err := strconv.Atoi(parts[1])
	if err != nil {
		return 0, 0, fmt.Errorf("invalid minor version: %w", err)
	}
	return major, minor, nil
}

func (s *Server) returnResponse(resp *Response) error {
	return nil
}

func (s *Server) handleConn(c net.Conn) error {
	// TODO: Construct Request object aka parse the request
	// TODO: Decide which route should handle given request
	// TODO: Execute the handler func for the given request and write the response

	defer c.Close()

	request, err := s.parseRequest(c)
	if err != nil {
		// TODO: return p
		return err
	}

	return nil
}
