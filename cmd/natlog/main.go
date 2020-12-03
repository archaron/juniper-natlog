package main

import (
	"github.com/archaron/juniper-natlog/misc"
	"github.com/archaron/juniper-natlog/modules/app"
	"github.com/im-kulikov/helium"
	"github.com/spf13/viper"
	"github.com/urfave/cli"
	"go.uber.org/dig"
	"os"
	"time"
)

func defaults(v *viper.Viper) {
	v.SetDefault("debug", true)

	// pprof-service
	v.SetDefault("pprof.address", ":6060")
	v.SetDefault("pprof.shutdown_timeout", "10s")

	// metrics-service
	v.SetDefault("metrics.address", ":8090")
	v.SetDefault("metrics.shutdown_timeout", "10s")

	// metrics-service
	v.SetDefault("api.address", ":8098")
	v.SetDefault("api.shutdown_timeout", "10s")

	// syslog listener
	v.SetDefault("syslog.timeout", 10 * time.Second)
	v.SetDefault("syslog.shutdown_timeout", 10 * time.Second)

	v.SetDefault("syslog.address", ":514")


	// logger:
	v.SetDefault("logger.format", "console")
	v.SetDefault("logger.level", "debug")
	v.SetDefault("logger.trace_level", "fatal")
	v.SetDefault("logger.no_disclaimer", false)
	v.SetDefault("logger.color", true)
	v.SetDefault("logger.full_caller", false)
	v.SetDefault("logger.sampling.initial", 100)
	v.SetDefault("logger.sampling.thereafter", 100)
}


func main() {
	c := cli.NewApp()
	c.Name = misc.Name
	c.Version = misc.Version

	// Default action
	c.Action = func(*cli.Context) error {
		h, err := helium.New(&helium.Settings{
			Prefix:       misc.Prefix,
			Name:         misc.Name,
			File:         misc.Config,
			BuildTime:    misc.Version,
			BuildVersion: misc.Build,
			Defaults: defaults,
		}, app.Module)

		err = dig.RootCause(err)
		helium.Catch(err)

		return h.Run()
	}

	err := c.Run(os.Args)

	err = dig.RootCause(err)
	helium.Catch(err)
}
