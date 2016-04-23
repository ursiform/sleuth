// Copyright 2016 Afshin Darian. All rights reserved.
// Use of this source code is governed by The MIT License
// that can be found in the LICENSE file.

package sleuth

import (
	"encoding/json"
	"io/ioutil"
)

const ConfigFile = "bear.json"

type appConfig struct {
	Service *serviceConfig `json:"service,omitempty"`
	Sleuth  *sleuthConfig  `json:"sleuth,omitempty"`
}

type serviceConfig struct {
	Name    string `json:"name,omitempty"`
	Version string `json:"version,omitempty"`
}

type sleuthConfig struct {
	Interface string `json:"interface,omitempty"`
	Port      int    `json:"port,omitempty"`
}

func loadConfig() *appConfig {
	appConfig := &appConfig{
		Service: new(serviceConfig),
		Sleuth:  new(sleuthConfig)}
	if data, err := ioutil.ReadFile(ConfigFile); err == nil {
		_ = json.Unmarshal(data, appConfig)
	}
	return appConfig
}
