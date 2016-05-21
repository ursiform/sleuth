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
	"errors"
	"fmt"
	"net/http"

	"github.com/ursiform/logger"
	"github.com/zeromq/gyre"
)

// Debug enables logging of underlying Gyre/Zyre messages when set to true.
var Debug = false

var (
	group          = "SLEUTH-v0"
	port           = 5670
	recv           = "RECV"
	repl           = "REPL"
	scheme         = "sleuth"
	serviceHeaders = [...]string{"group", "node", "type", "version"}
	headerErrors   = [...]int{
		errGroupHeader,
		errNodeHeader,
		errServiceHeader,
		errVersionHeader}
)

type connection struct {
	adapter string
	handler http.Handler
	name    string
	node    string
	port    int
	server  bool
	version string
}

type instance struct {
	client *Client
	err    error
}

func announce(conn *connection, log *logger.Logger, result chan *instance) {
	node, err := newNode(log, conn)
	if err != nil {
		result <- &instance{client: nil, err: err}
		return
	}
	client := newClient(node, log)
	client.handler = conn.handler
	result <- &instance{client: client, err: nil}
	for {
		event := <-node.Events()
		switch event.Type() {
		case gyre.EventEnter:
			groupID, _ := event.Header("group")
			name := event.Name()
			node, _ := event.Header("node")
			service, _ := event.Header("type")
			version, _ := event.Header("version")
			if err := client.add(groupID, name, node, service, version); err != nil {
				log.Warn(err.Error())
			}
		case gyre.EventExit, gyre.EventLeave:
			name := event.Name()
			client.remove(name)
		case gyre.EventWhisper:
			client.dispatch(event.Msg())
		}
	}
}

func failure(log *logger.Logger, message string, code int) error {
	text := fmt.Sprintf("sleuth: %s (%d)", message, code)
	log.Error(text)
	return errors.New(text)
}

func newNode(log *logger.Logger, conn *connection) (*gyre.Gyre, error) {
	node, err := gyre.New()
	if err != nil {
		return nil, failure(log, err.Error(), errInitialize)
	}
	if err := node.SetPort(conn.port); err != nil {
		return nil, failure(log, err.Error(), errPort)
	}
	if len(conn.adapter) > 0 {
		if err := node.SetInterface(conn.adapter); err != nil {
			return nil, failure(log, err.Error(), errInterface)
		}
	}
	if Debug {
		if err := node.SetVerbose(); err != nil {
			return nil, failure(log, err.Error(), errVerbose)
		}
	}
	// If announcing a service, add service headers.
	if conn.server {
		values := [...]string{group, node.UUID(), conn.name, conn.version}
		for i, header := range serviceHeaders {
			if err := node.SetHeader(header, values[i]); err != nil {
				return nil, failure(log, err.Error(), headerErrors[i])
			}
		}
	}
	if err := node.Start(); err != nil {
		return nil, failure(log, err.Error(), errStart)
	}
	if err := node.Join(group); err != nil {
		node.Stop()
		return nil, failure(log, err.Error(), errJoin)
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
// Client object that has joined the local network. If the handler argument is
// not nil, the Client also answers requests from other peers. If the configFile
// argument is an empty string, sleuth will automatically attempt to load
// the ConfigFile (bear.json).
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
	log, err := logger.New(config.LogLevel)
	if err != nil {
		return nil, err
	}
	if conn.server = handler != nil; conn.server {
		conn.handler = handler
		conn.name = config.Service.Name
		if len(conn.name) == 0 {
			message := fmt.Sprintf("service.name not defined in %s", file)
			return nil, failure(log, message, errServiceUndefined)
		}
	} else {
		log.Init("sleuth: handler is nil, client-only mode")
	}
	if conn.adapter = config.Sleuth.Interface; len(conn.adapter) == 0 {
		log.Warn("sleuth: sleuth.interface not defined in %s (%d)",
			file, warnInterface)
	}
	if conn.port = config.Sleuth.Port; conn.port == 0 {
		conn.port = port
	}
	if conn.version = config.Service.Version; len(conn.version) == 0 {
		conn.version = "unknown"
	}
	done := make(chan *instance, 1)
	go announce(conn, log, done)
	result := <-done
	return result.client, result.err
}
