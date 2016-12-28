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

// Client is the peer on the sleuth network that makes requests and, if a
// handler has been provided, responds to peer requests.
type Client struct {
	// Timeout is the duration to wait before an outstanding request times out.
	// By default, it is set to 500ms.
	Timeout time.Duration

	additions *notifier
	closed    bool
	group     string
	handle    int64
	handler   http.Handler
	listener  *listener
	log       *logger.Logger
	node      *gyre.Gyre

	directory map[string]string // map[node-name]service-type
	services  *pool
}

func (c *Client) add(group, name, node, service, version string) error {
	if group != c.group {
		c.log.Debug("sleuth: no group header for %s, client-only", name)
		return nil
	}
	// Node and service are required. Version is optional.
	if node == "" || service == "" {
		format := "add failed for name=\"%s\", node=\"%s\", service=\"%s\""
		return newError(errAdd, format, name, node, service)
	}
	// Associate the node name with its service in the directory.
	c.directory[name] = service
	// Idempotently create a service workers pool.
	c.services.add(service)
	// Add peer to the service workers.
	p := &peer{name: name, node: node, service: service, version: version}
	c.services.workers[service].add(p)
	c.additions.notify()
	c.log.Info("sleuth: add %s/%s %s to %s", service, version, name, c.group)
	return nil
}

// Blocks until the required services are available to the client.
// Returns true if it had to block and false if it returns immediately.
func (c *Client) block(required map[string]struct{}, services []string) bool {
	// Even though the client may have just checked to see if services exist,
	// the check is performed here in case there was a delay waiting for the
	// additions mutex to become available.
	if c.has(required) {
		return false
	}
	c.log.Blocked("sleuth: waiting for client to find %s", services)
	c.additions.activate()
	for range c.additions.stream {
		if c.has(required) {
			break
		}
	}
	c.additions.deactivate()
	c.log.Unblocked("sleuth: client found %s", services)
	return true
}

// Close leaves the sleuth network and stops the Gyre node. It can only be
// called once, even if it returns an error the first time it is called.
func (c *Client) Close() error {
	defer func(c *Client) { c.closed = true }(c)
	if c.closed {
		return newError(errClosed, "client is already closed")
	}
	c.log.Info("%s leaving %s...", c.node.Name(), c.group)
	if err := c.node.Leave(c.group); err != nil {
		return newError(errLeave, err.Error())
	}
	if err := c.node.Stop(); err != nil {
		c.log.Warn("sleuth: %s %s [%d]", c.node.Name(), err.Error(), warnClose)
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

// Do sends an HTTP request to a service and returns an HTTP response. URLs for
// requests use the following format:
// 	sleuth://service-name/requested-path
// For example, a request to the path /bar?baz=qux of a service called
// foo-service would have the URL:
// 	sleuth://foo-service/bar?baz=qux
func (c *Client) Do(req *http.Request) (*http.Response, error) {
	if c.closed {
		return nil, newError(errClosed, "client is closed").escalate(errDo)
	}
	url := req.URL.String()
	to := req.URL.Host
	// Handles are hexadecimal strings that are incremented by one.
	handle := strconv.FormatInt(c.handle, 16)
	c.handle++
	if req.URL.Scheme != scheme {
		err := newError(errScheme, "URL scheme must be \"%s\" in %s", scheme, url)
		return nil, err
	}
	peers, ok := c.services.get(to)
	if !ok {
		return nil, newError(errUnknownService, "%s is an unknown service", to)
	}
	payload, err := reqMarshal(c.group, c.node.UUID(), handle, req)
	if err != nil {
		return nil, err.(*Error).escalate(errDo)
	}
	p := peers.next()
	c.log.Debug("sleuth: %s %s via %s", req.Method, url, p.name)
	if err = c.node.Whisper(p.node, payload); err != nil {
		return nil, newError(errReqWhisper, err.Error())
	}
	listener := make(chan *http.Response, 1)
	c.listen(handle, listener)
	response := <-listener
	if response != nil {
		return response, nil
	}
	return nil, newError(errTimeout, "%s {%s}%s timed out", req.Method, to, url)
}

func (c *Client) has(required map[string]struct{}) bool {
	// Check to see if required services already exist locally.
	available := 0
	for service := range required {
		if peers, ok := c.services.get(service); ok && peers.available() {
			available += 1
		}
	}
	return available == len(required)
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
		return nil
	}
	return newError(errRECV, "unknown handle %d", handle)
}

func (c *Client) remove(name string) {
	if service, ok := c.directory[name]; ok {
		if peers, ok := c.services.get(service); ok {
			if remaining, _ := peers.remove(name); remaining == 0 {
				c.services.remove(service)
			}
		}
		delete(c.directory, name)
		c.log.Info("sleuth: remove %s:%s", service, name)
	}
}

func (c *Client) reply(payload []byte) error {
	dest, req, err := reqUnmarshal(c.group, payload)
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
func (c *Client) WaitFor(services ...string) error {
	if c.closed {
		return newError(errClosed, "client is closed").escalate(errWait)
	}
	// Collapse services and make sure all values are unique.
	required := make(map[string]struct{})
	for _, service := range services {
		required[service] = struct{}{}
	}
	if len(required) != len(services) {
		c.log.Warn("sleuth: %v contains duplicates [%d]", services, warnDuplicate)
	}
	if !c.has(required) {
		c.block(required, services)
	}
	return nil
}

func newClient(group string, node *gyre.Gyre, out *logger.Logger) *Client {
	return &Client{
		additions: &notifier{
			Mutex:  new(sync.Mutex),
			stream: make(chan struct{}),
		},
		directory: make(map[string]string),
		group:     group,
		listener: &listener{
			Mutex:   new(sync.Mutex),
			handles: make(map[string]chan *http.Response),
		},
		log:     out,
		node:    node,
		Timeout: time.Millisecond * 500,
		services: &pool{
			Mutex:   new(sync.Mutex),
			workers: make(map[string]*workers),
		},
	}
}
