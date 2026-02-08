package router

import (
	"fmt"

	"github.com/cnpg-broker/pkg/broker"
	"github.com/cnpg-broker/pkg/health"
	"github.com/cnpg-broker/pkg/metrics"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

type Router struct {
	echo    *echo.Echo
	health  *health.Handler
	metrics *metrics.Handler
	broker  *broker.Handler
	//ui      *ui.Handler
}

func New() *Router {
	// setup basic echo configuration
	e := echo.New()
	e.DisableHTTP2 = true
	e.HideBanner = false
	e.HidePort = false

	// middlewares
	e.Pre(middleware.RemoveTrailingSlash())
	e.Use(middleware.Secure())
	// e.Use(middleware.Recover()) // don't recover, let platform deal with panics
	e.Use(middleware.Static("static"))

	// setup router
	r := &Router{
		echo:    e,
		health:  health.New(),
		metrics: metrics.New(),
		broker:  broker.New(),
		//ui:      ui.New(),
	}

	// setup health route
	r.health.RegisterRoutes(r.echo)

	// setup metrics route
	r.metrics.RegisterRoutes(r.echo)

	// setup broker/api routes
	r.broker.RegisterRoutes(r.echo)

	// // setup Web-UI routes
	// r.ui.RegisterRoutes(r.echo)
	// // setup Web-UI rendering
	// r.ui.RegisterRenderer(r.echo)

	return r
}

func (r *Router) Start(port int) error {
	return r.echo.Start(fmt.Sprintf(":%d", port))
}
