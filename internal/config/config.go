package config

import (
	"errors"
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
	Clients []ClientConfig `json:"client" yaml:"client"`
}

func (c *Config) Validate() error {
	if len(c.Servers) == 0 {
		return errors.New("config: empty servers")
	}
	return nil
}

func NewConfig(path string) (*Config, error) {
	// 读取配置文件
	con, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var conf Config
	err = yaml.Unmarshal(con, &conf)
	if err != nil {
		return nil, err
	}

	if err := conf.Validate(); err != nil {
		return nil, err
	}
	return &conf, nil
}
