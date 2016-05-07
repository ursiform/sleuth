// Copyright 2016 Afshin Darian. All rights reserved.
// Use of this source code is governed by The MIT License
// that can be found in the LICENSE file.

package sleuth

import (
	"encoding/json"
	"io/ioutil"

	"github.com/ursiform/logger"
)

// ConfigFile is the default config file that sleuth will check for if no
// other file location is passed into New.
const ConfigFile = "bear.json"

type appConfig struct {
	LogLevel     int
	LogLevelName string         `json:"loglevel,omitempty"`
	Service      *serviceConfig `json:"service,omitempty"`
	Sleuth       *sleuthConfig  `json:"sleuth,omitempty"`
}

type serviceConfig struct {
	Name    string `json:"name,omitempty"`
	Version string `json:"version,omitempty"`
}

type sleuthConfig struct {
	Interface string `json:"interface,omitempty"`
	Port      int    `json:"port,omitempty"`
}

func loadConfig(file string) *appConfig {
	appConfig := &appConfig{
		Service: new(serviceConfig),
		Sleuth:  new(sleuthConfig)}
	if data, err := ioutil.ReadFile(file); err == nil {
		_ = json.Unmarshal(data, appConfig)
	}
	if len(appConfig.LogLevelName) == 0 {
		appConfig.LogLevelName = "listen"
	}
	level, ok := logger.LogLevel[appConfig.LogLevelName]
	if !ok {
		logger.MustError("loglevel=\"%s\" in %s is invalid; using \"%s\" (%d)",
			appConfig.LogLevelName, file, "debug", errLogLevel)
		appConfig.LogLevelName = "debug"
		appConfig.LogLevel = logger.Debug
	} else {
		appConfig.LogLevel = level
	}
	return appConfig
}
