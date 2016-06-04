// Copyright 2016 Afshin Darian. All rights reserved.
// Use of this source code is governed by The MIT License
// that can be found in the LICENSE file.

package sleuth

import (
	"net/http"

	"github.com/zeromq/gyre"
)

type writer struct {
	http.ResponseWriter
	node   *gyre.Gyre
	output *response
	peer   string
}

func (w *writer) Header() http.Header {
	return w.output.Header
}

func (w *writer) Write(data []byte) (int, error) {
	if w.output.Code == 0 {
		w.WriteHeader(http.StatusOK)
	}
	header := w.Header()
	if len(header.Get("Content-Type")) == 0 {
		header.Add("Content-Type", http.DetectContentType(data))
	}
	w.output.Body = data
	payload := marshalResponse(w.output)
	if err := w.node.Whisper(w.peer, payload); err != nil {
		return 0, newError(errResWhisper, err.Error())
	}
	return len(data), nil
}

func (w *writer) WriteHeader(code int) {
	w.output.Code = code
}

func newResponseWriter(node *gyre.Gyre, dest *destination) *writer {
	return &writer{
		node: node,
		output: &response{
			Handle: dest.handle,
			Header: http.Header(make(map[string][]string))},
		peer: dest.node}
}
