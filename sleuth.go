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
	"log"
	"net/http"
	"os"
	"os/signal"

	"github.com/zeromq/gyre"
)

const (
	port  = 5670
	group = "SLEUTH-v0"
	recv  = "RECV"
	resp  = "RESP"
)

type connection struct {
	adapter string
	name    string
	node    string
	port    int
	version string
}

func announce(done chan *Client, conn *connection, verbose bool) {
	node, err := newNode(verbose, conn)
	if err != nil {
		fmt.Printf("sleuth: %s\n", err.Error())
		done <- nil
		return
	}
	client := newClient(node)
	done <- client
	go interceptInterrupt(node)
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

func failure(verbose bool, err error, code string) error {
	notify(verbose, "warn",
		fmt.Sprintf("%s (%s)", err.Error(), code))
	return err
}

func interceptInterrupt(node *gyre.Gyre) {
	interrupt := make(chan os.Signal)
	signal.Notify(interrupt, os.Interrupt)
	_ = <-interrupt
	fmt.Printf("\nleaving %s...\n", group)
	_ = node.Leave(group)
	_ = node.Stop()
	os.Exit(0)
}

func newNode(verbose bool, conn *connection) (*gyre.Gyre, error) {
	node, err := gyre.New()
	if err != nil {
		return nil, failure(verbose, err, ErrorSleuthInitialize)
	}
	if err = node.SetPort(conn.port); err != nil {
		return nil, failure(verbose, err, ErrorSleuthSetPort)
	}
	if len(conn.adapter) > 0 {
		if err = node.SetInterface(conn.adapter); err != nil {
			return nil, failure(verbose, err, ErrorSleuthInterface)
		}
	}
	if verbose {
		if err = node.SetVerbose(); err != nil {
			return nil, failure(verbose, err, ErrorSleuthSetVerbose)
		}
	}
	// If announcing a service, add service headers.
	if len(conn.name) != 0 {
		if err = node.SetHeader("group", group); err != nil {
			return nil, failure(verbose, err, ErrorSleuthGroupHeader)
		}
		if err = node.SetHeader("node", node.UUID()); err != nil {
			return nil, failure(verbose, err, ErrorSleuthNodeHeader)
		}
		if err = node.SetHeader("type", conn.name); err != nil {
			return nil, failure(verbose, err, ErrorSleuthServiceHeader)
		}
		if err = node.SetHeader("version", conn.version); err != nil {
			return nil, failure(verbose, err, ErrorSleuthVersionHeader)
		}
	}
	if err = node.Start(); err != nil {
		return nil, failure(verbose, err, ErrorSleuthStart)
	}
	if err = node.Join(group); err != nil {
		node.Stop()
		return nil, failure(verbose, err, ErrorSleuthJoin)
	}
	notify(verbose, "listen", fmt.Sprintf(
		"sleuth [%s:%d][node:%s]",
		group, conn.port, node.Name()))
	return node, nil
}

// New is the entry point to the sleuth package. It returns a reference to a
// Client object that has joined the local network. If the handler argument is
// not nil, the Client also answers requests from other peers.
func New(handler http.Handler, verbose bool) (*Client, error) {
	config := loadConfig()
	conn := new(connection)
	if handler == nil {
		notify(verbose, "initialize", "sleuth.New handler is nil, client-only mode")
	} else {
		conn.name = config.Service.Name
		if len(conn.name) == 0 {
			err := fmt.Errorf(
				"sleuth: %s.%s not defined in %s",
				"service", "name", ConfigFile)
			return nil, failure(verbose, err, ErrorSleuthServiceUndefined)
		}
	}
	conn.adapter = config.Sleuth.Interface
	if len(conn.adapter) == 0 {
		notify(verbose, "warn", fmt.Sprintf(
			"%s.%s not defined in %s",
			"sleuth", "interface", ConfigFile))
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
	go announce(done, conn, verbose)
	client := <-done
	if client == nil {
		return nil, fmt.Errorf("sleuth: unable to announce")
	}
	client.handler = handler
	return client, nil
}

func notify(verbose bool, level string, message string) {
	var prefix string
	switch level {
	case "initialize":
		prefix = "[initialized]"
	case "listen":
		prefix = "[ listening ]"
	case "warn":
		prefix = "[**warning**]"
	}
	if verbose {
		log.Printf("%s %s", prefix, message)
	}
}
