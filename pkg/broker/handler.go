package broker

import (
	"github.com/labstack/echo/v4"
)

type Handler struct {
	broker *Broker
}

func New() *Handler {
	return &Handler{
		broker: NewBroker(),
	}
}

func (h *Handler) RegisterRoutes(e *echo.Echo) {
	// everything should be placed under /v2/ as per OSB spec
	g := e.Group("/v2")

	// TODO: properly implement config to uncomment below afterwards
	// // don't show timestamp unless specifically configured
	// format := `remote_ip="${remote_ip}", host="${host}", method=${method}, uri=${uri}, user_agent="${user_agent}", ` +
	// 	`status=${status}, error="${error}", latency_human="${latency_human}", bytes_out=${bytes_out}` + "\n"
	// if config.Get().LoggingTimestamp {
	// 	format = `time="${time_rfc3339}", ` + format
	// }
	// // add logger middlerware
	// g.Use(middleware.LoggerWithConfig(middleware.LoggerConfig{
	// 	Format: format,
	// }))

	// TODO: properly implement config to uncomment below afterwards
	// // secure routes with HTTP BasicAuth
	// username := config.Get().Username
	// password := config.Get().Password
	// g.Use(middleware.BasicAuth(func(u, p string, c echo.Context) (bool, error) {
	// 	if subtle.ConstantTimeCompare([]byte(u), []byte(username)) == 1 && subtle.ConstantTimeCompare([]byte(p), []byte(password)) == 1 {
	// 		return true, nil
	// 	}
	// 	return false, nil
	// }))

	g.GET("/catalog", h.broker.GetCatalog)
	g.PUT("/service_instances/:instance_id", h.broker.Provision)
	g.DELETE("/service_instances/:instance_id", h.broker.Deprovision)
	g.PUT("/service_instances/:instance_id/service_bindings/:binding_id", h.broker.Bind)
	g.DELETE("/service_instances/:instance_id/service_bindings/:binding_id", h.broker.Unbind)
}
