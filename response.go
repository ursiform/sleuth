// Copyright 2016 Afshin Darian. All rights reserved.
// Use of this source code is governed by The MIT License
// that can be found in the LICENSE file.

package sleuth

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
)

type response struct {
	Body   []byte      `json:"body"`
	Code   int         `json:"code"`
	Handle string      `json:"handle"`
	Header http.Header `json:"header"`
}

type body struct {
	io.Reader
}

func (*body) Close() error { return nil }

func marshalResponse(res *response) []byte {
	// This will never fail to marshal, so error can be ignored.
	marshalled, _ := json.Marshal(res)
	return append([]byte(group+recv), marshalled...)
}

func unmarshalResponse(payload []byte) (string, *http.Response, error) {
	var handle string
	var res *http.Response
	in := new(response)
	in.Header = http.Header(make(map[string][]string))
	if err := json.Unmarshal(payload, in); err != nil {
		return handle, res, err
	}
	handle = in.Handle
	res = new(http.Response)
	res.Body = &body{bytes.NewBuffer(in.Body)}
	res.ContentLength = int64(len(in.Body))
	res.Header = in.Header
	res.StatusCode = in.Code
	res.Status = http.StatusText(in.Code)
	return handle, res, nil
}
