package health

import (
	"context"
	"net/http"
	"time"

	"github.com/cnpg-broker/pkg/logger"
	"github.com/labstack/echo/v4"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

type Handler struct {
	client dynamic.Interface
}

func New() *Handler {
	config, err := rest.InClusterConfig()
	if err != nil {
		config, err = clientcmd.BuildConfigFromFlags("", clientcmd.RecommendedHomeFile)
		if err != nil {
			logger.Warn("failed to create k8s config for health checks: %v", err)
			return &Handler{client: nil}
		}
	}

	dynClient, err := dynamic.NewForConfig(config)
	if err != nil {
		logger.Warn("failed to create k8s client for health checks: %v", err)
		return &Handler{client: nil}
	}

	return &Handler{client: dynClient}
}

func (h *Handler) RegisterRoutes(e *echo.Echo) {
	g := e.Group("")

	g.GET("/health", h.healthz)
	g.GET("/healthz", h.healthz)
}

func (h *Handler) healthz(c echo.Context) error {
	if h.client == nil {
		return c.JSON(http.StatusServiceUnavailable, map[string]any{
			"status": "unhealthy",
			"checks": map[string]string{
				"kubernetes": "unchecked",
			},
		})
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	checks := make(map[string]string)
	healthy := true

	clusterResource := schema.GroupVersionResource{
		Group:    "postgresql.cnpg.io",
		Version:  "v1",
		Resource: "clusters",
	}
	_, err := h.client.Resource(clusterResource).List(ctx, metav1.ListOptions{Limit: 1})
	if err != nil {
		checks["kubernetes"] = "unhealthy"
		checks["error"] = err.Error()
		healthy = false
		logger.Error("health check failed: %v", err)
	} else {
		checks["kubernetes"] = "healthy"
	}

	status := "ok"
	httpStatus := http.StatusOK
	if !healthy {
		status = "unhealthy"
		httpStatus = http.StatusServiceUnavailable
	}

	return c.JSON(httpStatus, map[string]any{
		"status": status,
		"checks": checks,
	})
}
