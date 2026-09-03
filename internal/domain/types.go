package domain

import "time"

type Role string

const (
	RoleDeveloper     Role = "developer"
	RoleServiceOwner  Role = "service_owner"
	RolePlatformAdmin Role = "platform_admin"
	RoleAuditor       Role = "auditor"
)

type RuntimeSpec struct {
	Port     int `json:"port"`
	Replicas int `json:"replicas"`
}

type ResourceSpec struct {
	Type string `json:"type"`
	Plan string `json:"plan"`
}

type ObservabilitySpec struct {
	AvailabilitySLO float64 `json:"availabilitySLO"`
}

type ServiceSpec struct {
	Template      string            `json:"template"`
	Runtime       RuntimeSpec       `json:"runtime"`
	Resources     []ResourceSpec    `json:"resources"`
	Environments  []string          `json:"environments"`
	Observability ObservabilitySpec `json:"observability"`
}

type Metadata struct {
	Name  string `json:"name"`
	Owner string `json:"owner"`
}

type ServiceDescriptor struct {
	APIVersion string      `json:"apiVersion"`
	Kind       string      `json:"kind"`
	Metadata   Metadata    `json:"metadata"`
	Spec       ServiceSpec `json:"spec"`
}

type ServiceStatus string

const (
	ServiceProvisioning ServiceStatus = "PROVISIONING"
	ServiceReady        ServiceStatus = "READY"
	ServiceFailed       ServiceStatus = "FAILED"
)

type ServiceRecord struct {
	Descriptor    ServiceDescriptor `json:"descriptor"`
	Status        ServiceStatus     `json:"status"`
	RepositoryURL string            `json:"repositoryUrl,omitempty"`
	GitOpsPath    string            `json:"gitOpsPath,omitempty"`
	DashboardURL  string            `json:"dashboardUrl,omitempty"`
	CreatedAt     time.Time         `json:"createdAt"`
	UpdatedAt     time.Time         `json:"updatedAt"`
}

type OperationStatus string

const (
	OperationPending    OperationStatus = "PENDING"
	OperationValidating OperationStatus = "VALIDATING"
	OperationPlanning   OperationStatus = "PLANNING"
	OperationApplying   OperationStatus = "APPLYING"
	OperationVerifying  OperationStatus = "VERIFYING"
	OperationSucceeded  OperationStatus = "SUCCEEDED"
	OperationFailed     OperationStatus = "FAILED"
	OperationRetrying   OperationStatus = "RETRYING"
)

type StepStatus string

const (
	StepPending   StepStatus = "PENDING"
	StepRunning   StepStatus = "RUNNING"
	StepSucceeded StepStatus = "SUCCEEDED"
	StepFailed    StepStatus = "FAILED"
)

type OperationStep struct {
	Name       string     `json:"name"`
	Status     StepStatus `json:"status"`
	Attempt    int        `json:"attempt"`
	StartedAt  *time.Time `json:"startedAt,omitempty"`
	FinishedAt *time.Time `json:"finishedAt,omitempty"`
	Error      string     `json:"error,omitempty"`
}

type Operation struct {
	ID             string            `json:"id"`
	Type           string            `json:"type"`
	Target         string            `json:"target"`
	IdempotencyKey string            `json:"idempotencyKey"`
	RequestHash    string            `json:"requestHash"`
	Status         OperationStatus   `json:"status"`
	Steps          []OperationStep   `json:"steps"`
	Attempt        int               `json:"attempt"`
	Error          string            `json:"error,omitempty"`
	Links          map[string]string `json:"links,omitempty"`
	CreatedAt      time.Time         `json:"createdAt"`
	UpdatedAt      time.Time         `json:"updatedAt"`
}

type AuditEvent struct {
	ID        string    `json:"id"`
	Actor     string    `json:"actor"`
	Action    string    `json:"action"`
	Target    string    `json:"target"`
	Reason    string    `json:"reason"`
	Before    any       `json:"before,omitempty"`
	After     any       `json:"after,omitempty"`
	CreatedAt time.Time `json:"createdAt"`
}

type CreateResult struct {
	Service   ServiceRecord `json:"service"`
	Operation Operation     `json:"operation"`
	Replayed  bool          `json:"replayed"`
}
