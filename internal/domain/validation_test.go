package domain

import "testing"

func validDescriptor() ServiceDescriptor {
	return ServiceDescriptor{
		APIVersion: "platform.demo/v1", Kind: "Service", Metadata: Metadata{Name: "payments-notifier", Owner: "team-payments"},
		Spec: ServiceSpec{Template: "go-http@1.0.0", Runtime: RuntimeSpec{Port: 8080, Replicas: 2}, Resources: []ResourceSpec{{Type: "postgres", Plan: "small"}}, Environments: []string{"dev", "staging", "production"}, Observability: ObservabilitySpec{AvailabilitySLO: 99.9}},
	}
}

func TestValidateDescriptor(t *testing.T) {
	if err := ValidateDescriptor(validDescriptor()); err != nil {
		t.Fatalf("valid descriptor rejected: %v", err)
	}
	invalid := validDescriptor()
	invalid.Metadata.Name = "Invalid_Name"
	if err := ValidateDescriptor(invalid); err == nil {
		t.Fatal("invalid name accepted")
	}
}

func TestRBAC(t *testing.T) {
	if !Can(RoleDeveloper, "service:create") {
		t.Fatal("developer should create services")
	}
	if Can(RoleDeveloper, "audit:read") {
		t.Fatal("developer must not read audit events")
	}
	if Can(RoleAuditor, "operation:retry") {
		t.Fatal("auditor must not mutate operations")
	}
}
