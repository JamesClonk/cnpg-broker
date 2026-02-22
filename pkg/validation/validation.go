package validation

import (
	"fmt"
	"regexp"

	"github.com/cnpg-broker/pkg/catalog"
)

var (
	uuidRegex    = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$`)
	k8sNameRegex = regexp.MustCompile(`^[a-z0-9]([-a-z0-9]*[a-z0-9])?$`)
)

type ValidationError struct {
	Field   string
	Message string
}

func (e *ValidationError) Error() string {
	return fmt.Sprintf("%s: %s", e.Field, e.Message)
}

func ValidateInstanceID(instanceId string) error {
	if instanceId == "" {
		return &ValidationError{"instance_id", "required"}
	}
	if len(instanceId) > 63 {
		return &ValidationError{"instance_id", "must be 63 characters or less"}
	}
	if !uuidRegex.MatchString(instanceId) {
		return &ValidationError{"instance_id", "must be valid UUID"}
	}
	if !k8sNameRegex.MatchString(instanceId) {
		return &ValidationError{"instance_id", "must be valid Kubernetes name (lowercase alphanumeric and hyphens)"}
	}
	return nil
}

func ValidateBindingID(bindingId string) error {
	if bindingId == "" {
		return &ValidationError{"binding_id", "required"}
	}
	if len(bindingId) > 63 {
		return &ValidationError{"binding_id", "must be 63 characters or less"}
	}
	if !uuidRegex.MatchString(bindingId) {
		return &ValidationError{"binding_id", "must be valid UUID"}
	}
	return nil
}

func ValidateServiceID(serviceId string) error {
	if serviceId == "" {
		return &ValidationError{"service_id", "required"}
	}
	if len(serviceId) > 63 {
		return &ValidationError{"service_id", "must be 63 characters or less"}
	}
	if !uuidRegex.MatchString(serviceId) {
		return &ValidationError{"service_id", "must be valid UUID"}
	}

	cat := catalog.GetCatalog()
	services := cat["services"].([]catalog.Service)
	for _, svc := range services {
		if svc.ID == serviceId {
			return nil
		}
	}
	return &ValidationError{"service_id", "not found in catalog"}
}

func ValidatePlanID(serviceId, planId string) error {
	if planId == "" {
		return &ValidationError{"plan_id", "required"}
	}
	if len(planId) > 63 {
		return &ValidationError{"plan_id", "must be 63 characters or less"}
	}
	if !uuidRegex.MatchString(planId) {
		return &ValidationError{"plan_id", "must be valid UUID"}
	}

	cat := catalog.GetCatalog()
	services := cat["services"].([]catalog.Service)
	for _, svc := range services {
		if svc.ID == serviceId {
			for _, plan := range svc.Plans {
				if plan.ID == planId {
					return nil
				}
			}
			return &ValidationError{"plan_id", "not found for this service"}
		}
	}
	return &ValidationError{"plan_id", "service not found"}
}
