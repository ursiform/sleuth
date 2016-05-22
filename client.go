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
	additions chan *peer
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
	services  map[string]*serviceWorkers // map[service-type]service-workers
}

func (c *Client) add(gid, name, node, service, version string) error {
	if gid != group {
		c.log.Debug("sleuth: no group header for %s, client-only", name)
		return nil
	}
	p := &peer{Name: name}
	// Node and service are required. Version is optional.
	if len(node) == 0 || len(service) == 0 {
		return fmt.Errorf("sleuth: failed to add %s node?=%t, type?=%t (%d)",
			name, len(node) > 0, len(service) > 0, warnAdd)
	}
	p.Node = node
	p.Service = service
	p.Version = version
	c.directory[name] = service
	if c.services[service] == nil {
		c.services[service] = newWorkers()
	}
	c.services[service].add(p)
	if c.waiters.additions != nil {
		c.waiters.additions <- p
	}
	c.log.Info("sleuth: add %s/%s %s to %s", service, version, name, group)
	return nil
}

// Close leaves the sleuth network and stops the Gyre/Zyre node.
func (c *Client) Close() error {
	c.log.Info("%s leaving %s...", c.node.Name(), group)
	if err := c.node.Leave(group); err != nil {
		return err
	}
	if err := c.node.Stop(); err != nil {
		c.log.Warn("sleuth: %s %s (%d)",
			c.node.Name(), err.Error(), warnClose)
	}
	return nil
}

func (c *Client) dispatch(payload []byte) error {
	// Returned responses (RECV command) and outstanding requests (REPL command)
	// have these headers, respectively: SLEUTH-V0RECV and SLEUTH-V0REPL
	groupLength := len(group)
	dispatchLength := 4
	headerLength := groupLength + dispatchLength
	// If the message header does not match the group, bail.
	if len(payload) < headerLength || string(payload[0:groupLength]) != group {
		return fmt.Errorf("sleuth: bad header (%d)", errDispatchHeader)
	}
	action := string(payload[groupLength : groupLength+dispatchLength])
	switch action {
	case recv:
		c.receive(payload[headerLength:])
		return nil
	case repl:
		c.reply(payload[headerLength:])
		return nil
	default:
		return fmt.Errorf("sleuth: bad action: %s (%d)", action, errDispatchAction)
	}
}

// Do sends an HTTP request to a service and returns and HTTP response. The URL
// for requests needs to use the following format:
//	sleuth://service-name/requested-path
// For example, a request to the path /users?foo=bar of a service called
// user-service would have the URL:
// 	sleuth://user-service/users?foo=bar
func (c *Client) Do(req *http.Request) (*http.Response, error) {
	handle := uuid.New()
	to := req.URL.Host
	if req.URL.Scheme != scheme {
		err := fmt.Errorf("sleuth: URL scheme must be \"%s\" in %s (%d)",
			scheme, req.URL.String(), errUnsupportedScheme)
		c.log.Error(err.Error())
		return nil, err
	}
	services, ok := c.services[to]
	if !ok {
		return nil, fmt.Errorf("sleuth: %s is an unknown service (%d)",
			to, errUnknownService)
	}
	p := services.next()
	receiver := c.node.UUID()
	payload, err := marshalRequest(receiver, handle, req)
	if err != nil {
		return nil, err
	}
	c.log.Debug("sleuth: %s %s://%s@%s%s",
		req.Method, scheme, to, p.Name, req.URL.String())
	if err = c.node.Whisper(p.Node, payload); err != nil {
		return nil, err
	}
	listener := make(chan *http.Response, 1)
	c.listen(handle, listener)
	response := <-listener
	if response != nil {
		return response, nil
	} else {
		return nil, fmt.Errorf("sleuth: %s {%s}%s timed out (%d)",
			req.Method, to, req.URL.String(), errTimeout)
	}
}

func (c *Client) listen(handle string, listener chan *http.Response) {
	c.listeners.Lock()
	defer c.listeners.Unlock()
	c.listeners.handles[handle] = listener
	go c.timeout(handle)
}

func (c *Client) receive(payload []byte) {
	handle, res, err := unmarshalResponse(payload)
	if err != nil {
		c.log.Error("sleuth: %s (%d)", err.Error(), errReceiveUnmarshal)
		return
	}
	c.listeners.Lock()
	defer c.listeners.Unlock()
	if listener, ok := c.listeners.handles[handle]; ok {
		listener <- res
		delete(c.listeners.handles, handle)
	} else {
		c.log.Error("sleuth: unknown handle %s (%d)", handle, errReceiveHandle)
	}
}

func (c *Client) remove(name string) {
	if service, ok := c.directory[name]; ok {
		remaining, _ := c.services[service].remove(name)
		if remaining == 0 {
			delete(c.services, service)
		}
		delete(c.directory, name)
		c.log.Info("sleuth: remove %s (%s) from %s", service, name, group)
	}
}

// reply fails silently because a request that cannot be unmarshalled may
// have not even originated from a compliant client and even if it did, it
// will time out and return an error to the client.
func (c *Client) reply(payload []byte) {
	if dest, req, err := unmarshalRequest(payload); err != nil {
		c.log.Error("sleuth: %s (%d)", err.Error(), errReply)
	} else {
		c.handler.ServeHTTP(newResponseWriter(c.node, dest), req)
	}
}

func (c *Client) timeout(handle string) {
	<-time.After(c.Timeout)
	c.listeners.Lock()
	defer c.listeners.Unlock()
	if listener, ok := c.listeners.handles[handle]; ok {
		listener <- nil
		delete(c.listeners.handles, handle)
	}
}

// WaitFor blocks until the required services are available in the pool.
func (c *Client) WaitFor(services ...string) {
	c.waiters.Lock()
	defer c.waiters.Unlock()
	verified := make(map[string]bool)
	available := 0
	for _, service := range services {
		verified[service] = false
	}
	total := len(verified)
	for service, _ := range verified {
		if workers, ok := c.services[service]; ok && workers.available() {
			verified[service] = true
			available += 1
		}
	}
	if available == total {
		return
	}
	c.log.Blocked("sleuth: waiting for client to find services %s", services)
	c.waiters.additions = make(chan *peer)
	for p := range c.waiters.additions {
		service := p.Service
		if exists, ok := verified[service]; ok && !exists {
			verified[service] = true
			available += 1
			if available == total {
				break
			}
		}
	}
	c.waiters.additions = nil
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
