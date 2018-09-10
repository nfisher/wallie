package reqlog

import (
	"log"
	"net/http"
	"time"
)

func LogRequests(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		wr := &ResponseWriter{ResponseWriter: w}
		start := time.Now()
		h.ServeHTTP(wr, req)
		log.Printf(`%v %s %s - %v - %v - %vB`, wr.Status(), req.Method, req.URL.Path, req.RemoteAddr, time.Now().Sub(start), wr.Bytes())
	})
}

type ResponseWriter struct {
	http.ResponseWriter
	bytes       int
	status      int
	wroteHeader bool
}

func (w *ResponseWriter) Status() int {
	return w.status
}

func (w *ResponseWriter) Bytes() int {
	return w.bytes
}

func (w *ResponseWriter) Write(p []byte) (n int, err error) {
	if !w.wroteHeader {
		w.WriteHeader(http.StatusOK)
	}
	w.bytes += len(p)

	return w.ResponseWriter.Write(p)
}

func (w *ResponseWriter) WriteHeader(code int) {
	w.ResponseWriter.WriteHeader(code)
	// Check after in case there's error handling in the wrapped ResponseWriter.
	if w.wroteHeader {
		return
	}
	w.status = code
	w.wroteHeader = true
}
