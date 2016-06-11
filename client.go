// Copyright 2016 Afshin Darian. All rights reserved.
// Use of this source code is governed by The MIT License
// that can be found in the LICENSE file.

package sleuth

import (
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/ursiform/logger"
	"github.com/zeromq/gyre"
)

type listener struct {
	*sync.Mutex
	handles map[string]chan *http.Response
}

type notifier struct {
	*sync.Mutex
	notify chan struct{}
}

// Client is the peer on the sleuth network that makes requests and, if a
// handler has been provided, responds to peer requests.
type Client struct {
	// Timeout is the duration to wait before an outstanding request times out.
	// By default, it is set to 500ms.
	Timeout time.Duration

	additions *notifier
	group     string
	handle    int64
	handler   http.Handler
	listener  *listener
	log       *logger.Logger
	node      *gyre.Gyre

	directory map[string]string   // map[node-name]service-type
	services  map[string]*workers // map[service-type]service-workers
}

func (c *Client) add(group, name, node, service, version string) error {
	if group != c.group {
		c.log.Debug("sleuth: no group header for %s, client-only", name)
		return nil
	}
	// Node and service are required. Version is optional.
	if len(node) == 0 || len(service) == 0 {
		return newError(errAdd, "failed to add %s node?=%t, type?=%t",
			name, len(node) > 0, len(service) > 0)
	}
	// Associate the node name with its service in the directory.
	c.directory[name] = service
	// Create a service workers collection if necessary.
	if c.services[service] == nil {
		c.services[service] = newWorkers()
	}
	// Add peer to the service workers.
	p := &peer{name: name, node: node, service: service, version: version}
	c.services[service].add(p)
	// If necessary, notify the additions channel that a peer has been added.
	if c.additions.notify != nil {
		c.additions.notify <- struct{}{}
	}
	c.log.Info("sleuth: add %s/%s %s to %s", service, version, name, c.group)
	return nil
}

// Returns true if it had to block and false if it returns immediately.
func (c *Client) block(services ...string) bool {
	// Block until the required services are available to the client.
	c.additions.Lock()
	defer c.additions.Unlock()
	// Even though the client may have just checked to see if services exist,
	// the check is performed here in case there was a delay waiting for the
	// additions mutex to become available.
	if c.has(services...) {
		return false
	}
	c.log.Blocked("sleuth: waiting for client to find %s", services)
	c.additions.notify = make(chan struct{})
	for range c.additions.notify {
		if c.has(services...) {
			break
		}
	}
	c.additions.notify = nil
	c.log.Unblocked("sleuth: client found %s", services)
	return true
}

// Close leaves the sleuth network and stops the Gyre node.
func (c *Client) Close() error {
	c.log.Info("%s leaving %s...", c.node.Name(), c.group)
	if err := c.node.Leave(c.group); err != nil {
		return newError(errLeave, err.Error())
	}
	if err := c.node.Stop(); err != nil {
		c.log.Warn("sleuth: %s %s [%d]",
			c.node.Name(), err.Error(), warnClose)
	}
	return nil
}

func (c *Client) dispatch(payload []byte) error {
	// Returned responses (RECV command) and outstanding requests (REPL command)
	// have these headers, respectively: SLEUTH-V0RECV and SLEUTH-V0REPL
	groupLength := len(c.group)
	dispatchLength := 4
	headerLength := groupLength + dispatchLength
	// If the message header does not match the group, bail.
	if len(payload) < headerLength || string(payload[0:groupLength]) != c.group {
		return newError(errDispatchHeader, "bad dispatch header")
	}
	action := string(payload[groupLength : groupLength+dispatchLength])
	switch action {
	case recv:
		return c.receive(payload[headerLength:])
	case repl:
		return c.reply(payload[headerLength:])
	default:
		return newError(errDispatchAction, "bad dispatch action: %s", action)
	}
}

// Do sends an HTTP request to a service and returns and HTTP response. The URL
// for requests needs to use the following format:
// 	sleuth://service-name/requested-path
// For example, a request to the path /bar?baz=qux of a service called
// foo-service would have the URL:
// 	sleuth://foo-service/bar?baz=qux
func (c *Client) Do(req *http.Request) (*http.Response, error) {
	to := req.URL.Host
	// Handles are hexadecimal strings that are incremented by one.
	handle := strconv.FormatInt(c.handle, 16)
	c.handle++
	if req.URL.Scheme != scheme {
		err := newError(errScheme,
			"URL scheme must be \"%s\" in %s", scheme, req.URL.String())
		return nil, err
	}
	services, ok := c.services[to]
	if !ok {
		return nil, newError(errUnknownService, "%s is an unknown service", to)
	}
	p := services.next()
	payload, err := marshalReq(c.group, c.node.UUID(), handle, req)
	if err != nil {
		return nil, err.(*Error).escalate(errRequest)
	}
	c.log.Debug("sleuth: %s %s://%s@%s%s",
		req.Method, scheme, to, p.name, req.URL.String())
	if err = c.node.Whisper(p.node, payload); err != nil {
		return nil, newError(errReqWhisper, err.Error())
	}
	listener := make(chan *http.Response, 1)
	c.listen(handle, listener)
	response := <-listener
	if response != nil {
		return response, nil
	}
	return nil, newError(errTimeout,
		"%s {%s}%s timed out", req.Method, to, req.URL.String())
}

func (c *Client) has(services ...string) bool {
	// Check to see if required services are already registered.
	verified := make(map[string]bool)
	available := 0
	for _, service := range services {
		verified[service] = false
	}
	total := len(verified)
	for service := range verified {
		if workers, ok := c.services[service]; ok && workers.available() {
			verified[service] = true
			available += 1
		}
	}
	return available == total
}

func (c *Client) listen(handle string, listener chan *http.Response) {
	c.listener.Lock()
	defer c.listener.Unlock()
	c.listener.handles[handle] = listener
	go c.timeout(handle)
}

func (c *Client) receive(payload []byte) error {
	handle, res, err := resUnmarshal(payload)
	if err != nil {
		return err.(*Error).escalate(errRECV)
	}
	c.listener.Lock()
	defer c.listener.Unlock()
	if listener, ok := c.listener.handles[handle]; ok {
		listener <- res
		delete(c.listener.handles, handle)
	} else {
		return newError(errRECV, "unknown handle %d", handle)
	}
	return nil
}

func (c *Client) remove(name string) {
	if service, ok := c.directory[name]; ok {
		remaining, _ := c.services[service].remove(name)
		if remaining == 0 {
			delete(c.services, service)
		}
		delete(c.directory, name)
		c.log.Info("sleuth: remove %s:%s", service, name)
		return
	}
}

func (c *Client) reply(payload []byte) error {
	dest, req, err := unmarshalReq(payload)
	if err != nil {
		return err.(*Error).escalate(errREPL)
	}
	c.handler.ServeHTTP(newWriter(c.node, dest), req)
	return nil
}

func (c *Client) timeout(handle string) {
	<-time.After(c.Timeout)
	c.listener.Lock()
	defer c.listener.Unlock()
	if listener, ok := c.listener.handles[handle]; ok {
		listener <- nil
		delete(c.listener.handles, handle)
	}
}

// WaitFor blocks until the required services are available to the client.
func (c *Client) WaitFor(services ...string) {
	if !c.has(services...) {
		c.block(services...)
	}
}

func newClient(group string, node *gyre.Gyre, out *logger.Logger) *Client {
	return &Client{
		additions: &notifier{Mutex: new(sync.Mutex)},
		directory: make(map[string]string),
		group:     group,
		listener: &listener{
			new(sync.Mutex),
			make(map[string]chan *http.Response)},
		log:      out,
		node:     node,
		Timeout:  time.Millisecond * 500,
		services: make(map[string]*workers)}
}
