// Copyright 2016 Afshin Darian. All rights reserved.
// Use of this source code is governed by The MIT License
// that can be found in the LICENSE file.

package sleuth

import (
	"bytes"
	"compress/gzip"
	"io/ioutil"
)

func zip(in []byte) []byte {
	out := new(bytes.Buffer)
	writer := gzip.NewWriter(out)
	writer.Write(in)
	writer.Close()
	return out.Bytes()
}

func unzip(in []byte) ([]byte, error) {
	var out []byte
	reader, err := gzip.NewReader(bytes.NewBuffer(in))
	if err != nil {
		return out, newError(errUnzip, err.Error())
	}
	reader.Close()
	out, err = ioutil.ReadAll(reader)
	if err != nil {
		return out, newError(errUnzipRead, err.Error())
	}
	return out, nil
}
