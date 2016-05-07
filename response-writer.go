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

func (res *responseWriter) Header() http.Header {
	return res.output.Header
}

func (res *responseWriter) Write(data []byte) (int, error) {
	if res.output.Code == 0 {
		res.WriteHeader(http.StatusOK)
	}
	header := res.Header()
	if len(header.Get("Content-Type")) == 0 {
		header.Add("Content-Type", http.DetectContentType(data[:512]))
	}
	res.output.Body = data
	if payload, err := marshalResponse(res.output); err != nil {
		return 0, err
	} else if err := res.node.Whisper(res.peer, payload); err != nil {
		return 0, err
	} else {
		return len(data), nil
	}
}

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
