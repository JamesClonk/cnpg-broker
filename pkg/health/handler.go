package health

import (
	"net/http"

	"github.com/labstack/echo/v4"
)

type Handler struct{}

func New() *Handler {
	return &Handler{}
}

func (h *Handler) RegisterRoutes(e *echo.Echo) {
	g := e.Group("")

	// setup health endpoint
	g.GET("/health", h.healthz)
	g.GET("/healthz", h.healthz)
}

func (h *Handler) healthz(c echo.Context) error {
	return c.JSON(http.StatusOK, struct{ Status string }{Status: "ok"}) // TODO: needs an actual health check, probably ;)
}
