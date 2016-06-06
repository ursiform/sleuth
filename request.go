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
	handle int64
	node   string
}

type request struct {
	Body     []byte              `json:"body,omitempty"`
	Handle   int64               `json:"handle"`
	Header   map[string][]string `json:"header"`
	Method   string              `json:"method"`
	Receiver string              `json:"receiver"`
	URL      string              `json:"url"`
}

func marshalRequest(receiver string, handle int64, in *http.Request) ([]byte, error) {
	out := new(request)
	if in.Body != nil {
		if body, err := ioutil.ReadAll(in.Body); err == nil {
			out.Body = body
		}
	}
	out.Header = map[string][]string(in.Header)
	out.Method = in.Method
	// Scheme and Host are used by sleuth for routing, but should not be sent.
	in.URL.Scheme = ""
	in.URL.Host = ""
	out.URL = in.URL.String()
	out.Receiver = receiver
	out.Handle = handle
	marshalled, err := json.Marshal(out)
	if err != nil {
		return nil, newError(errReqMarshal, err.Error())
	}
	return append([]byte(group+repl), zip(marshalled)...), nil
}

func unmarshalRequest(payload []byte) (*destination, *http.Request, error) {
	unzipped, err := unzip(payload)
	if err != nil {
		return nil, nil, err.(*Error).escalate(errReqUnmarshal)
	}
	in := new(request)
	if err = json.Unmarshal(unzipped, in); err != nil {
		return nil, nil, newError(errReqUnmarshalJSON, err.Error())
	}
	out, err := http.NewRequest(in.Method, in.URL, bytes.NewBuffer(in.Body))
	if err != nil {
		return nil, nil, newError(errReqUnmarshalHTTP, err.Error())
	}
	out.Header = http.Header(in.Header)
	dest := new(destination)
	dest.handle = in.Handle
	dest.node = in.Receiver
	return dest, out, nil
}
