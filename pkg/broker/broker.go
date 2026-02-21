package broker

import (
	"context"
	"fmt"
	"net/http"
	"strings"

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
	instanceId := c.Param("instance_id")
	if len(instanceId) == 0 {
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

	url, err := b.client.CreateCluster(context.Background(), instanceId, req.PlanID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusCreated, map[string]any{
		"dashboard_url": url,
	})
}

func (b *Broker) GetInstance(c echo.Context) error {
	instanceId := c.Param("instance_id")
	if len(instanceId) == 0 {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "instance_id required"})
	}

	// HTTP 410 must be returned if Service Instance does not exist
	url, err := b.client.GetCluster(context.Background(), instanceId)
	if err != nil {
		if strings.Contains(err.Error(), fmt.Sprintf("\"%s\" not found", instanceId)) {
			return c.JSON(http.StatusNotFound, map[string]any{})
		} else {
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
		}
	}

	return c.JSON(http.StatusOK, map[string]any{
		"dashboard_url": url,
	})
}

func (b *Broker) DeprovisionInstance(c echo.Context) error {
	instanceId := c.Param("instance_id")
	if len(instanceId) == 0 {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "instance_id required"})
	}

	// HTTP 410 must be returned if Service Instance does not exist
	_, err := b.client.GetCluster(context.Background(), instanceId)
	if err != nil {
		if strings.Contains(err.Error(), fmt.Sprintf("\"%s\" not found", instanceId)) {
			return c.JSON(http.StatusNotFound, map[string]any{})
		} else {
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
		}
	}

	err = b.client.DeleteCluster(context.Background(), instanceId)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	// respond with HTTP 200 for DELETE
	return c.JSON(http.StatusOK, map[string]any{})
}

func (b *Broker) BindInstance(c echo.Context) error {
	instanceId := c.Param("instance_id")
	if len(instanceId) == 0 {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "instance_id required"})
	}

	credentials, err := b.client.GetCredentials(context.Background(), instanceId)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	// respond with HTTP 201 for new PUTs
	return c.JSON(http.StatusCreated, map[string]any{
		"credentials": credentials,
	})
}

func (b *Broker) GetBinding(c echo.Context) error {
	instanceId := c.Param("instance_id")
	if len(instanceId) == 0 {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "instance_id required"})
	}

	credentials, err := b.client.GetCredentials(context.Background(), instanceId)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	// respond with HTTP 200 for GETs
	return c.JSON(http.StatusOK, map[string]any{
		"credentials": credentials,
	})
}

func (b *Broker) UnbindInstance(c echo.Context) error {
	return c.JSON(http.StatusOK, map[string]any{})
}
