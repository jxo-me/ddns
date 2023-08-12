package main

import (
	"errors"
	"fmt"
	"github.com/jxo-me/ddns/config"
	"net/url"
	"strings"
)

var (
	ErrInvalidCmd = errors.New("invalid cmd")
)

type stringList []string

func (l *stringList) String() string {
	return fmt.Sprintf("%s", *l)
}

func (l *stringList) Set(value string) error {
	*l = append(*l, value)
	return nil
}

func buildConfigFromCmd(services stringList) (*config.Config, error) {
	namePrefix := ""
	cfg := &config.Config{}

	for i, svc := range services {
		url, err := normCmd(svc)
		if err != nil {
			return nil, err
		}

		service, err := buildServiceConfig(url)
		if err != nil {
			return nil, err
		}
		service.Name = fmt.Sprintf("%sservice-%d", namePrefix, i)

		cfg.DDns = append(cfg.DDns, service)

	}

	return cfg, nil
}

func buildServiceConfig(url *url.URL) (*config.DDnsConfig, error) {

	return nil, nil
}

func normCmd(s string) (*url.URL, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil, ErrInvalidCmd
	}

	if s[0] == ':' || !strings.Contains(s, "://") {
		s = "auto://" + s
	}

	url, err := url.Parse(s)
	if err != nil {
		return nil, err
	}
	if url.Scheme == "https" {
		url.Scheme = "http+tls"
	}

	return url, nil
}
