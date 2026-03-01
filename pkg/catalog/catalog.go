package catalog

import (
	"os"

	"github.com/cnpg-broker/pkg/logger"
	"gopkg.in/yaml.v3"
)

var catalog Catalog

type Catalog struct {
	Services []Service `yaml:"services"`
}

type Service struct {
	ID                   string          `yaml:"id" json:"id"`
	Name                 string          `yaml:"name" json:"name"`
	Description          string          `yaml:"description" json:"description"`
	Bindable             bool            `yaml:"bindable" json:"bindable"`
	InstancesRetrievable bool            `yaml:"instances_retrievable" json:"instances_retrievable"`
	BindingsRetrievable  bool            `yaml:"bindings_retrievable" json:"bindings_retrievable"`
	PlanUpdateable       bool            `yaml:"plan_updateable" json:"plan_updateable"`
	Tags                 []string        `yaml:"tags" json:"tags,omitempty"`
	Metadata             ServiceMetadata `yaml:"metadata" json:"metadata"`
	Plans                []Plan          `yaml:"plans" json:"plans"`
}

type ServiceMetadata struct {
	DisplayName         string `yaml:"displayName" json:"displayName"`
	ImageUrl            string `yaml:"imageUrl" json:"imageUrl,omitempty"`
	LongDescription     string `yaml:"longDescription" json:"longDescription,omitempty"`
	ProviderDisplayName string `yaml:"providerDisplayName" json:"providerDisplayName,omitempty"`
	DocumentationUrl    string `yaml:"documentationUrl" json:"documentationUrl,omitempty"`
	SupportUrl          string `yaml:"supportUrl" json:"supportUrl,omitempty"`
}

type Plan struct {
	ID          string       `yaml:"id" json:"id"`
	Name        string       `yaml:"name" json:"name"`
	Description string       `yaml:"description" json:"description"`
	Free        bool         `yaml:"free" json:"free"`
	Metadata    PlanMetadata `yaml:"metadata" json:"metadata"`
}

type PlanMetadata struct {
	Instances        int64  `yaml:"instances" json:"instances"`
	CPU              string `yaml:"cpu" json:"cpu"`
	Memory           string `yaml:"memory" json:"memory"`
	Storage          string `yaml:"storage" json:"storage"`
	HighAvailability bool   `yaml:"highAvailability" json:"highAvailability"`
	SLA              bool   `yaml:"sla" json:"sla"`
}

func init() {
	data, err := os.ReadFile("catalog.yaml")
	if err != nil {
		logger.Fatal("failed to read catalog.yaml: %v", err)
	}
	if err := yaml.Unmarshal(data, &catalog); err != nil {
		logger.Fatal("failed to parse catalog.yaml: %v", err)
	}
	logger.Info("loaded catalog with %d service(s)", len(catalog.Services))
}

func GetCatalog() map[string]any {
	return map[string]any{"services": catalog.Services}
}

func PlanSpec(planId string) (int64, string, string, string) {
	for _, svc := range catalog.Services {
		for _, plan := range svc.Plans {
			if plan.ID == planId {
				logger.Debug("found plan %s: instances=%d, cpu=%s, memory=%s, storage=%s",
					planId, plan.Metadata.Instances, plan.Metadata.CPU,
					plan.Metadata.Memory, plan.Metadata.Storage)
				return plan.Metadata.Instances, plan.Metadata.CPU,
					plan.Metadata.Memory, plan.Metadata.Storage
			}
		}
	}
	logger.Warn("plan %s not found, using defaults", planId)
	return 3, "4", "4Gi", "250Gi"
}
