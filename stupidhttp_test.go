package stupidhttp

import (
	"io"
	"net"
	"strings"
	"testing"
	"time"
)

func TestParseHTTPProto(t *testing.T) {
	tests := []struct {
		input         string
		expectedMajor int
		expectedMinor int
		shouldErr     bool
	}{
		{"HTTP/1.1", 1, 1, false},
		{"HTTP/2.0", 2, 0, false},
		{"HTTP/1.0", 1, 0, false},
		{"HTTP/invalid", 0, 0, true},
		{"HTTPS/1.1", 0, 0, true},
	}

	for _, test := range tests {
		major, minor, err := parseHTTPProto(test.input)
		if test.shouldErr {
			if err == nil {
				t.Errorf("parseHTTPProto(%s) expected error, got nil", test.input)
			}
		} else {
			if err != nil {
				t.Errorf("parseHTTPProto(%s) unexpected error: %v", test.input, err)
			}
			if major != test.expectedMajor || minor != test.expectedMinor {
				t.Errorf("parseHTTPProto(%s) = (%d, %d), expected (%d, %d)", test.input, major, minor, test.expectedMajor, test.expectedMinor)
			}
		}
	}
}

func TestParseRequest(t *testing.T) {
	server := &Server{}
	testRequest := `GET /test HTTP/1.1
Host: example.com
Content-Type: text/plain
Content-Length: 13

Hello, World!`

	mockConn := &mockConn{Reader: strings.NewReader(testRequest)}
	req, err := server.parseRequest(mockConn)

	if err != nil {
		t.Fatalf("parseRequest() unexpected error: %v", err)
	}

	if req.Method != MethodGet {
		t.Errorf("Expected method GET, got %v", req.Method)
	}

	if req.Path != "/test" {
		t.Errorf("Expected path /test, got %s", req.Path)
	}

	if req.ProtoMajor != 1 || req.ProtoMinor != 1 {
		t.Errorf("Expected HTTP/1.1, got HTTP/%d.%d", req.ProtoMajor, req.ProtoMinor)
	}

	if req.Headers["Host"] != "example.com" {
		t.Errorf("Expected Host header example.com, got %s", req.Headers["Host"])
	}

	body, _ := io.ReadAll(req.Body)
	if string(body) != "Hello, World!" {
		t.Errorf("Expected body 'Hello, World!', got '%s'", string(body))
	}
}

type mockConn struct {
	Reader io.Reader
	Writer io.Writer
}

func (m *mockConn) Read(b []byte) (n int, err error)   { return m.Reader.Read(b) }
func (m *mockConn) Write(b []byte) (n int, err error)  { return m.Writer.Write(b) }
func (m *mockConn) Close() error                       { return nil }
func (m *mockConn) LocalAddr() net.Addr                { return nil }
func (m *mockConn) RemoteAddr() net.Addr               { return nil }
func (m *mockConn) SetDeadline(t time.Time) error      { return nil }
func (m *mockConn) SetReadDeadline(t time.Time) error  { return nil }
func (m *mockConn) SetWriteDeadline(t time.Time) error { return nil }
