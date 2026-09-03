package domain

import (
	"errors"
	"fmt"
	"regexp"
	"slices"
	"strings"
)

var serviceNamePattern = regexp.MustCompile(`^[a-z][a-z0-9-]{1,61}[a-z0-9]$`)

var allowedTemplates = []string{"go-http@1.0.0", "python-http@1.0.0", "node-http@1.0.0"}
var allowedEnvironments = []string{"dev", "staging", "production"}
var allowedResources = []string{"postgres", "redis"}

func ValidateDescriptor(descriptor ServiceDescriptor) error {
	if descriptor.APIVersion != "platform.demo/v1" {
		return errors.New("apiVersion must be platform.demo/v1")
	}
	if descriptor.Kind != "Service" {
		return errors.New("kind must be Service")
	}
	if !serviceNamePattern.MatchString(descriptor.Metadata.Name) {
		return errors.New("metadata.name must be a lowercase DNS label between 3 and 63 characters")
	}
	if strings.TrimSpace(descriptor.Metadata.Owner) == "" {
		return errors.New("metadata.owner is required")
	}
	if !slices.Contains(allowedTemplates, descriptor.Spec.Template) {
		return fmt.Errorf("unsupported template %q", descriptor.Spec.Template)
	}
	if descriptor.Spec.Runtime.Port < 1 || descriptor.Spec.Runtime.Port > 65535 {
		return errors.New("spec.runtime.port must be between 1 and 65535")
	}
	if descriptor.Spec.Runtime.Replicas < 1 || descriptor.Spec.Runtime.Replicas > 10 {
		return errors.New("spec.runtime.replicas must be between 1 and 10")
	}
	if len(descriptor.Spec.Environments) == 0 {
		return errors.New("at least one environment is required")
	}
	seen := map[string]bool{}
	for _, environment := range descriptor.Spec.Environments {
		if !slices.Contains(allowedEnvironments, environment) {
			return fmt.Errorf("unsupported environment %q", environment)
		}
		if seen[environment] {
			return fmt.Errorf("duplicate environment %q", environment)
		}
		seen[environment] = true
	}
	for _, resource := range descriptor.Spec.Resources {
		if !slices.Contains(allowedResources, resource.Type) {
			return fmt.Errorf("unsupported resource type %q", resource.Type)
		}
		if resource.Plan != "small" && resource.Plan != "medium" {
			return fmt.Errorf("unsupported plan %q for resource %q", resource.Plan, resource.Type)
		}
	}
	if descriptor.Spec.Observability.AvailabilitySLO < 90 || descriptor.Spec.Observability.AvailabilitySLO > 100 {
		return errors.New("availabilitySLO must be between 90 and 100")
	}
	return nil
}

func Can(role Role, action string) bool {
	permissions := map[Role]map[string]bool{
		RoleDeveloper: {
			"service:create": true, "service:read": true, "operation:read": true, "operation:retry": true,
		},
		RoleServiceOwner: {
			"service:create": true, "service:read": true, "operation:read": true, "operation:retry": true,
		},
		RolePlatformAdmin: {
			"service:create": true, "service:read": true, "operation:read": true, "operation:retry": true, "audit:read": true,
		},
		RoleAuditor: {
			"service:read": true, "operation:read": true, "audit:read": true,
		},
	}
	return permissions[role][action]
}
