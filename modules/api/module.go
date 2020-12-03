package api

import (
	"net/http"

	"github.com/archaron/juniper-natlog/modules/clickhouse"
	"github.com/im-kulikov/helium/module"
	"github.com/im-kulikov/helium/settings"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/spf13/viper"
	"go.uber.org/dig"
	"go.uber.org/zap"
)

type (
	Router struct {
		dig.In

		Engine  *echo.Echo
		Logger  *zap.Logger
		Config  *viper.Viper
		Setting *settings.Core
		Clickhouse *clickhouse.Service
	}
)

// Module application
var Module = module.Module{
	{Constructor: newRouter},
}

func newRouter(r Router) http.Handler {

	e := r.Engine
	e.Pre(middleware.AddTrailingSlash())
	e.Use(middleware.CORSWithConfig(middleware.CORSConfig{
		AllowOrigins: []string{"*"},
		AllowMethods: []string{http.MethodGet, http.MethodPut, http.MethodPost, http.MethodDelete},
	}))

	e.Use(middleware.Recover())
	//e.Use(hecho.LoggerMiddleware(r.Logger))

	e.GET("/version/", func(ctx echo.Context) error {
		return ctx.JSON(http.StatusOK, map[string]interface{}{
			"version":   r.Setting.BuildVersion,
			"buildTime": r.Setting.BuildTime,
			"status":    "ok",
		})
	})

	e.GET("/health/", func(ctx echo.Context) error {
		return ctx.JSON(http.StatusOK, map[string]interface{}{"status": "ok"})
	})

	e.GET("/readiness/", func(ctx echo.Context) error {

		if err := r.Clickhouse.Ping(); err != nil {
			return ctx.JSON(http.StatusInternalServerError, map[string]interface{}{"status": "fail", "reason": "clickhouse server ping fail", "error": err.Error()})
		}

		return ctx.JSON(http.StatusOK, map[string]interface{}{"status": "ok"})
	})

	return e
}
