package broker

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/cnpg-broker/pkg/catalog"
	"github.com/cnpg-broker/pkg/cnpg"
	"github.com/cnpg-broker/pkg/logger"
	"github.com/cnpg-broker/pkg/validation"
	"github.com/labstack/echo/v4"
)

type Broker struct {
	client *cnpg.Client
}

func NewBroker() *Broker {
	return &Broker{
		client: cnpg.NewClient(),
	}
}

func (b *Broker) GetCatalog(c echo.Context) error {
	logger.Debug("catalog requested")
	return c.JSON(http.StatusOK, catalog.GetCatalog())
}

func (b *Broker) ProvisionInstance(c echo.Context) error {
	instanceId := c.Param("instance_id")

	if err := validation.ValidateInstanceID(instanceId); err != nil {
		logger.Warn("invalid instance_id: %v", err)
		return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}

	var req struct {
		ServiceID string         `json:"service_id"`
		PlanID    string         `json:"plan_id"`
		Context   map[string]any `json:"context"`
	}
	if err := c.Bind(&req); err != nil {
		logger.Error("failed to parse provision request for %s: %v", instanceId, err)
		return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}

	if err := validation.ValidateServiceID(req.ServiceID); err != nil {
		logger.Warn("invalid service_id [%s] for %s: %v", req.ServiceID, instanceId, err)
		return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}
	if err := validation.ValidatePlanID(req.ServiceID, req.PlanID); err != nil {
		logger.Warn("invalid plan_id [%s] for %s: %v", req.PlanID, instanceId, err)
		return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}

	existing, err := b.client.GetCluster(context.Background(), instanceId)
	if err == nil {
		logger.Info("instance %s already exists, checking compatibility", instanceId)

		// TODO: also compare CPU, Memory and Storage to "existing", must add those fields to GetCluster!
		instances, _, _, _ := catalog.PlanSpec(req.PlanID)
		if existing.Instances == instances {
			logger.Info("instance %s already provisioned with matching t-shirt size", instanceId)
			return c.JSON(http.StatusOK, map[string]any{
				"instance_id": instanceId,
			})
		} else {
			logger.Warn("instance %s exists but with different t-shirt size", instanceId)
			return c.JSON(http.StatusConflict, map[string]string{
				"error": "instance exists with different configuration",
			})
		}
	}

	logger.Info("provisioning instance %s with plan %s", instanceId, req.PlanID)
	_, err = b.client.CreateCluster(context.Background(), instanceId, req.PlanID)
	if err != nil {
		if strings.Contains(err.Error(), "already exists") {
			logger.Info("instance %s was created concurrently", instanceId)
			return c.JSON(http.StatusOK, map[string]any{})
		}
		logger.Error("failed to provision instance %s: %v", instanceId, err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	logger.Info("successfully provisioned instance %s", instanceId)
	return c.JSON(http.StatusCreated, map[string]any{
		"instance_id": instanceId,
	})
}

func (b *Broker) GetInstance(c echo.Context) error {
	instanceId := c.Param("instance_id")

	if err := validation.ValidateInstanceID(instanceId); err != nil {
		logger.Warn("invalid instance_id: %v", err)
		return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}

	logger.Debug("checking instance %s", instanceId)
	clusterInfo, err := b.client.GetCluster(context.Background(), instanceId)
	if err != nil {
		if strings.Contains(err.Error(), fmt.Sprintf("\"%s\" not found", instanceId)) {
			logger.Debug("instance %s not found", instanceId)
			return c.JSON(http.StatusNotFound, map[string]any{})
		} else {
			logger.Error("failed to get instance %s: %v", instanceId, err)
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
		}
	}

	return c.JSON(http.StatusOK, clusterInfo)
}

func (b *Broker) DeprovisionInstance(c echo.Context) error {
	instanceId := c.Param("instance_id")

	if err := validation.ValidateInstanceID(instanceId); err != nil {
		logger.Warn("invalid instance_id: %v", err)
		return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}

	logger.Info("deprovisioning instance %s", instanceId)
	_, err := b.client.GetCluster(context.Background(), instanceId)
	if err != nil {
		if strings.Contains(err.Error(), fmt.Sprintf("\"%s\" not found", instanceId)) {
			logger.Warn("attempted to deprovision non-existent instance %s", instanceId)
			return c.JSON(http.StatusNotFound, map[string]any{})
		} else {
			logger.Error("failed to check instance %s: %v", instanceId, err)
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
		}
	}

	err = b.client.DeleteCluster(context.Background(), instanceId)
	if err != nil {
		logger.Error("failed to deprovision instance %s: %v", instanceId, err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	logger.Info("successfully deprovisioned instance %s", instanceId)
	return c.JSON(http.StatusOK, map[string]any{})
}

func (b *Broker) BindInstance(c echo.Context) error {
	instanceId := c.Param("instance_id")
	bindingId := c.Param("binding_id")

	if err := validation.ValidateInstanceID(instanceId); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}
	if err := validation.ValidateBindingID(bindingId); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}

	logger.Info("creating binding %s for instance %s", bindingId, instanceId)
	_, err := b.client.GetCluster(context.Background(), instanceId)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			logger.Warn("attempted to bind non-existent instance %s", instanceId)
			return c.JSON(http.StatusNotFound, map[string]string{
				"error": "instance not found",
			})
		}
		logger.Error("failed to check instance %s: %v", instanceId, err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	credentials, err := b.client.GetCredentials(context.Background(), instanceId)
	if err != nil {
		logger.Error("failed to get credentials for instance %s: %v", instanceId, err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	logger.Info("successfully created binding %s for instance %s", bindingId, instanceId)
	return c.JSON(http.StatusOK, map[string]any{
		"credentials": credentials,
	})
}

func (b *Broker) GetBinding(c echo.Context) error {
	instanceId := c.Param("instance_id")
	bindingId := c.Param("binding_id")

	if err := validation.ValidateInstanceID(instanceId); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}
	if err := validation.ValidateBindingID(bindingId); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}

	logger.Debug("retrieving binding %s for instance %s", bindingId, instanceId)
	credentials, err := b.client.GetCredentials(context.Background(), instanceId)
	if err != nil {
		logger.Error("failed to get credentials for instance %s: %v", instanceId, err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, map[string]any{
		"credentials": credentials,
	})
}

func (b *Broker) UnbindInstance(c echo.Context) error {
	instanceId := c.Param("instance_id")
	bindingId := c.Param("binding_id")

	if err := validation.ValidateInstanceID(instanceId); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}
	if err := validation.ValidateBindingID(bindingId); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}

	logger.Info("unbinding %s from instance %s (no-op)", bindingId, instanceId)
	return c.JSON(http.StatusOK, map[string]any{})
}
