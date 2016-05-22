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

func (r *responseWriter) Header() http.Header {
	return r.output.Header
}

func (r *responseWriter) Write(data []byte) (int, error) {
	if r.output.Code == 0 {
		r.WriteHeader(http.StatusOK)
	}
	header := r.Header()
	if len(header.Get("Content-Type")) == 0 {
		header.Add("Content-Type", http.DetectContentType(data[:512]))
	}
	r.output.Body = data
	payload := marshalResponse(r.output)
	if err := r.node.Whisper(r.peer, payload); err != nil {
		return 0, err
	} else {
		return len(data), nil
	}
}

func (r *responseWriter) WriteHeader(code int) {
	r.output.Code = code
}

func newResponseWriter(node *gyre.Gyre, dest *destination) *responseWriter {
	r := new(responseWriter)
	r.node = node
	r.output = new(response)
	r.output.Handle = dest.handle
	r.output.Header = http.Header(make(map[string][]string))
	r.peer = dest.node
	return r
}
