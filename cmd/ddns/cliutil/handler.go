package cliutil

import (
	"errors"
	"github.com/jxo-me/ddns/config"
	"github.com/jxo-me/ddns/pkg/logger"
	"github.com/urfave/cli/v2"
	"github.com/urfave/cli/v2/altsrc"
)

func Action(actionFunc cli.ActionFunc) cli.ActionFunc {
	return WithErrorHandler(actionFunc)
}

func ConfiguredAction(actionFunc cli.ActionFunc) cli.ActionFunc {
	// Adapt actionFunc to the type signature required by ConfiguredActionWithWarnings
	f := func(context *cli.Context, _ string) error {
		return actionFunc(context)
	}

	return ConfiguredActionWithWarnings(f)
}

// ConfiguredActionWithWarnings Just like ConfiguredAction, but accepts a second parameter with configuration warnings.
func ConfiguredActionWithWarnings(actionFunc func(*cli.Context, string) error) cli.ActionFunc {
	return WithErrorHandler(func(c *cli.Context) error {
		warnings, err := setFlagsFromConfigFile(c)
		if err != nil {
			return err
		}
		return actionFunc(c, warnings)
	})
}

func setFlagsFromConfigFile(c *cli.Context) (configWarnings string, err error) {
	const errorExitCode = 1
	log := logger.CreateLoggerFromContext(c, logger.EnableTerminalLog)
	inputSource, warnings, err := config.ReadConfigFile(c, log)
	if err != nil {
		if errors.Is(err, config.ErrNoConfigFile) {
			return "", nil
		}
		return "", cli.Exit(err, errorExitCode)
	}
	var flag []cli.Flag
	if err := altsrc.ApplyInputSourceValues(c, inputSource, flag); err != nil {
		return "", cli.Exit(err, errorExitCode)
	}
	return warnings, nil
}
