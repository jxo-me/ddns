package config

import (
	"encoding/json"
	"github.com/spf13/viper"
	"gopkg.in/yaml.v3"
	"io"
	"sync"
)

var (
	v = viper.GetViper()
)

func init() {
	v.SetConfigName("config")
	v.AddConfigPath("/etc/ddns/")
	v.AddConfigPath("$HOME/.ddns/")
	v.AddConfigPath(".")
}

var (
	global    = &Config{}
	globalMux sync.RWMutex
)

type Config struct {
	Services []*DnsConfig `json:"services"`
	Log      *LogConfig   `yaml:",omitempty" json:"log,omitempty"`
}

func Global() *Config {
	globalMux.RLock()
	defer globalMux.RUnlock()

	cfg := &Config{}
	*cfg = *global
	return cfg
}

func Set(c *Config) {
	globalMux.Lock()
	defer globalMux.Unlock()

	global = c
}

func OnUpdate(f func(c *Config) error) error {
	globalMux.Lock()
	defer globalMux.Unlock()

	return f(global)
}

func (c *Config) Load() error {
	if err := v.ReadInConfig(); err != nil {
		return err
	}

	return v.Unmarshal(c)
}

func (c *Config) Read(r io.Reader) error {
	if err := v.ReadConfig(r); err != nil {
		return err
	}

	return v.Unmarshal(c)
}

func (c *Config) ReadFile(file string) error {
	v.SetConfigFile(file)
	if err := v.ReadInConfig(); err != nil {
		return err
	}
	return v.Unmarshal(c)
}

func (c *Config) Write(w io.Writer, format string) error {
	switch format {
	case "json":
		enc := json.NewEncoder(w)
		enc.SetIndent("", "  ")
		enc.Encode(c)
		return nil
	case "yaml":
		fallthrough
	default:
		enc := yaml.NewEncoder(w)
		defer enc.Close()
		enc.SetIndent(2)

		return enc.Encode(c)
	}
}
