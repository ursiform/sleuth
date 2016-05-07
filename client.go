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
	"github.com/ursiform/logger"
	"github.com/zeromq/gyre"
)

type listenerHandles struct {
	*sync.Mutex
	handles map[string]chan *http.Response
}

type waiterChannels struct {
	*sync.Mutex
	additions chan *Peer
}

// Client is the peer on the sleuth network that makes requests and, if a
// handler has been provided, responds to peer requests.
type Client struct {
	handler   http.Handler
	node      *gyre.Gyre
	listeners *listenerHandles
	log       *logger.Logger
	// Timeout is the duration to wait before an outstanding request times out.
	// By default, it is set to 500ms.
	Timeout   time.Duration
	waiters   *waiterChannels
	directory map[string]string          // map[node-name]service-type
	services  map[string]*serviceWorkers // map[service-name]service-workers
}

func (client *Client) add(event *gyre.Event) {
	name := event.Name()
	if header, ok := event.Header("group"); !ok || header != group {
		client.log.Debug("sleuth: add - no group header for %s, client-only", name)
		return
	}
	peer := &Peer{Name: name}
	service, okService := event.Header("type")
	node, okNode := event.Header("node")
	version, _ := event.Header("version")
	// Service and node are required. Version is optional.
	if !(okService && okNode) {
		client.log.Warn("sleuth: failed to add %s type?=%t, node?=%t (%d)",
			name, okService, okNode, warnAdd)
		return
	}
	peer.Node = node
	peer.Service = service
	peer.Version = version
	client.directory[name] = service
	if client.services[service] == nil {
		client.services[service] = newWorkers()
	}
	client.services[service].add(peer)
	if client.waiters.additions != nil {
		client.waiters.additions <- peer
	}
	client.log.Info("sleuth: add %s/%s %s to %s", service, version, name, group)
}

// Close leaves the sleuth network and stops the Gyre/Zyre node.
func (client *Client) Close() error {
	client.log.Info("%s leaving %s...", client.node.Name(), group)
	if err := client.node.Leave(group); err != nil {
		return err
	}
	if err := client.node.Stop(); err != nil {
		client.log.Warn("sleuth: %s %s (%d)",
			client.node.Name(), err.Error(), warnClose)
	}
	return nil
}

func (client *Client) dispatch(event *gyre.Event) {
	message := event.Msg()
	// Returned responses (RECV command) and outstanding requests (REPL command)
	// have these headers, respectively: SLEUTH-V0RECV and SLEUTH-V0REPL
	groupLength := len(group)
	dispatchLength := 4
	headerLength := groupLength + dispatchLength
	// If the message header does not match the group, bail.
	if len(message) < headerLength || string(message[0:groupLength]) != group {
		client.log.Error("sleuth: bad header (%d)", errDispatchHeader)
		return
	}
	action := string(message[groupLength : groupLength+dispatchLength])
	switch action {
	case recv:
		client.receive(message[headerLength:])
	case repl:
		client.reply(message[headerLength:])
	default:
		client.log.Error("sleuth: bad action: %s (%d)", action, errDispatchAction)
	}
}

// Do sends an HTTP request to a service and returns and HTTP response.
func (client *Client) Do(req *http.Request, to string) (*http.Response, error) {
	handle := uuid.New()
	services, ok := client.services[to]
	if !ok {
		return nil, fmt.Errorf("sleuth: %s is an unknown service (%d)",
			to, errUnknownService)
	}
	peer := services.next()
	receiver := client.node.UUID()
	payload, err := marshalRequest(receiver, handle, req)
	if err != nil {
		return nil, err
	}
	client.log.Debug("sleuth: Do - request {%s@%s}%s",
		to, peer.Name, req.URL.String())
	if err = client.node.Whisper(peer.Node, payload); err != nil {
		return nil, err
	}
	listener := make(chan *http.Response, 1)
	client.listen(handle, listener)
	response := <-listener
	if response != nil {
		return response, nil
	} else {
		return nil, fmt.Errorf("sleuth: %s {%s}%s timed out (%d)",
			req.Method, to, req.URL.String(), errTimeout)
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
		client.log.Error("sleuth: %s (%d)", err.Error(), errReceiveUnmarshal)
		return
	}
	client.listeners.Lock()
	defer client.listeners.Unlock()
	if listener, ok := client.listeners.handles[handle]; ok {
		listener <- res
		delete(client.listeners.handles, handle)
	} else {
		client.log.Error("sleuth: unknown handle %s (%d)", handle, errReceiveHandle)
	}
}

func (client *Client) remove(event *gyre.Event) {
	name := event.Name()
	if service, ok := client.directory[name]; ok {
		remaining, _ := client.services[service].remove(name)
		if remaining == 0 {
			delete(client.services, service)
		}
		delete(client.directory, name)
		client.log.Info("sleuth: remove %s (%s) from %s", service, name, group)
	}
}

// reply fails silently because a request that cannot be unmarshalled may
// have not even originated from a compliant client and even if it did, it
// will time out and return an error to the client.
func (client *Client) reply(payload []byte) {
	if dest, req, err := unmarshalRequest(payload); err != nil {
		client.log.Error("sleuth: %s (%d)", err.Error(), errReply)
	} else {
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

// WaitFor blocks until the required services are available in the pool.
func (client *Client) WaitFor(services ...string) {
	client.waiters.Lock()
	defer client.waiters.Unlock()
	verified := make(map[string]bool)
	available := 0
	for _, service := range services {
		verified[service] = false
	}
	total := len(verified)
	for service, _ := range verified {
		if workers, ok := client.services[service]; ok && workers.available() {
			verified[service] = true
			available += 1
		}
	}
	if available == total {
		return
	}
	client.log.Blocked("sleuth: waiting for client to find services %s", services)
	client.waiters.additions = make(chan *Peer)
	for peer := range client.waiters.additions {
		service := peer.Service
		if exists, ok := verified[service]; ok && !exists {
			verified[service] = true
			available += 1
			if available == total {
				break
			}
		}
	}
	client.waiters.additions = nil
}

func newClient(node *gyre.Gyre, out *logger.Logger) *Client {
	return &Client{
		directory: make(map[string]string),
		listeners: &listenerHandles{
			new(sync.Mutex),
			make(map[string]chan *http.Response)},
		log:      out,
		node:     node,
		Timeout:  time.Millisecond * 500,
		services: make(map[string]*serviceWorkers),
		waiters:  &waiterChannels{Mutex: new(sync.Mutex)}}
}
