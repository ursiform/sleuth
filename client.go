// Copyright 2016 Afshin Darian. All rights reserved.
// Use of this source code is governed by The MIT License
// that can be found in the LICENSE file.

package sleuth

import (
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/pborman/uuid"
	"github.com/zeromq/gyre"
)

// Returned responses (RECV command) and outstanding requests (RESP command)
// have these headers, respectively: SLEUTH-V0RECV and SLEUTH-V0RESP
const (
	headGroup = len(group)
	// This is the length of "RECV" or "RESP".
	headDispatchAction = 4
	// This is the total length of headers before the actual payload.
	headLength = headGroup + headDispatchAction
	// This is the default timeout duration of outstanding requests.
	timeout = time.Millisecond * 500
)

type listenerHandles struct {
	*sync.Mutex
	handles map[string]chan *http.Response
}

type Client struct {
	// directory maps the name of a node with the type of service it is.
	directory map[string]string
	handler   http.Handler
	node      *gyre.Gyre
	listeners *listenerHandles
	Timeout   time.Duration
	// services maps the name of a service with the list of service worker nodes.
	services map[string]*serviceWorkers
	verbose  bool
}

func (client *Client) add(event *gyre.Event) {
	if header, ok := event.Header("group"); !ok || header != group {
		return
	}
	name := event.Name()
	service, okService := event.Header("type")
	node, okNode := event.Header("node")
	version, _ := event.Header("version")
	// Service and node are required. Version is optional.
	if !(okService && okNode) {
		return
	}
	client.directory[name] = service
	if client.services[service] == nil {
		client.services[service] = newWorkers()
	}
	client.services[service].add(name, node, version)
	notify(client.verbose, "info",
		fmt.Sprintf("sleuth: add %s/%s [%s] to %s", service, version, name, group))
}

func (client *Client) dispatch(event *gyre.Event) {
	message := event.Msg()
	// If the message header does not match the group, bail.
	if len(message) < headLength || string(message[0:headGroup]) != group {
		return
	}
	action := string(message[headGroup : headGroup+headDispatchAction])
	switch action {
	case recv:
		client.receive(message[headLength:])
	case resp:
		client.respond(message[headLength:])
	}
}

func (client *Client) Do(req *http.Request, to string) (*http.Response, error) {
	handle := uuid.New()
	service := to
	services, ok := client.services[service]
	if !ok {
		return nil, fmt.Errorf("sleuth: %s is an unknown service", service)
	}
	peer := services.next().node
	receiver := client.node.UUID()
	payload, err := marshalRequest(receiver, handle, req)
	if err != nil {
		return nil, err
	}
	if err = client.node.Whisper(peer, payload); err != nil {
		return nil, err
	}
	listener := make(chan *http.Response, 1)
	client.listen(handle, listener)
	response := <-listener
	if response != nil {
		return response, nil
	} else {
		return nil, fmt.Errorf("sleuth: %s {%s}%s timed out",
			req.Method, service, req.URL.String())
	}
}

func (client *Client) listen(handle string, listener chan *http.Response) {
	client.listeners.Lock()
	defer client.listeners.Unlock()
	client.listeners.handles[handle] = listener
	go client.timeout(handle)
}

func (client *Client) receive(payload []byte) {
	handle, res, err := unmarshalResponse(payload)
	if err != nil {
		fmt.Printf("sleuth: receive error %s\n", err.Error())
		return
	}
	client.listeners.Lock()
	defer client.listeners.Unlock()
	if listener, ok := client.listeners.handles[handle]; ok {
		listener <- res
		delete(client.listeners.handles, handle)
	}
}

func (client *Client) remove(event *gyre.Event) {
	name := event.Name()
	if service, ok := client.directory[name]; ok {
		remaining := client.services[service].remove(name)
		if remaining == 0 {
			delete(client.services, service)
		}
		delete(client.directory, name)
		notify(client.verbose, "info",
			fmt.Sprintf("sleuth: remove %s [%s] from %s", service, name, group))
	}
}

// respond fails silently because a request that cannot be unmarshalled may
// have not even originated from a compliant client and even if it did, it
// will time out and return an error to the client.
func (client *Client) respond(payload []byte) {
	if dest, req, err := unmarshalRequest(payload); err == nil {
		client.handler.ServeHTTP(newResponseWriter(client.node, dest), req)
	}
}

func (client *Client) timeout(handle string) {
	<-time.After(client.Timeout)
	client.listeners.Lock()
	defer client.listeners.Unlock()
	if listener, ok := client.listeners.handles[handle]; ok {
		listener <- nil
		delete(client.listeners.handles, handle)
	}
}

func newClient(node *gyre.Gyre, verbose bool) *Client {
	return &Client{
		directory: make(map[string]string),
		listeners: &listenerHandles{
			new(sync.Mutex),
			make(map[string]chan *http.Response)},
		node:     node,
		Timeout:  timeout,
		services: make(map[string]*serviceWorkers),
		verbose:  verbose}
}
