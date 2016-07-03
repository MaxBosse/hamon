package main

import (
	"io/ioutil"
	"log"

	"github.com/MaxBosse/hamon/haproxy"
	"gopkg.in/yaml.v2"
)

type Config struct {
	HideNoCheck   bool
	Timeout       int
	Loadbalancers map[string]haproxy.Loadbalancer
}

func (config *Config) Parse(data []byte) error {
	if err := yaml.Unmarshal(data, &config); err != nil {
		return err
	}
	return nil
}

func (config *Config) Load(path string) {
	data, err := ioutil.ReadFile(path)
	if err != nil {
		log.Fatal(err)
	}
	if err := config.Parse(data); err != nil {
		log.Fatal(err)
	}
}
