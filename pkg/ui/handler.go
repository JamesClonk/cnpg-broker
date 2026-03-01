package ui

import (
	"context"
	"crypto/subtle"
	"errors"
	"net/http"
	"time"

	"github.com/cnpg-broker/pkg/cnpg"
	"github.com/cnpg-broker/pkg/config"
	"github.com/cnpg-broker/pkg/logger"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

type Handler struct{
	client *cnpg.Client
}

type Page struct {

	Title    string
	Clusters []cnpg.ClusterInfo
	Error    struct {
		Code    int
		Message string
		Time    time.Time
	}
}

func New() *Handler {
	return &Handler{
		client: cnpg.NewClient(),
	}
}

func (h *Handler) RegisterRoutes(e *echo.Echo) {
	g := e.Group("")
	cfg := config.Get()

	// don't show timestamp unless specifically configured
	format := `remote_ip="${remote_ip}", host="${host}", method=${method}, uri=${uri}, user_agent="${user_agent}", ` +
		`status=${status}, error="${error}", latency_human="${latency_human}", bytes_out=${bytes_out}` + "\n"
	if cfg.LogTimestamp {
		format = `time="${time_rfc3339}", ` + format
	}
	// add logger middleware
	g.Use(middleware.LoggerWithConfig(middleware.LoggerConfig{
		Format: format,
	}))

	// add auth middleware
	if cfg.Username != "" && cfg.Password != "" {
		g.Use(middleware.BasicAuth(func(u, p string, c echo.Context) (bool, error) {
			if subtle.ConstantTimeCompare([]byte(u), []byte(cfg.Username)) == 1 &&
				subtle.ConstantTimeCompare([]byte(p), []byte(cfg.Password)) == 1 {
				return true, nil
			}
			return false, nil
		}))
	}

	g.GET("/", h.IndexHandler)
	g.GET("/json", h.JSONDataHandler)

	e.HTTPErrorHandler = h.ErrorHandler
}

func (h *Handler) ErrorHandler(err error, c echo.Context) {
	code := http.StatusInternalServerError
	message := "Error"
	if err != nil {
		message = err.Error()
	}

	var he *echo.HTTPError
	if errors.As(err, &he) {
		code = he.Code
		message, _ = he.Message.(string)
	}

	page := h.newPage("Error")
	page.Error.Code = code
	page.Error.Message = message
	page.Error.Time = time.Now()

	logger.Error("%v", err)
	_ = c.Render(code, "error.html", page)
}

func (h *Handler) newPage(title string) *Page {
	return &Page{
		Title:    title,
		Clusters: nil,
	}
}

func (h *Handler) IndexHandler(c echo.Context) error {
	page := h.newPage("Clusters")

	// get data from k8s
	page.Clusters = nil // TODO: read all clusters / service instances from cnpg client.go

	return c.Render(http.StatusOK, "index.html", page)
}

func (h *Handler) JSONDataHandler(c echo.Context) error {
	clusters, err := h.client.ListClusters(context.Background())
	if err != nil {
		logger.Error("failed to list clusters: %v", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": err.Error(),
		})
	}

	return c.JSON(http.StatusOK, clusters)
}
