package main

import (
	"github.com/jxo-me/ddns/config"
	"github.com/jxo-me/ddns/config/parsing"
	"github.com/jxo-me/ddns/core/logger"
	"github.com/jxo-me/ddns/core/service"
	"github.com/jxo-me/ddns/sdk/app"
	xlogger "github.com/jxo-me/ddns/sdk/logger"
	"gopkg.in/natefinch/lumberjack.v2"
	"io"
	"os"
	"path/filepath"
)

func buildService(cfg *config.Config) (services []service.IDDNSService) {
	if cfg == nil {
		return
	}
	log := logger.Default()
	for _, svcCfg := range cfg.DDns {
		svc, err := parsing.ParseService(svcCfg, log)
		if err != nil {
			log.Fatal(err)
		}
		if svc != nil {
			if err = app.Runtime.DDNSRegistry().Register(svcCfg.Name, svc); err != nil {
				log.Fatal(err)
			}
			services = append(services, svc)
		}
	}
	return
}

func logFromConfig(cfg *config.LogConfig) logger.ILogger {
	if cfg == nil {
		cfg = &config.LogConfig{}
	}
	opts := []xlogger.LoggerOption{
		xlogger.FormatLoggerOption(logger.LogFormat(cfg.Format)),
		xlogger.LevelLoggerOption(logger.LogLevel(cfg.Level)),
	}

	var out io.Writer = os.Stderr
	switch cfg.Output {
	case "none", "null":
		return xlogger.Nop()
	case "stdout":
		out = os.Stdout
	case "stderr", "":
		out = os.Stderr
	default:
		if cfg.Rotation != nil {
			out = &lumberjack.Logger{
				Filename:   cfg.Output,
				MaxSize:    cfg.Rotation.MaxSize,
				MaxAge:     cfg.Rotation.MaxAge,
				MaxBackups: cfg.Rotation.MaxBackups,
				LocalTime:  cfg.Rotation.LocalTime,
				Compress:   cfg.Rotation.Compress,
			}
		} else {
			os.MkdirAll(filepath.Dir(cfg.Output), 0755)
			f, err := os.OpenFile(cfg.Output, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
			if err != nil {
				logger.Default().Warn(err)
			} else {
				out = f
			}
		}
	}
	opts = append(opts, xlogger.OutputLoggerOption(out))

	return xlogger.NewLogger(opts...)
}
