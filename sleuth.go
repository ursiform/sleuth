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
	"fmt"
	"net/http"

	"github.com/ursiform/logger"
	"github.com/zeromq/gyre"
)

var (
	Debug = false
	group = "SLEUTH-v0"
	port  = 5670
	recv  = "RECV"
	repl  = "REPL"
)

type connection struct {
	adapter string
	name    string
	node    string
	port    int
	server  bool
	version string
}

func announce(done chan *Client, conn *connection, out *logger.Logger) {
	node, err := newNode(out, conn)
	if err != nil {
		done <- nil
		return
	}
	client := newClient(node, out)
	done <- client
	for {
		event := <-node.Events()
		switch event.Type() {
		case gyre.EventEnter:
			client.add(event)
		case gyre.EventExit, gyre.EventLeave:
			client.remove(event)
		case gyre.EventWhisper:
			client.dispatch(event)
		}
	}
}

func failure(out *logger.Logger, err error, code int) error {
	out.Error("sleuth: %s (%d)", err.Error(), code)
	return err
}

func newNode(out *logger.Logger, conn *connection) (*gyre.Gyre, error) {
	node, err := gyre.New()
	if err != nil {
		return nil, failure(out, err, ErrorInitialize)
	}
	if err := node.SetPort(conn.port); err != nil {
		return nil, failure(out, err, ErrorSetPort)
	}
	if len(conn.adapter) > 0 {
		if err := node.SetInterface(conn.adapter); err != nil {
			return nil, failure(out, err, ErrorInterface)
		}
	}
	if Debug {
		if err := node.SetVerbose(); err != nil {
			return nil, failure(out, err, ErrorSetVerbose)
		}
	}
	// If announcing a service, add service headers.
	if conn.server {
		if err := node.SetHeader("group", group); err != nil {
			return nil, failure(out, err, ErrorGroupHeader)
		}
		if err := node.SetHeader("node", node.UUID()); err != nil {
			return nil, failure(out, err, ErrorNodeHeader)
		}
		if err := node.SetHeader("type", conn.name); err != nil {
			return nil, failure(out, err, ErrorServiceHeader)
		}
		if err := node.SetHeader("version", conn.version); err != nil {
			return nil, failure(out, err, ErrorVersionHeader)
		}
	}
	if err := node.Start(); err != nil {
		return nil, failure(out, err, ErrorStart)
	}
	if err := node.Join(group); err != nil {
		node.Stop()
		return nil, failure(out, err, ErrorJoin)
	}
	var role string
	if conn.server {
		role = conn.name
	} else {
		role = "client-only"
	}
	out.Listen("sleuth: [%s:%d][%s %s]", group, conn.port, role, node.Name())
	return node, nil
}

// New is the entry point to the sleuth package. It returns a reference to a
// Client object that has joined the local network. If the handler argument is
// not nil, the Client also answers requests from other peers.
func New(handler http.Handler, configFile string) (*Client, error) {
	var file string
	if len(configFile) > 0 {
		file = configFile
	} else {
		file = ConfigFile
	}
	config := loadConfig(file)
	conn := new(connection)
	// Use the same log level as the instantiator of the client.
	out, err := logger.New(config.LogLevel)
	if err != nil {
		return nil, err
	}
	if handler == nil {
		out.Init("sleuth: New - handler is nil, client-only mode")
	} else {
		conn.name = config.Service.Name
		if len(conn.name) == 0 {
			err := fmt.Errorf("sleuth: New - %s not defined in %s",
				"service.name", ConfigFile)
			return nil, failure(out, err, ErrorServiceUndefined)
		}
	}
	conn.server = handler != nil
	conn.adapter = config.Sleuth.Interface
	if len(conn.adapter) == 0 {
		out.Warn("sleuth: New - sleuth.interface not defined in %s", ConfigFile)
	}
	conn.port = config.Sleuth.Port
	if conn.port == 0 {
		conn.port = port
	}
	conn.version = config.Service.Version
	if len(conn.version) == 0 {
		conn.version = "unknown"
	}
	done := make(chan *Client, 1)
	go announce(done, conn, out)
	client := <-done
	if client == nil {
		return nil, fmt.Errorf("sleuth: New - unable to announce")
	}
	client.log = out
	client.handler = handler
	return client, nil
}
