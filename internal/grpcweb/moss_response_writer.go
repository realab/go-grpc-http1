package grpcweb

import (
	"bytes"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/realab/go-grpc-http1/internal/sliceutils"
)

const (
	noWritten     = -1
	defaultStatus = http.StatusOK
)

type mossResponseWriter struct {
	w http.ResponseWriter

	size    int
	status  int
	headers http.Header
	body    bytes.Buffer
	startAt time.Time

	// List of trailers that were announced via the `Trailer` header at the time headers were written. Also used to keep
	// track of whether headers were already written (in which case this is non-nil, even if it is the empty slice).
	announcedTrailers []string
}

// NewMossResponseWriter returns a response writer that transparently transcodes an gRPC HTTP/2 response to a gRPC-Web
// response. It can be used as the response writer in the `ServeHTTP` method of a `grpc.Server`.
// The second return value is a finalization function that takes care of sending the data frame with trailers. It
// *needs* to be called before the response handler exits successfully (the returned error is simply any error of the
// underlying response writer passed through).
func NewMossResponseWriter(w http.ResponseWriter) (http.ResponseWriter, func() error) {
	rw := &mossResponseWriter{
		w:       w,
		size:    noWritten,
		status:  defaultStatus,
		startAt: time.Now(),
	}
	return rw, rw.Finalize
}

func (w *mossResponseWriter) Written() bool {
	return w.size != noWritten
}

// Header returns the HTTP Header of the underlying response writer.
func (w *mossResponseWriter) Header() http.Header {
	if w.headers == nil {
		w.headers = w.w.Header()
	}
	return w.headers
}

// Flush flushes any data not yet written. In contrast to most `http.ResponseWriter` implementations, it does not send
// headers if no data has been written yet.
func (w *mossResponseWriter) Flush() {
	hdr := w.w.Header()
	for k, vs := range w.headers {
		hdr[k] = vs
	}
}

// prepareHeadersIfNecessary is called internally on any action that might cause headers to be sent.
func (w *mossResponseWriter) prepareHeadersIfNecessary() {
	if w.announcedTrailers != nil {
		return
	}

	hdr := w.Header()
	w.announcedTrailers = sliceutils.StringClone(hdr["Trailer"])
	// Trailers are sent in a data frame, so don't announce trailers as otherwise downstream proxies might get confused.
	hdr.Del("Trailer")

	for k, vs := range hdr {
		if !strings.HasPrefix(k, http.TrailerPrefix) {
			continue
		}
		trailerName := http.CanonicalHeaderKey(k[len(http.TrailerPrefix):])
		hdr[trailerName] = vs
		delete(hdr, k)
	}

	// Any content length that might be set is no longer accurate because of trailers.
	hdr.Del("Content-Length")
}

func (w *mossResponseWriter) WriteHeaderNow() {
	if !w.Written() {
		w.size = 0
	}
}

// WriteHeader sends HTTP headers to the client, along with the given status code.
func (w *mossResponseWriter) WriteHeader(statusCode int) {
	if statusCode > 0 && w.status != statusCode {
		if w.Written() {
			fmt.Fprintf(os.Stderr, "[Moss-Writer][WARNING] Headers were already written. Wanted to override status code %d with %d", w.status, statusCode)
		}
		w.status = statusCode
	}
	w.status = statusCode
}

// Write writes a chunk of data.
func (w *mossResponseWriter) Write(buf []byte) (int, error) {
	w.WriteHeaderNow()
	n, err := w.body.Write(buf)
	w.size += n
	return n, err
}

func (w *mossResponseWriter) Finalize() error {
	w.prepareHeadersIfNecessary()
	w.w.WriteHeader(w.status)
	if _, err := w.w.Write(w.body.Bytes()); err != nil {
		return err
	}
	return nil
}
