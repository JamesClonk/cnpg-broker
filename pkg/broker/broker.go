package broker

import (
	"context"
	"net/http"

	"github.com/cnpg-broker/pkg/catalog"
	"github.com/cnpg-broker/pkg/cnpg"
	"github.com/labstack/echo/v4"
)

type Broker struct {
	client *cnpg.Client
}

func NewBroker() *Broker {
	return &Broker{
		client: cnpg.NewClient(), // TODO: uncomment
	}
}

func (b *Broker) GetCatalog(c echo.Context) error {
	return c.JSON(http.StatusOK, catalog.GetCatalog())
}

func (b *Broker) Provision(c echo.Context) error {
	instanceID := c.Param("instance_id")

	var req struct {
		ServiceID string                 `json:"service_id"`
		PlanID    string                 `json:"plan_id"`
		Context   map[string]interface{} `json:"context"`
	}
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}

	namespace, err := b.client.CreateCluster(context.Background(), instanceID, req.PlanID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusCreated, map[string]interface{}{
		"dashboard_url": namespace,
	})
}

func (b *Broker) Deprovision(c echo.Context) error {
	namespace := c.QueryParam("namespace")
	if namespace == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "namespace required"})
	}

	err := b.client.DeleteCluster(context.Background(), namespace)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, map[string]interface{}{})
}

func (b *Broker) Bind(c echo.Context) error {
	instanceID := c.Param("instance_id")

	namespace := c.QueryParam("namespace")
	if namespace == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "namespace required"})
	}

	creds, err := b.client.GetCredentials(context.Background(), instanceID, namespace)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusCreated, map[string]interface{}{
		"credentials": creds,
	})
}

func (b *Broker) Unbind(c echo.Context) error {
	return c.JSON(http.StatusOK, map[string]interface{}{})
}
