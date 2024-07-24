package stupidhttp

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"log"
	"net"
	"strconv"
	"strings"
)

type MethodType int

type HandleFunc func(*Request) (*Response, error)

var (
	NotFoundResponse = &Response{
		StatusCode: 404,
		Status:     "Not Found",
		ProtoMajor: 1,
		ProtoMinor: 1,
		Headers:    map[string]string{"Content-Type": "text/plain"},
		Body:       bytes.NewBufferString("404 Not Found"),
	}
)

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
	Status     string
	Body       io.ReadCloser
	ProtoMajor int
	ProtoMinor int
}

type Route struct {
	Method     MethodType
	HandleFunc func(*Request) (*Response, error)
}

type Config struct {
	MaxHeaderSize int
	Address       string
}

type Server struct {
	Routes map[string]Route
	Config Config
}

func (s *Server) AddHandler(path string, method MethodType, handlerFunc HandleFunc) {
	s.Routes[path] = Route{
		Method:     method,
		HandleFunc: handlerFunc,
	}
}

func NewServer(config Config) *Server {
	return &Server{
		Routes: make(map[string]Route),
		Config: config,
	}
}

func (s *Server) Start() error {
	l, err := net.Listen("tcp", s.Config.Address)
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

func (s *Server) writeResponse(c net.Conn, resp *Response) error {
	reqLine := fmt.Sprintf("HTTP/%d.%d %d %s\r\n", resp.ProtoMajor, resp.ProtoMinor, resp.StatusCode, resp.Status)

	_, err := c.Write([]byte(reqLine))
	if err != nil {
		return err
	}

	for key, val := range resp.Headers {
		_, err = c.Write([]byte(fmt.Sprintf("%s: %s\r\n", key, val)))
		if err != nil {
			return err
		}
	}

	// signal end of headers
	_, err = c.Write([]byte("\r\n"))
	if err != nil {
		return err
	}

	if resp.Body != nil {
		_, err = io.Copy(c, resp.Body)
	}
	return err
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
	defer request.Body.Close()

	// find the correct method

	return nil
}
