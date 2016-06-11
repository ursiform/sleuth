// Copyright 2016 Afshin Darian. All rights reserved.
// Use of this source code is governed by The MIT License
// that can be found in the LICENSE file.

package sleuth

// destination describes the group, node, and specific handle of a message.
type destination struct {
	group  string
	handle string
	node   string
}
