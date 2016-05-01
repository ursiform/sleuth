// Copyright 2016 Afshin Darian. All rights reserved.
// Use of this source code is governed by The MIT License
// that can be found in the LICENSE file.

package sleuth

import "sync"

type serviceWorkers struct {
	*sync.Mutex
	current int
	list    []*Peer
}

func (workers *serviceWorkers) add(peer *Peer) int {
	workers.Mutex.Lock()
	defer workers.Mutex.Unlock()
	for _, service := range workers.list {
		if service.Name == peer.Name {
			return len(workers.list)
		}
	}
	workers.list = append(workers.list, peer)
	return len(workers.list)
}

func (workers *serviceWorkers) available() bool {
	workers.Mutex.Lock()
	defer workers.Mutex.Unlock()
	return len(workers.list) > 0
}

func (workers *serviceWorkers) next() *Peer {
	workers.Mutex.Lock()
	defer workers.Mutex.Unlock()
	length := len(workers.list)
	current := workers.current
	if length == 0 {
		return nil
	}
	if current < length {
		workers.current++
		return workers.list[current]
	}
	workers.current = 1
	return workers.list[0]
}

func (workers *serviceWorkers) remove(name string) (int, *Peer) {
	workers.Mutex.Lock()
	defer workers.Mutex.Unlock()
	for index, peer := range workers.list {
		if peer.Name == name {
			list := workers.list
			workers.list = append(list[0:index], list[index+1:len(list)]...)
			return len(workers.list), peer
		}
	}
	return len(workers.list), nil
}

func newWorkers() *serviceWorkers {
	workers := &serviceWorkers{}
	workers.Mutex = new(sync.Mutex)
	workers.list = make([]*Peer, 0)
	return workers
}
