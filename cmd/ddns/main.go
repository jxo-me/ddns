package main

import (
	"context"
	"flag"
	"fmt"
	"github.com/jxo-me/ddns/cmd/ddns/cliutil"
	"github.com/jxo-me/ddns/cmd/ddns/service"
	"github.com/jxo-me/ddns/config"
	"github.com/jxo-me/ddns/core/logger"
	"github.com/jxo-me/ddns/pkg/overwatch"
	"github.com/jxo-me/ddns/pkg/watcher"
	xlogger "github.com/jxo-me/ddns/sdk/logger"
	"github.com/urfave/cli/v2"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"sync"
)

var (
	cfgFile      string
	services     stringList
	outputFormat string

	Version   = "DEV"
	BuildTime = "unknown"
	BuildType = ""
)

func init() {
	args := strings.Join(os.Args[1:], "  ")

	if strings.Contains(args, " -- ") {
		var (
			wg  sync.WaitGroup
			ret int
		)

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		for wid, wargs := range strings.Split(" "+args+" ", " -- ") {
			wg.Add(1)
			go func(wid int, wargs string) {
				defer wg.Done()
				defer cancel()
				worker(wid, strings.Split(wargs, "  "), &ctx, &ret)
			}(wid, strings.TrimSpace(wargs))
		}

		wg.Wait()

		os.Exit(ret)
	}
}

func worker(id int, args []string, ctx *context.Context, ret *int) {
	cmd := exec.CommandContext(*ctx, os.Args[0], args...)

	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Env = append(os.Environ(), fmt.Sprintf("_DDNS_ID=%d", id))

	cmd.Run()
	if cmd.ProcessState.Exited() {
		*ret = cmd.ProcessState.ExitCode()
	}
}

func init() {
	var printVersion bool
	flag.StringVar(&cfgFile, "C", "", "configuration file")
	flag.BoolVar(&printVersion, "V", false, "print version")
	flag.Parse()
	if printVersion {
		fmt.Fprintf(os.Stdout, "ddns %s (%s %s/%s)\n",
			version, runtime.Version(), runtime.GOOS, runtime.GOARCH)
		os.Exit(0)
	}
	logger.SetDefault(xlogger.NewLogger())
}

func main() {
	bInfo := cliutil.GetBuildInfo(BuildType, Version)
	// Graceful shutdown channel used by the app. When closed, app must terminate gracefully.
	// Windows service manager closes this channel when it receives stop command.
	graceShutdownC := make(chan struct{})

	app := &cli.App{}
	app.Name = "DDns"
	app.Usage = "DDns's command-line tool and agent"
	app.UsageText = "cloudflared [global options] [command] [command options]"

	app.Version = fmt.Sprintf("%s (built %s%s)", Version, BuildTime, bInfo.GetBuildTypeMsg())
	app.Description = `ddns connects your machine or user identity to Cloudflare's global network.
	You can use it to authenticate a session to reach an API behind Access, route web traffic to this machine,
	and configure access control.

	See https://developers.cloudflare.com/cloudflare-one/connections/connect-apps for more in-depth documentation.`
	app.Flags = flags()
	app.Action = action(graceShutdownC)
	app.Commands = commands(cli.ShowVersion)

	service.RunApp(app, graceShutdownC)
}

func isEmptyInvocation(c *cli.Context) bool {
	return c.NArg() == 0 && c.NumFlags() == 0
}

func flags() []cli.Flag {
	flags := tunnel.Flags()
	return append(flags, access.Flags()...)
}

func action(graceShutdownC chan struct{}) cli.ActionFunc {
	return cliutil.ConfiguredAction(func(c *cli.Context) (err error) {
		if isEmptyInvocation(c) {
			return handleServiceMode(c, graceShutdownC)
		}
		func() {
			defer sentry.Recover()
			err = tunnel.TunnelCommand(c)
		}()
		if err != nil {
			captureError(err)
		}
		return err
	})
}

func commands(version func(c *cli.Context)) []*cli.Command {
	cmds := []*cli.Command{
		{
			Name:   "update",
			Action: cliutil.ConfiguredAction(updater.Update),
			Usage:  "Update the agent if a new version exists",
			Flags: []cli.Flag{
				&cli.BoolFlag{
					Name:  "beta",
					Usage: "specify if you wish to update to the latest beta version",
				},
				&cli.BoolFlag{
					Name:   "force",
					Usage:  "specify if you wish to force an upgrade to the latest version regardless of the current version",
					Hidden: true,
				},
				&cli.BoolFlag{
					Name:   "staging",
					Usage:  "specify if you wish to use the staging url for updating",
					Hidden: true,
				},
				&cli.StringFlag{
					Name:   "version",
					Usage:  "specify a version you wish to upgrade or downgrade to",
					Hidden: false,
				},
			},
			Description: `Looks for a new version on the official download server.
If a new version exists, updates the agent binary and quits.
Otherwise, does nothing.

To determine if an update happened in a script, check for error code 11.`,
		},
		{
			Name: "version",
			Action: func(c *cli.Context) (err error) {
				version(c)
				return nil
			},
			Usage:       versionText,
			Description: versionText,
		},
	}
	cmds = append(cmds, tunnel.Commands()...)
	cmds = append(cmds, proxydns.Command(false))
	cmds = append(cmds, access.Commands()...)
	cmds = append(cmds, tail.Command())
	return cmds
}

func handleServiceMode(c *cli.Context, shutdownC chan struct{}) error {
	log := logger.CreateLoggerFromContext(c, logger.DisableTerminalLog)

	// start the main run loop that reads from the config file
	f, err := watcher.NewFile()
	if err != nil {
		log.Err(err).Msg("Cannot load config file")
		return err
	}

	configPath := config.FindOrCreateConfigPath()
	configManager, err := config.NewFileManager(f, configPath, log)
	if err != nil {
		log.Err(err).Msg("Cannot setup config file for monitoring")
		return err
	}
	log.Info().Msgf("monitoring config file at: %s", configPath)

	serviceCallback := func(t string, name string, err error) {
		if err != nil {
			log.Err(err).Msgf("%s service: %s encountered an error", t, name)
		}
	}
	serviceManager := overwatch.NewAppManager(serviceCallback)

	appService := NewAppService(configManager, serviceManager, shutdownC, log)
	if err := appService.Run(); err != nil {
		log.Err(err).Msg("Failed to start app service")
		return err
	}
	return nil
}
