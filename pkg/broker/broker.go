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

func (b *Broker) ProvisionInstance(c echo.Context) error {
	instanceID := c.Param("instance_id")
	if instanceID == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "instance_id required"})
	}

	var req struct {
		ServiceID string         `json:"service_id"`
		PlanID    string         `json:"plan_id"`
		Context   map[string]any `json:"context"`
	}
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}

	namespace, err := b.client.CreateCluster(context.Background(), instanceID, req.PlanID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusCreated, map[string]any{
		"dashboard_url": namespace,
	})
}

func (b *Broker) GetInstance(c echo.Context) error {
	instanceID := c.Param("instance_id")
	if instanceID == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "instance_id required"})
	}

	cluster, err := b.client.GetCluster(context.Background(), instanceID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, map[string]any{
		"dashboard_url": cluster,
	})
}

func (b *Broker) DeprovisionInstance(c echo.Context) error {
	instanceID := c.Param("instance_id")
	if instanceID == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "instance_id required"})
	}

	err := b.client.DeleteCluster(context.Background(), instanceID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, map[string]any{})
}

func (b *Broker) BindInstance(c echo.Context) error {
	instanceID := c.Param("instance_id")
	if instanceID == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "instance_id required"})
	}

	creds, err := b.client.GetCredentials(context.Background(), instanceID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusCreated, map[string]any{
		"credentials": creds,
	})
}

func (b *Broker) GetBinding(c echo.Context) error {
	return b.BindInstance(c)
}

func (b *Broker) UnbindInstance(c echo.Context) error {
	return c.JSON(http.StatusOK, map[string]interface{}{})
}
