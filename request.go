// Copyright 2016 Afshin Darian. All rights reserved.
// Use of this source code is governed by The MIT License
// that can be found in the LICENSE file.

package sleuth

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"net/http"
)

type destination struct {
	handle string
	node   string
}

type request struct {
	Body     []byte              `json:"a,omitempty"`
	Handle   string              `json:"b"`
	Header   map[string][]string `json:"c"`
	Method   string              `json:"d"`
	Receiver string              `json:"e"`
	URL      string              `json:"f"`
}

func marshalRequest(receiver, handle string, in *http.Request) ([]byte, error) {
	out := new(request)
	if in.Body != nil {
		if body, err := ioutil.ReadAll(in.Body); err == nil {
			out.Body = body
		}
	}
	out.Header = map[string][]string(in.Header)
	out.Method = in.Method
	out.URL = in.URL.String()
	out.Receiver = receiver
	out.Handle = handle
	if marshalled, err := json.Marshal(out); err != nil {
		return nil, err
	} else {
		return append([]byte(group+repl), marshalled...), nil
	}
}

func unmarshalRequest(payload []byte) (*destination, *http.Request, error) {
	in := new(request)
	if err := json.Unmarshal(payload, in); err != nil {
		return nil, nil, err
	}
	out, err := http.NewRequest(in.Method, in.URL, bytes.NewBuffer(in.Body))
	if err != nil {
		return nil, nil, err
	}
	out.Header = http.Header(in.Header)
	dest := new(destination)
	dest.handle = in.Handle
	dest.node = in.Receiver
	return dest, out, nil
}
