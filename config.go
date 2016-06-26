// Copyright 2016 Afshin Darian. All rights reserved.
// Use of this source code is governed by The MIT License
// that can be found in the LICENSE file.

package sleuth

import (
	"net/http"

	"github.com/ursiform/logger"
)

// Config is the configuration specification for sleuth client instantiation.
// It has JSON tag values defined for all public fields except Handler in order
// to allow users to store sleuth configuration in JSON files. All fields are
// optional, but Interface is particularly important to guarantee all peers
// reside on the same subnet.
type Config struct {
	group string

	// Handler is the HTTP handler for a service made available via sleuth.
	Handler http.Handler `json:"-"`

	// Interface is the system network interface sleuth should use, e.g. "en0".
	Interface string `json:"interface,omitempty"`

	// LogLevel is the ursiform.Logger level for sleuth. The default is "listen".
	// The options, in order of increasing verbosity, are:
	// "silent"    No log output at all.
	// "error"     Only errors are logged.
	// "blocked"   Blocking calls and lower are logged.
	// "unblocked" Unblocked notifications and lower are logged.
	// "warn"      Warnings and lower are logged.
	// "reject"    Rejections (e.g., in a firewall) and lower are logged.
	// "listen"    Listeners and lower are logged.
	// "install"   Install notifications and lower are logged.
	// "init"      Initialization notifications and lower are logged.
	// "request"   Incoming requests and lower are logged.
	// "info"      Info output and lower are logged.
	// "debug"     All log output is shown.
	LogLevel string `json:"loglevel,omitempty"`

	// Port is the UDP port that sleuth should broadcast on. The default is 5670.
	Port int `json:"port,omitempty"`

	// Service is the name of the service being offered if a Handler exists.
	Service string `json:"service,omitempty"`

	// Version is the optional version string of the service being offered.
	Version string `json:"version,omitempty"`

	logLevel int
}

func initConfig(config *Config) *Config {
	if config == nil {
		config = new(Config)
	}
	if config.group == "" {
		config.group = group
	}
	if config.LogLevel == "" {
		config.LogLevel = "listen"
	}
	if level, ok := logger.LogLevel[config.LogLevel]; !ok {
		logger.MustError("LogLevel=\"%s\" is invalid; using \"%s\" [%d]",
			config.LogLevel, "debug", errLogLevel)
		config.LogLevel = "debug"
		config.logLevel = logger.Debug
	} else {
		config.logLevel = level
	}
	return config
}
