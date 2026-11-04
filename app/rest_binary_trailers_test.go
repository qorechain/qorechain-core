package app

import (
	"bufio"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// A handler that behaves like the gateway on a refused simulation: it
// announces a text and a binary trailer and sets both after the body.
func gatewayLike(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Add("Trailer", "Grpc-Trailer-Grpc-Status-Details-Bin")
	w.Header().Add("Trailer", "Grpc-Trailer-Keep")
	w.WriteHeader(http.StatusInternalServerError)
	_, _ = w.Write([]byte(`{"code":2,"message":"tx parse error","details":[]}`))
	w.Header().Set("Grpc-Trailer-Grpc-Status-Details-Bin", "\x08\x02\x12\xa1\x01raw")
	w.Header().Set("Grpc-Trailer-Keep", "ok")
}

func TestDropBinaryTrailers(t *testing.T) {
	srv := httptest.NewServer(dropBinaryTrailers(http.HandlerFunc(gatewayLike)))
	defer srv.Close()

	// Read the raw HTTP/1.1 response: the binary trailer must not be on the wire.
	conn, err := net.Dial("tcp", strings.TrimPrefix(srv.URL, "http://"))
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	_, _ = conn.Write([]byte("POST / HTTP/1.1\r\nHost: x\r\nContent-Length: 0\r\nConnection: close\r\n\r\n"))
	var raw strings.Builder
	sc := bufio.NewScanner(conn)
	for sc.Scan() {
		raw.WriteString(sc.Text() + "\n")
	}
	got := raw.String()
	if strings.Contains(strings.ToLower(got), "status-details-bin") {
		t.Fatalf("binary trailer still sent:\n%q", got)
	}
	if !strings.Contains(got, "Grpc-Trailer-Keep: ok") {
		t.Fatalf("text trailer lost:\n%q", got)
	}
	if !strings.Contains(got, `"message":"tx parse error"`) || !strings.HasPrefix(got, "HTTP/1.1 500") {
		t.Fatalf("status or body changed:\n%q", got)
	}

	// And a strict client parses it.
	resp, err := http.Post(srv.URL, "application/json", nil)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != 500 {
		t.Fatalf("status %d", resp.StatusCode)
	}
}
