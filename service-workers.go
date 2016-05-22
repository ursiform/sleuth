// Copyright 2016 Afshin Darian. All rights reserved.
// Use of this source code is governed by The MIT License
// that can be found in the LICENSE file.

package sleuth

import "sync"

type serviceWorkers struct {
	*sync.Mutex
	current int
	list    []*peer
}

func (w *serviceWorkers) add(p *peer) int {
	w.Mutex.Lock()
	defer w.Mutex.Unlock()
	for _, service := range w.list {
		if service.Name == p.Name {
			return len(w.list)
		}
	}
	w.list = append(w.list, p)
	return len(w.list)
}

func (w *serviceWorkers) available() bool {
	w.Mutex.Lock()
	defer w.Mutex.Unlock()
	return len(w.list) > 0
}

func (w *serviceWorkers) next() *peer {
	w.Mutex.Lock()
	defer w.Mutex.Unlock()
	length := len(w.list)
	current := w.current
	if length == 0 {
		return nil
	}
	if current < length {
		w.current++
		return w.list[current]
	}
	w.current = 1
	return w.list[0]
}

func (w *serviceWorkers) remove(name string) (int, *peer) {
	w.Mutex.Lock()
	defer w.Mutex.Unlock()
	for i, p := range w.list {
		if p.Name == name {
			list := w.list
			w.list = append(list[0:i], list[i+1:len(list)]...)
			return len(w.list), p
		}
	}
	return len(w.list), nil
}

func newWorkers() *serviceWorkers {
	w := &serviceWorkers{}
	w.Mutex = new(sync.Mutex)
	w.list = make([]*peer, 0)
	return w
}
