package stupidhttp

import (
	"bufio"
	"crypto/tls"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"strconv"
	"strings"
)

// MethodType signifies the HTTP method of the request
type MethodType int

// HandleFunc is a function that the user writes to handle a given request for a path
type HandleFunc func(*Request) *Response

var (
	NotFoundResponse = &Response{
		StatusCode: 404,
		Status:     "Not Found",
		ProtoMajor: 1,
		ProtoMinor: 1,
		Headers:    map[string]string{"Content-Type": "text/plain"},
		Body:       io.NopCloser(strings.NewReader("Not Found")),
	}
	BadRequestResponse = &Response{
		StatusCode: 400,
		Status:     "Bad Request",
		ProtoMajor: 1,
		ProtoMinor: 1,
		Headers:    map[string]string{"Content-Type": "text/plain"},
		Body:       io.NopCloser(strings.NewReader("Bad Request")),
	}
)

const (
	MethodGet MethodType = iota
	MethodPost
	MethodPut
	MethodDelete
	MethodUnrecognized
)

// Request is the result that is parsed from the HTTP request.
type Request struct {
	Headers    map[string]string
	Body       io.ReadCloser
	Method     MethodType
	ProtoMajor int
	ProtoMinor int
	Path       string
}

// Response is used to write the correct data to a given response.
type Response struct {
	Headers    map[string]string
	StatusCode int
	Status     string
	Body       io.ReadCloser
	ProtoMajor int
	ProtoMinor int
}

func (r *Response) SetRedirect(path string, statusCode int) {
	r.Status = http.StatusText(statusCode)
	r.StatusCode = statusCode
	r.Headers["Location"] = path
}

func (r *Response) SetHeader(key, value string) {
	if r.Headers == nil {
		r.Headers = make(map[string]string)
	}

	r.Headers[key] = value
}

type Route struct {
	HandleFunc HandleFunc
}

type Config struct {
	MaxHeaderSize int
	Address       string
	TLSCertFile   string
	TLSKeyFile    string
}

type Server struct {
	routes    map[string]Route
	Config    Config
	tlsConfig *tls.Config
}

func (s *Server) AddHandler(path string, handlerFunc HandleFunc) {
	s.routes[path] = Route{
		HandleFunc: handlerFunc,
	}
}

func NewServer(config Config) (*Server, error) {
	server := &Server{
		routes: make(map[string]Route),
		Config: config,
	}

	if config.TLSCertFile != "" && config.TLSKeyFile != "" {
		cert, err := tls.LoadX509KeyPair(config.TLSCertFile, config.TLSKeyFile)
		if err != nil {
			return nil, fmt.Errorf("failed to load tls cert and key: %w", err)
		}

		server.tlsConfig = &tls.Config{
			Certificates: []tls.Certificate{cert},
		}
	}

	return server, nil
}

func (s *Server) Start() error {
	var l net.Listener
	var err error

	if s.tlsConfig != nil {
		l, err = tls.Listen("tcp", s.Config.Address, s.tlsConfig)
	} else {
		l, err = net.Listen("tcp", s.Config.Address)
	}
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
		Path:       splitted[1],
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
	defer c.Close()

	request, err := s.parseRequest(c)
	if err != nil {
		return s.writeResponse(c, BadRequestResponse)
	}
	defer request.Body.Close()

	method, ok := s.routes[request.Path]
	if !ok {
		return s.writeResponse(c, NotFoundResponse)
	}

	response := method.HandleFunc(request)
	return s.writeResponse(c, response)
}

// func cleanPath(path string) string {
// 	if path == "" {
// 		return "/"
// 	}

// 	pathSize := len(path)
// 	buffer := make([]byte, 0, 128) // buffer to store the result for efficiency reasons do it like this
// 	r := 1                         // read index
// 	w := 1                         // write index

// 	if path[0] != '/' {
// 		r = 0

// 		if pathSize+1 > 128 {
// 			buffer = make([]byte, pathSize+1)
// 		} else {
// 			buffer = buffer[:pathSize+1]
// 		}

// 		buffer[0] = '/'
// 	}

// 	hasTrailingSlash := pathSize > 1 && path[pathSize-1] == '/'
// 	for r < pathSize { // while the read pointer is valid
// 		switch {
// 		case path[r] == '/':
// 			r++
// 		case path[r] == '/' && r+1 == pathSize:
// 			hasTrailingSlash = true
// 			r++
// 	}
// 	return ""
// }
