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
	acceptsIncomplete := c.QueryParam("accepts_incomplete") == "true"

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

	clusterStatus, err := b.client.GetClusterStatus(context.Background(), instanceId)
	if err != nil {
		logger.Error("failed to check cluster status for %s: %v", instanceId, err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	if clusterStatus.Exists {
		logger.Info("instance %s already exists, checking compatibility", instanceId)
		instances, cpu, memory, storage := catalog.PlanSpec(req.PlanID)

		if clusterStatus.SpecInstances == instances &&
			clusterStatus.SpecCPU == cpu &&
			clusterStatus.SpecMemory == memory &&
			clusterStatus.SpecStorage == storage {

			if clusterStatus.IsReady {
				logger.Info("instance %s already provisioned and ready", instanceId)
				return c.JSON(http.StatusOK, map[string]any{})
			}
			if clusterStatus.IsProvisioning {
				if acceptsIncomplete {
					logger.Info("instance %s provisioning in progress", instanceId)
					return c.JSON(http.StatusAccepted, map[string]any{})
				}
				return c.JSON(http.StatusUnprocessableEntity, map[string]any{
					"error":       "AsyncRequired",
					"description": "Service instance provisioning is in progress and requires async support",
				})
			}
			if clusterStatus.IsFailed {
				logger.Warn("instance %s exists but in failed state", instanceId)
				return c.JSON(http.StatusConflict, map[string]string{
					"error": fmt.Sprintf("instance exists in failed state: %s", clusterStatus.FailureReason),
				})
			}

			return c.JSON(http.StatusOK, map[string]any{})

		} else {
			logger.Warn("instance %s exists but with different specs", instanceId)
			return c.JSON(http.StatusConflict, map[string]string{
				"error": "instance exists with different configuration",
			})
		}
	}

	if !acceptsIncomplete {
		return c.JSON(http.StatusUnprocessableEntity, map[string]any{
			"error":       "AsyncRequired",
			"description": "This service plan requires client support for asynchronous service operations",
		})
	}

	logger.Info("starting async provisioning for instance %s with plan %s", instanceId, req.PlanID)

	_, err = b.client.CreateCluster(context.Background(), instanceId, req.ServiceID, req.PlanID)
	if err != nil {
		if strings.Contains(err.Error(), "already exists") {
			logger.Info("instance %s was created concurrently", instanceId)
			return c.JSON(http.StatusAccepted, map[string]any{})
		}
		logger.Error("failed to start provisioning for instance %s: %v", instanceId, err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	logger.Info("provisioning initiated for instance %s", instanceId)
	response := c.Response()
	response.Header().Set("Retry-After", "10")
	return c.JSON(http.StatusAccepted, map[string]any{})
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
	acceptsIncomplete := c.QueryParam("accepts_incomplete") == "true"

	if err := validation.ValidateInstanceID(instanceId); err != nil {
		logger.Warn("invalid instance_id: %v", err)
		return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}

	nsStatus, err := b.client.GetNamespaceStatus(context.Background(), instanceId)
	if err != nil {
		logger.Error("failed to check namespace status for %s: %v", instanceId, err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	if !nsStatus.Exists {
		logger.Warn("attempted to deprovision non-existent instance %s", instanceId)
		return c.JSON(http.StatusGone, map[string]any{})
	}
	if nsStatus.IsTerminating {
		if acceptsIncomplete {
			logger.Info("instance %s deprovision in progress", instanceId)
			return c.JSON(http.StatusAccepted, map[string]any{})
		}
		return c.JSON(http.StatusUnprocessableEntity, map[string]any{
			"error":       "AsyncRequired",
			"description": "Service instance deprovision is in progress and requires async support",
		})
	}

	if !acceptsIncomplete {
		return c.JSON(http.StatusUnprocessableEntity, map[string]any{
			"error":       "AsyncRequired",
			"description": "This service plan requires client support for asynchronous service operations",
		})
	}

	logger.Info("starting async deprovision for instance %s", instanceId)
	err = b.client.DeleteCluster(context.Background(), instanceId)
	if err != nil {
		logger.Error("failed to start deprovision for instance %s: %v", instanceId, err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	logger.Info("deprovision initiated for instance %s", instanceId)
	response := c.Response()
	response.Header().Set("Retry-After", "10")
	return c.JSON(http.StatusAccepted, map[string]any{})
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

func (b *Broker) LastOperation(c echo.Context) error {
	instanceID := c.Param("instance_id")

	if err := validation.ValidateInstanceID(instanceID); err != nil {
		logger.Warn("invalid instance_id: %v", err)
		return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}

	logger.Debug("checking last operation for instance %s", instanceID)
	nsStatus, err := b.client.GetNamespaceStatus(context.Background(), instanceID)
	if err != nil {
		logger.Error("failed to check namespace status for %s: %v", instanceID, err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	if !nsStatus.Exists {
		logger.Debug("namespace for instance %s does not exist", instanceID)
		return c.JSON(http.StatusGone, map[string]any{
			"state":       "succeeded",
			"description": "Service instance has been deleted",
		})
	}
	if nsStatus.IsTerminating {
		logger.Debug("namespace for instance %s is terminating", instanceID)
		response := c.Response()
		response.Header().Set("Retry-After", "10")
		return c.JSON(http.StatusOK, map[string]any{
			"state":       "in progress",
			"description": "Deprovision in progress - namespace terminating",
		})
	}

	clusterStatus, err := b.client.GetClusterStatus(context.Background(), instanceID)
	if err != nil {
		logger.Error("failed to check cluster status for %s: %v", instanceID, err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	if !clusterStatus.Exists {
		logger.Debug("cluster for instance %s does not exist yet", instanceID)
		response := c.Response()
		response.Header().Set("Retry-After", "10")
		return c.JSON(http.StatusOK, map[string]any{
			"state":       "in progress",
			"description": "Provisioning in progress - waiting for cluster creation",
		})
	}
	if clusterStatus.IsFailed {
		logger.Warn("cluster for instance %s is in failed state: %s", instanceID, clusterStatus.FailureReason)
		return c.JSON(http.StatusOK, map[string]any{
			"state":       "failed",
			"description": fmt.Sprintf("Operation failed: %s", clusterStatus.FailureReason),
		})
	}
	if clusterStatus.IsReady {
		servicesReady, err := b.client.CheckServicesReady(context.Background(), instanceID)
		if err != nil {
			logger.Error("failed to check services for %s: %v", instanceID, err)
			servicesReady = true
		}
		if !servicesReady {
			logger.Debug("cluster for instance %s is ready but services not ready", instanceID)
			response := c.Response()
			response.Header().Set("Retry-After", "10")
			return c.JSON(http.StatusOK, map[string]any{
				"state":       "in progress",
				"description": fmt.Sprintf("Provisioning in progress - %d/%d instances ready, waiting for services",
					clusterStatus.Ready, clusterStatus.Total),
			})
		}

		logger.Debug("cluster for instance %s is fully ready", instanceID)
		return c.JSON(http.StatusOK, map[string]any{
			"state":       "succeeded",
			"description": fmt.Sprintf("Operation succeeded - %d/%d instances ready",
				clusterStatus.Ready, clusterStatus.Total),
		})
	}

	logger.Debug("cluster for instance %s is provisioning: %d/%d instances ready",
		instanceID, clusterStatus.Ready, clusterStatus.Total)
	response := c.Response()
	response.Header().Set("Retry-After", "10")
	return c.JSON(http.StatusOK, map[string]any{
		"state":       "in progress",
		"description": fmt.Sprintf("Operation in progress - %d/%d instances ready",
			clusterStatus.Ready, clusterStatus.Total),
	})
}

func (b *Broker) UpdateInstance(c echo.Context) error {
	instanceId := c.Param("instance_id")
	acceptsIncomplete := c.QueryParam("accepts_incomplete") == "true"

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
		logger.Error("failed to parse update request for %s: %v", instanceId, err)
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

	clusterStatus, err := b.client.GetClusterStatus(context.Background(), instanceId)
	if err != nil {
		logger.Error("failed to check cluster status for %s: %v", instanceId, err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	if !clusterStatus.Exists {
		logger.Warn("attempted to update non-existent instance %s", instanceId)
		return c.JSON(http.StatusNotFound, map[string]string{"error": "instance not found"})
	}

	existing, err := b.client.GetCluster(context.Background(), instanceId)
	if err != nil {
		logger.Error("failed to get instance %s: %v", instanceId, err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	if len(existing.ServiceID) > 0 && existing.ServiceID != req.ServiceID {
		logger.Warn("cannot change service_id for %s: %s -> %s", instanceId, existing.ServiceID, req.ServiceID)
		return c.JSON(http.StatusUnprocessableEntity, map[string]string{
			"error": "cannot change service_id",
		})
	}

	newInstances, newCPU, newMemory, newStorage := catalog.PlanSpec(req.PlanID)
	if newInstances < existing.Instances {
		logger.Warn("cannot downgrade number of instances for %s: %d -> %d", instanceId, existing.Instances, newInstances)
		return c.JSON(http.StatusUnprocessableEntity, map[string]string{
			"error": "cannot decrease number of instances",
		})
	}

	existingStorage := parseStorage(existing.Storage)
	newStorageBytes := parseStorage(newStorage)
	if newStorageBytes < existingStorage {
		logger.Warn("cannot downgrade storage for %s: %s -> %s", instanceId, existing.Storage, newStorage)
		return c.JSON(http.StatusUnprocessableEntity, map[string]string{
			"error": "cannot decrease storage size",
		})
	}

	if clusterStatus.SpecPlanID == req.PlanID && clusterStatus.IsReady {
		logger.Info("instance %s already at target plan %s and ready", instanceId, req.PlanID)
		return c.JSON(http.StatusOK, map[string]any{})
	}
	if clusterStatus.SpecPlanID == req.PlanID && clusterStatus.IsProvisioning {
		if acceptsIncomplete {
			logger.Info("instance %s update to plan %s in progress", instanceId, req.PlanID)
			return c.JSON(http.StatusAccepted, map[string]any{})
		}
		return c.JSON(http.StatusUnprocessableEntity, map[string]any{
			"error":       "AsyncRequired",
			"description": "Service instance update is in progress and requires async support",
		})
	}

	if !acceptsIncomplete {
		return c.JSON(http.StatusUnprocessableEntity, map[string]any{
			"error":       "AsyncRequired",
			"description": "This service plan requires client support for asynchronous service operations",
		})
	}

	logger.Info("starting async update for instance %s to plan %s", instanceId, req.PlanID)
	if err := b.client.UpdateCluster(context.Background(), instanceId, req.PlanID,
		newInstances, newCPU, newMemory, newStorage); err != nil {
		logger.Error("failed to start update for instance %s: %v", instanceId, err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	logger.Info("update initiated for instance %s", instanceId)
	response := c.Response()
	response.Header().Set("Retry-After", "10")
	return c.JSON(http.StatusAccepted, map[string]any{})
}

func parseStorage(storage string) int64 {
	if storage == "" {
		return 0
	}
	var multiplier int64 = 1
	size := storage
	if strings.HasSuffix(storage, "Gi") {
		multiplier = 1024 * 1024 * 1024
		size = strings.TrimSuffix(storage, "Gi")
	} else if strings.HasSuffix(storage, "Mi") {
		multiplier = 1024 * 1024
		size = strings.TrimSuffix(storage, "Mi")
	} else if strings.HasSuffix(storage, "Ki") {
		multiplier = 1024
		size = strings.TrimSuffix(storage, "Ki")
	} else if strings.HasSuffix(storage, "G") {
		multiplier = 1000 * 1000 * 1000
		size = strings.TrimSuffix(storage, "G")
	} else if strings.HasSuffix(storage, "M") {
		multiplier = 1000 * 1000
		size = strings.TrimSuffix(storage, "M")
	} else if strings.HasSuffix(storage, "K") {
		multiplier = 1000
		size = strings.TrimSuffix(storage, "K")
	}

	var value int64
	fmt.Sscanf(size, "%d", &value)
	return value * multiplier
}
