// Copyright 2016 Afshin Darian. All rights reserved.
// Use of this source code is governed by The MIT License
// that can be found in the LICENSE file.

package sleuth

import "sync"

type notifier struct {
	*sync.Mutex
	active bool
	stream chan struct{}
}

func (n *notifier) activate() {
	n.Lock()
	defer n.Unlock()
	n.active = true
}

func (n *notifier) deactivate() {
	n.Lock()
	defer n.Unlock()
	n.active = false
}

func (n *notifier) notify() {
	n.Lock()
	defer n.Unlock()
	if n.active {
		n.stream <- struct{}{}
	}
}
