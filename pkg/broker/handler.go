package broker

import (
	"crypto/subtle"

	"github.com/cnpg-broker/pkg/config"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
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
	g := e.Group("/v2")
	cfg := config.Get()

	format := `remote_ip="${remote_ip}", host="${host}", method=${method}, uri=${uri}, user_agent="${user_agent}", ` +
		`status=${status}, error="${error}", latency_human="${latency_human}", bytes_out=${bytes_out}` + "\n"
	if cfg.LogTimestamp {
		format = `time="${time_rfc3339}", ` + format
	}
	g.Use(middleware.LoggerWithConfig(middleware.LoggerConfig{
		Format: format,
	}))

	if cfg.Username != "" && cfg.Password != "" {
		g.Use(middleware.BasicAuth(func(u, p string, c echo.Context) (bool, error) {
			if subtle.ConstantTimeCompare([]byte(u), []byte(cfg.Username)) == 1 &&
				subtle.ConstantTimeCompare([]byte(p), []byte(cfg.Password)) == 1 {
				return true, nil
			}
			return false, nil
		}))
	}

	g.GET("/catalog", h.broker.GetCatalog)
	g.PUT("/service_instances/:instance_id", h.broker.ProvisionInstance)
	g.GET("/service_instances/:instance_id", h.broker.GetInstance)
	g.DELETE("/service_instances/:instance_id", h.broker.DeprovisionInstance)
	g.PUT("/service_instances/:instance_id/service_bindings/:binding_id", h.broker.BindInstance)
	g.GET("/service_instances/:instance_id/service_bindings/:binding_id", h.broker.GetBinding)
	g.DELETE("/service_instances/:instance_id/service_bindings/:binding_id", h.broker.UnbindInstance)
}
