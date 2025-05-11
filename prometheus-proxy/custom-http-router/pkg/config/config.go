package config

import (
    "io/ioutil"
    "log"

    "gopkg.in/yaml.v2"
)

type Config struct {
    Server struct {
        Port int `yaml:"port"`
    } `yaml:"server"`
}

func LoadConfig(filePath string) (*Config, error) {
    data, err := ioutil.ReadFile(filePath)
    if err != nil {
        return nil, err
    }

    var config Config
    if err := yaml.Unmarshal(data, &config); err != nil {
        return nil, err
    }

    return &config, nil
}

func (c *Config) Validate() error {
    if c.Server.Port == 0 {
        return log.Output(2, "Server port must be specified")
    }
    return nil
}