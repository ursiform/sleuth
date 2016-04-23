// Copyright 2016 Afshin Darian. All rights reserved.
// Use of this source code is governed by The MIT License
// that can be found in the LICENSE file.

package sleuth

import (
	"net/http"

	"github.com/zeromq/gyre"
)

type responseWriter struct {
	http.ResponseWriter
	node   *gyre.Gyre
	output *response
	peer   string
}

// Header returns the header map that will be sent by
// WriteHeader. Changing the header after a call to
// WriteHeader (or Write) has no effect unless the modified
// headers were declared as trailers by setting the
// "Trailer" header before the call to WriteHeader (see example).
// To suppress implicit response headers, set their value to nil.
func (res *responseWriter) Header() http.Header {
	return res.output.Header
}

// Write writes the data to the connection as part of an HTTP reply.
// If WriteHeader has not yet been called, Write calls
// WriteHeader(http.StatusOK) before writing the data.
//
// If the Header does not contain a Content-Type line, Write adds a Content-Type
// set to the result of passing the initial 512 bytes of written data to
// DetectContentType.
func (res *responseWriter) Write(data []byte) (int, error) {
	if res.output.Code == 0 {
		res.WriteHeader(http.StatusOK)
	}
	header := res.Header()
	if len(header.Get("Content-Type")) == 0 {
		header.Add("Content-Type", http.DetectContentType(data[:512]))
	}
	res.output.Body = data
	payload, err := marshalResponse(res.output)
	if err != nil {
		return 0, err
	}
	err = res.node.Whisper(res.peer, payload)
	if err != nil {
		return 0, err
	} else {
		return len(data), nil
	}
}

// WriteHeader sends an HTTP response header with status code.
// If WriteHeader is not called explicitly, the first call to Write
// will trigger an implicit WriteHeader(http.StatusOK).
// Thus explicit calls to WriteHeader are mainly used to
// send error codes.
func (res *responseWriter) WriteHeader(code int) {
	res.output.Code = code
}

func newResponseWriter(node *gyre.Gyre, dest *destination) *responseWriter {
	res := new(responseWriter)
	res.node = node
	res.output = new(response)
	res.output.Handle = dest.handle
	res.output.Header = http.Header(make(map[string][]string))
	res.peer = dest.node
	return res
}
