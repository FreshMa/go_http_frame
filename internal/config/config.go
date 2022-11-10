package config

import (
	"errors"
	"fmt"
	"io/ioutil"

	"gopkg.in/yaml.v2"
)

// 这里是config.yml文件对应的结构

// ServerConfig 服务的配置
type ServerConfig struct {
	Name     string `json:"name" yaml:"name"`
	Listen   string `json:"listen" yaml:"listen"`
	Protocol string `json:"http" yaml:"http"`
}

type LogConfig struct {
	Path  string `json:"path" yaml:"path"`
	Level string `json:"level" yaml:"level"`
}

type ClientConfig struct {
	Name string `json:"name" yaml:"name"`
	// 客户端类型，可以是其他的rpc服务地址，也可以是mysql或者redis的服务地址
	Type string `json:"type" yaml:"type"`
	// 多个addr使用逗号分割
	Addr         string `json:"addr" yaml:"addr"`
	User         string `json:"user" yaml:"user"`
	Auth         string `json:"auth" yaml:"auth"`
	ReadTimeout  int    `json:"read_timeout" yaml:"read_timeout"`
	WriteTimeout int    `json:"write_timeout" yaml:"write_timeout"`
}

type Config struct {
	Servers []ServerConfig `json:"server" yaml:"server"`
	Log     LogConfig      `json:"log" yaml:"log"`
	Clients []ClientConfig `json:"clients" yaml:"clients"`

	mpClients map[string]ClientConfig
}

func (c *Config) Validate() error {
	if len(c.Servers) == 0 {
		return errors.New("config: empty servers")
	}

	for _, cli := range c.Clients {
		if cli.Name == "" || cli.Addr == "" {
			return errors.New("config: empty cli name or addr")
		}
	}
	return nil
}

func (c *Config) GetCliConfigByName(name string) (*ClientConfig, error) {
	cli, ok := c.mpClients[name]
	if !ok {
		return nil, errors.New("invalid name")
	}
	return &cli, nil
}

func NewConfig(path string) (*Config, error) {
	// 读取配置文件
	con, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}

	conf := &Config{
		mpClients: make(map[string]ClientConfig),
	}
	err = yaml.Unmarshal(con, conf)
	if err != nil {
		return nil, err
	}

	if err := conf.Validate(); err != nil {
		return nil, err
	}

	for _, cli := range conf.Clients {
		cli := cli
		fmt.Printf("name:%s\n", cli.Name)
		conf.mpClients[cli.Name] = cli
	}
	return conf, nil
}
