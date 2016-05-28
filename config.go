// Copyright 2016 Afshin Darian. All rights reserved.
// Use of this source code is governed by The MIT License
// that can be found in the LICENSE file.

package sleuth

import (
	"net/http"

	"github.com/ursiform/logger"
)

// Config is the configuration specification for sleuth client instantiation.
// It has JSON tag values defined for all public fields except handler in order
// to allow users to store sleuth configuration in JSON files. All fields are
// optional, but in production settings, Interface is recommended, if known.
type Config struct {
	// Handler is the HTTP handler for a service made available via sleuth.
	Handler http.Handler `json:"-"`

	// Interface is the system network interface sleuth should use, i.e. "en0".
	Interface string `json:"interface,omitempty"`

	// LogLevel is the ursiform.Logger level for sleuth. The default is "listen".
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
	if len(config.LogLevel) == 0 {
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
