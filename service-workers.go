// Copyright 2016 Afshin Darian. All rights reserved.
// Use of this source code is governed by The MIT License
// that can be found in the LICENSE file.

package sleuth

import "sync"

type worker struct {
	// name is the short public name attached to all events.
	name string
	// node is the full peer node name used for whispering.
	node string
	// version is the optional service version running on a node.
	version string
}

type serviceWorkers struct {
	*sync.Mutex
	current int
	list    []*worker
}

func (workers *serviceWorkers) add(name, node, version string) int {
	workers.Mutex.Lock()
	defer workers.Mutex.Unlock()
	for _, service := range workers.list {
		if service.name == name {
			return len(workers.list)
		}
	}
	workers.list = append(workers.list, &worker{
		name:    name,
		node:    node,
		version: version})
	return len(workers.list)
}

func (workers *serviceWorkers) next() *worker {
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

func (workers *serviceWorkers) remove(name string) int {
	workers.Mutex.Lock()
	defer workers.Mutex.Unlock()
	for index, service := range workers.list {
		if service.name == name {
			list := workers.list
			workers.list = append(list[0:index], list[index+1:len(list)]...)
			break
		}
	}
	return len(workers.list)
}

func newWorkers() *serviceWorkers {
	workers := &serviceWorkers{}
	workers.Mutex = new(sync.Mutex)
	workers.list = make([]*worker, 0)
	return workers
}
