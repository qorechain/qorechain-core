package app

import (
	"net/http"
	"strings"
)

// dropBinaryTrailers keeps the REST gateway from sending gRPC binary metadata
// as HTTP trailers. The gateway forwards every gRPC trailer, and a refused
// simulation carries its status details (simulate_refusal.go) in
// grpc-status-details-bin, which the gateway writes raw as
// Grpc-Trailer-Grpc-Status-Details-Bin. Raw bytes are not a valid header
// value: Node's fetch drops the connection and Cloudflare answers 502/520,
// while curl, which ignores trailers, saw a normal 500. The same details are
// already in the JSON body (details[]), so nothing is lost.
//
// net/http only sends a trailer that was announced in the Trailer header when
// the status was written, so removing the announcement drops the trailer.
// Node-local: REST only.
func dropBinaryTrailers(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		next.ServeHTTP(&binaryTrailerFilter{ResponseWriter: w}, r)
	})
}

type binaryTrailerFilter struct {
	http.ResponseWriter
	wroteHeader bool
}

func (f *binaryTrailerFilter) filter() {
	if f.wroteHeader {
		return
	}
	f.wroteHeader = true
	h := f.ResponseWriter.Header()
	announced := h.Values("Trailer")
	if len(announced) == 0 {
		return
	}
	var keep []string
	for _, v := range announced {
		for _, name := range strings.Split(v, ",") {
			name = strings.TrimSpace(name)
			if name == "" || strings.HasSuffix(strings.ToLower(name), "-bin") {
				continue
			}
			keep = append(keep, name)
		}
	}
	h.Del("Trailer")
	for _, name := range keep {
		h.Add("Trailer", name)
	}
}

func (f *binaryTrailerFilter) WriteHeader(code int) {
	f.filter()
	f.ResponseWriter.WriteHeader(code)
}

func (f *binaryTrailerFilter) Write(b []byte) (int, error) {
	f.filter()
	return f.ResponseWriter.Write(b)
}

func (f *binaryTrailerFilter) Flush() {
	f.filter()
	if fl, ok := f.ResponseWriter.(http.Flusher); ok {
		fl.Flush()
	}
}
