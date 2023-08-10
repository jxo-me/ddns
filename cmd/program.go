package main

import (
	"github.com/judwhite/go-svc"
	"github.com/jxo-me/ddns/config"
	"github.com/jxo-me/ddns/core/logger"
	"github.com/jxo-me/ddns/x/app"
	"log"
	"os"
)

type program struct{}

func (p *program) Init(env svc.Environment) error {
	cfg := &config.Config{}
	if cfgFile != "" {
		if err := cfg.ReadFile(cfgFile); err != nil {
			return err
		}
	}
	// build config from command line
	cmdCfg, err := buildConfigFromCmd(services)
	if err != nil {
		return err
	}
	// merge config
	cfg = p.mergeConfig(cfg, cmdCfg)
	// set default logger
	logger.SetDefault(logFromConfig(cfg.Log))
	// set default output format
	if outputFormat != "" {
		if err := cfg.Write(os.Stdout, outputFormat); err != nil {
			return err
		}
		os.Exit(0)
	}
	// load config
	config.Set(cfg)
	return nil
}

func (p *program) Start() error {
	cfg := config.Global()
	for _, ddns := range buildService(cfg) {
		srv := ddns
		go func() {
			_ = srv.Serve()
		}()
	}
	return nil
}

func (p *program) Stop() error {
	for name, srv := range app.Runtime.DDNSRegistry().GetAll() {
		_ = srv.Close()
		log.Default().Printf("service %s shutdown", name)
	}
	return nil
}

func (p *program) mergeConfig(cfg1, cfg2 *config.Config) *config.Config {
	if cfg1 == nil {
		return cfg2
	}
	if cfg2 == nil {
		return cfg1
	}

	cfg := &config.Config{
		Services: append(cfg1.Services, cfg2.Services...),
	}

	return cfg
}
