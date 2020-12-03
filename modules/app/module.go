package app

import (
	"github.com/archaron/juniper-natlog/modules/api"
	"github.com/archaron/juniper-natlog/modules/clickhouse"
	"github.com/go-helium/echo"
	"github.com/im-kulikov/helium"
	"github.com/im-kulikov/helium/grace"
	"github.com/im-kulikov/helium/logger"
	"github.com/im-kulikov/helium/module"
	"github.com/im-kulikov/helium/service"
	"github.com/im-kulikov/helium/settings"
	"github.com/im-kulikov/helium/web"
)

var Module = module.Module{
	{Constructor: newSyslogService},
}.
	Append(
		helium.DefaultApp,
		grace.Module,    // grace context
		settings.Module, // settings module
		logger.Module,   // logger module
		echo.Module,
		api.Module,
		service.Module,
		clickhouse.Module,
		web.DefaultServersModule,
	)
