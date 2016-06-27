// Copyright 2016 Afshin Darian. All rights reserved.
// Use of this source code is governed by The MIT License
// that can be found in the LICENSE file.

// Package sleuth provides master-less peer-to-peer autodiscovery and RPC
// between HTTP services that reside on the same network. It works with minimal
// configuration and provides a mechanism to join a local network both as a
// client that offers no services and as any service that speaks HTTP. Its
// primary use case is for microservices on the same network that make calls to
// one another.
package sleuth

import (
	"net/http"

	"github.com/ursiform/logger"
	"github.com/zeromq/gyre"
)

const (
	group  = "SLEUTH-v0"
	port   = 5670
	recv   = "RECV"
	repl   = "REPL"
	scheme = "sleuth"
)

type connection struct {
	adapter string
	group   string
	handler http.Handler
	name    string
	node    string
	port    int
	server  bool
	version string
}

func dispatch(client *Client, event *gyre.Event) (err error) {
	name := event.Name()
	switch event.Type() {
	case gyre.EventEnter:
		group, _ := event.Header("group")
		node, _ := event.Header("node")
		service, _ := event.Header("type")
		version, _ := event.Header("version")
		err = client.add(group, name, node, service, version)
	case gyre.EventExit, gyre.EventLeave:
		client.remove(name)
	case gyre.EventWhisper:
		err = client.dispatch(event.Msg())
	}
	if err != nil {
		err.(*Error).escalate(errDispatch)
	}
	return
}

func listen(client *Client) {
	for {
		if err := dispatch(client, <-client.node.Events()); err != nil {
			client.log.Error(err.Error())
		}
	}
}

func newNode(conn *connection, log *logger.Logger) (*gyre.Gyre, error) {
	node, err := gyre.New()
	if err != nil {
		return nil, newError(errInitialize, err.Error())
	}
	if err := node.SetPort(conn.port); err != nil {
		return nil, newError(errPort, err.Error())
	}
	if conn.adapter != "" {
		if err := node.SetInterface(conn.adapter); err != nil {
			return nil, newError(errInterface, err.Error())
		}
	}
	// If announcing a service, add service headers.
	if conn.server {
		errors := [...]int{
			errGroupHeader, errNodeHeader, errServiceHeader, errVersionHeader}
		values := [...]string{conn.group, node.UUID(), conn.name, conn.version}
		for i, header := range [...]string{"group", "node", "type", "version"} {
			if err := node.SetHeader(header, values[i]); err != nil {
				return nil, newError(errors[i], err.Error())
			}
		}
	}
	if err := node.Start(); err != nil {
		return nil, newError(errStart, err.Error())
	}
	if err := node.Join(group); err != nil {
		node.Stop()
		return nil, newError(errJoin, err.Error())
	}
	var role string
	if conn.server {
		role = conn.name
	} else {
		role = "client-only"
	}
	log.Listen("sleuth: [%s:%d][%s %s]", group, conn.port, role, node.Name())
	return node, nil
}

// New is the entry point to the sleuth package. It returns a reference to a
// Client object that has joined the local network. If the config argument is
// nil, sleuth will use sensible defaults. If the Handler attribute of the
// config object is not set, sleuth will operate in client-only mode.
func New(config *Config) (*Client, error) {
	// Sanitize the configuration object.
	config = initConfig(config)
	// Ignore errors because log level is guaranteed to be correct in initConfig.
	log, _ := logger.New(config.logLevel)
	conn := &connection{group: config.group}
	if conn.server = config.Handler != nil; conn.server {
		conn.handler = config.Handler
		conn.name = config.Service
		if conn.name == "" {
			return nil, newError(errService, "config.Service not defined")
		}
	} else {
		log.Init("sleuth: config.Handler is nil, client-only mode")
	}
	if conn.adapter = config.Interface; conn.adapter == "" {
		log.Warn("sleuth: config.Interface not defined [%d]", warnInterface)
	}
	if conn.port = config.Port; conn.port == 0 {
		conn.port = port
	}
	if conn.version = config.Version; conn.version == "" {
		conn.version = "unknown"
	}
	node, err := newNode(conn, log)
	if err != nil {
		return nil, err.(*Error).escalate(errNew)
	}
	client := newClient(config.group, node, log)
	client.handler = conn.handler
	go listen(client)
	return client, nil
}
