package httpd

import (
	"errors"
	"io"
	"net"
	"net/http"
	"testing"
	"time"
)

// Een client die zijn request-headers nooit afmaakt, moet de verbinding kwijtraken.
// Zonder ReadHeaderTimeout hield de server ze voor altijd open, elk met een goroutine
// en een file descriptor.
func TestStalledHeaderIsClosed(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	srv := newServer(ln.Addr().String(), http.NotFoundHandler())
	srv.ReadHeaderTimeout = 200 * time.Millisecond
	go srv.Serve(ln)
	defer srv.Close()

	c, err := net.Dial("tcp", ln.Addr().String())
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()
	if _, err := c.Write([]byte("GET /api/summary HTTP/1.1\r\nHost: x\r\n")); err != nil {
		t.Fatal(err)
	}
	c.SetReadDeadline(time.Now().Add(3 * time.Second))
	_, err = io.ReadAll(c)
	var ne net.Error
	if errors.As(err, &ne) && ne.Timeout() {
		t.Fatal("server hield de verbinding met onvolledige headers open")
	}
}

func TestNewServerSetsTimeouts(t *testing.T) {
	srv := newServer("127.0.0.1:0", http.NotFoundHandler())
	if srv.ReadHeaderTimeout == 0 || srv.ReadTimeout == 0 || srv.IdleTimeout == 0 {
		t.Fatalf("timeouts niet gezet: header %v read %v idle %v", srv.ReadHeaderTimeout, srv.ReadTimeout, srv.IdleTimeout)
	}
}
