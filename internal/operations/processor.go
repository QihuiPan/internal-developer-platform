package operations

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/QihuiPan/internal-developer-platform/internal/domain"
	"github.com/QihuiPan/internal-developer-platform/internal/store"
)

type Processor struct {
	store      *store.Store
	outputRoot string
	failAt     string
	queue      chan string
	stop       chan struct{}
	waitGroup  sync.WaitGroup
}

func NewProcessor(state *store.Store, outputRoot, failAt string) *Processor {
	return &Processor{store: state, outputRoot: outputRoot, failAt: strings.ToLower(failAt), queue: make(chan string, 100), stop: make(chan struct{})}
}

func (p *Processor) Start() {
	p.waitGroup.Add(1)
	go func() {
		defer p.waitGroup.Done()
		for {
			select {
			case operationID := <-p.queue:
				_ = p.Process(operationID)
			case <-p.stop:
				return
			}
		}
	}()
	for _, operationID := range p.store.PendingOperations() {
		p.Enqueue(operationID)
	}
}

func (p *Processor) Stop()                      { close(p.stop); p.waitGroup.Wait() }
func (p *Processor) Enqueue(operationID string) { p.queue <- operationID }

func (p *Processor) Process(operationID string) error {
	operation, err := p.store.Operation(operationID)
	if err != nil {
		return err
	}
	service, err := p.store.Service(operation.Target)
	if err != nil {
		return err
	}
	steps := []struct {
		name   string
		status domain.OperationStatus
		run    func(domain.ServiceRecord, domain.Operation) error
	}{
		{"validate", domain.OperationValidating, p.validate},
		{"plan", domain.OperationPlanning, p.plan},
		{"render", domain.OperationApplying, p.render},
		{"verify", domain.OperationVerifying, p.verify},
	}
	for _, step := range steps {
		current, lookupErr := p.store.Operation(operationID)
		if lookupErr != nil {
			return lookupErr
		}
		if stepComplete(current, step.name) {
			continue
		}
		if _, err := p.store.StartStep(operationID, step.name, step.status); err != nil {
			return err
		}
		if p.failAt == step.name {
			err = fmt.Errorf("fault injection requested at %s step", step.name)
		} else {
			err = step.run(service, operation)
		}
		if err != nil {
			_ = p.store.FailOperation(operationID, step.name, err.Error())
			return err
		}
		if err := p.store.CompleteStep(operationID, step.name); err != nil {
			return err
		}
	}
	root := filepath.Join(p.outputRoot, service.Descriptor.Metadata.Name)
	links := map[string]string{
		"repository": "file://" + filepath.ToSlash(root),
		"gitops":     filepath.ToSlash(filepath.Join(root, "deploy", "kubernetes.yaml")),
		"dashboard":  "http://localhost:3000/d/" + service.Descriptor.Metadata.Name,
	}
	return p.store.SucceedOperation(operationID, links)
}

func stepComplete(operation domain.Operation, name string) bool {
	for _, step := range operation.Steps {
		if step.Name == name {
			return step.Status == domain.StepSucceeded
		}
	}
	return false
}

func (p *Processor) validate(service domain.ServiceRecord, _ domain.Operation) error {
	return domain.ValidateDescriptor(service.Descriptor)
}

func (p *Processor) plan(service domain.ServiceRecord, operation domain.Operation) error {
	directory := filepath.Join(p.outputRoot, ".platform", "operations", operation.ID)
	if err := os.MkdirAll(directory, 0o750); err != nil {
		return err
	}
	plan := map[string]any{"service": service.Descriptor.Metadata.Name, "template": service.Descriptor.Spec.Template, "resources": service.Descriptor.Spec.Resources, "environments": service.Descriptor.Spec.Environments, "destructiveChanges": false}
	data, err := json.MarshalIndent(plan, "", "  ")
	if err != nil {
		return err
	}
	return writeAtomic(filepath.Join(directory, "plan.json"), data, 0o600)
}

func (p *Processor) render(service domain.ServiceRecord, _ domain.Operation) error {
	descriptor := service.Descriptor
	root := filepath.Join(p.outputRoot, descriptor.Metadata.Name)
	files := map[string]string{
		"README.md":                generatedReadme(descriptor),
		"service.yaml":             descriptorYAML(descriptor),
		"deploy/kubernetes.yaml":   generatedDeployment(descriptor),
		".github/workflows/ci.yml": generatedWorkflow(descriptor.Spec.Template),
		"CODEOWNERS":               fmt.Sprintf("* @%s\n", strings.TrimPrefix(descriptor.Metadata.Owner, "team-")),
	}
	switch descriptor.Spec.Template {
	case "go-http@1.0.0":
		files["cmd/server/main.go"] = generatedGoService(descriptor)
		files["go.mod"] = fmt.Sprintf("module generated.local/%s\n\ngo 1.26\n", descriptor.Metadata.Name)
		files["Dockerfile"] = "FROM golang:1.26-alpine AS build\nWORKDIR /src\nCOPY . .\nRUN CGO_ENABLED=0 go build -o /out/service ./cmd/server\nFROM gcr.io/distroless/static-debian12:nonroot\nCOPY --from=build /out/service /service\nUSER nonroot:nonroot\nENTRYPOINT [\"/service\"]\n"
	case "python-http@1.0.0":
		files["app.py"] = generatedPythonService(descriptor)
		files["Dockerfile"] = "FROM python:3.14-alpine\nWORKDIR /app\nCOPY app.py .\nUSER 65532:65532\nENTRYPOINT [\"python\", \"app.py\"]\n"
	case "node-http@1.0.0":
		files["server.js"] = generatedNodeService(descriptor)
		files["package.json"] = fmt.Sprintf("{\n  \"name\": \"%s\",\n  \"version\": \"1.0.0\",\n  \"private\": true,\n  \"scripts\": {\"start\": \"node server.js\", \"test\": \"node --check server.js\"}\n}\n", descriptor.Metadata.Name)
		files["Dockerfile"] = "FROM node:24-alpine\nWORKDIR /app\nCOPY --chown=node:node package.json server.js ./\nUSER node\nENTRYPOINT [\"node\", \"server.js\"]\n"
	}
	for relativePath, content := range files {
		path := filepath.Join(root, filepath.FromSlash(relativePath))
		if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
			return err
		}
		if err := writeAtomic(path, []byte(content), 0o640); err != nil {
			return err
		}
	}
	return nil
}

func (p *Processor) verify(service domain.ServiceRecord, _ domain.Operation) error {
	root := filepath.Join(p.outputRoot, service.Descriptor.Metadata.Name)
	required := []string{"README.md", "service.yaml", "Dockerfile", "deploy/kubernetes.yaml", ".github/workflows/ci.yml", "CODEOWNERS"}
	switch service.Descriptor.Spec.Template {
	case "go-http@1.0.0":
		required = append(required, "cmd/server/main.go", "go.mod")
	case "python-http@1.0.0":
		required = append(required, "app.py")
	case "node-http@1.0.0":
		required = append(required, "server.js", "package.json")
	}
	for _, relativePath := range required {
		info, err := os.Stat(filepath.Join(root, filepath.FromSlash(relativePath)))
		if err != nil {
			return fmt.Errorf("verify %s: %w", relativePath, err)
		}
		if info.Size() == 0 {
			return fmt.Errorf("verify %s: file is empty", relativePath)
		}
	}
	return nil
}

func writeAtomic(path string, data []byte, mode os.FileMode) error {
	temporary := path + ".tmp"
	if err := os.WriteFile(temporary, data, mode); err != nil {
		return err
	}
	if err := os.Rename(temporary, path); err != nil {
		if errors.Is(err, os.ErrExist) {
			_ = os.Remove(path)
			return os.Rename(temporary, path)
		}
		return err
	}
	return nil
}

func generatedReadme(descriptor domain.ServiceDescriptor) string {
	runCommand := map[string]string{"go-http@1.0.0": "go run ./cmd/server", "python-http@1.0.0": "python app.py", "node-http@1.0.0": "node server.js"}[descriptor.Spec.Template]
	return fmt.Sprintf("# %s\n\nGenerated by the Internal Developer Platform from `%s`.\n\nOwner: `%s`\n\n## Run\n\n```sh\n%s\n```\n\nThe service exposes `/healthz`, `/readyz`, and `/metrics`.\n", descriptor.Metadata.Name, descriptor.Spec.Template, descriptor.Metadata.Owner, runCommand)
}

func descriptorYAML(descriptor domain.ServiceDescriptor) string {
	resources := ""
	for _, resource := range descriptor.Spec.Resources {
		resources += fmt.Sprintf("    - type: %s\n      plan: %s\n", resource.Type, resource.Plan)
	}
	environments := ""
	for _, environment := range descriptor.Spec.Environments {
		environments += fmt.Sprintf("    - %s\n", environment)
	}
	resourceBlock := "  resources: []\n"
	if resources != "" {
		resourceBlock = "  resources:\n" + resources
	}
	return fmt.Sprintf("apiVersion: %s\nkind: %s\nmetadata:\n  name: %s\n  owner: %s\nspec:\n  template: %s\n  runtime:\n    port: %d\n    replicas: %d\n%s  environments:\n%s  observability:\n    availabilitySLO: %.3g\n", descriptor.APIVersion, descriptor.Kind, descriptor.Metadata.Name, descriptor.Metadata.Owner, descriptor.Spec.Template, descriptor.Spec.Runtime.Port, descriptor.Spec.Runtime.Replicas, resourceBlock, environments, descriptor.Spec.Observability.AvailabilitySLO)
}

func generatedGoService(descriptor domain.ServiceDescriptor) string {
	return fmt.Sprintf(`package main

import (
	"fmt"
	"log"
	"net/http"
)

func main() {
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) { _, _ = w.Write([]byte("ok")) })
	mux.HandleFunc("/readyz", func(w http.ResponseWriter, _ *http.Request) { _, _ = w.Write([]byte("ready")) })
	mux.HandleFunc("/metrics", func(w http.ResponseWriter, _ *http.Request) { _, _ = w.Write([]byte("service_up 1\n")) })
	address := fmt.Sprintf(":%%d", %d)
	log.Printf("service listening on %%s", address)
	log.Fatal(http.ListenAndServe(address, mux))
}
`, descriptor.Spec.Runtime.Port)
}

func generatedPythonService(descriptor domain.ServiceDescriptor) string {
	return fmt.Sprintf(`from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer


class Handler(BaseHTTPRequestHandler):
    def do_GET(self):
        responses = {"/healthz": "ok", "/readyz": "ready", "/metrics": "service_up 1\n"}
        if self.path not in responses:
            self.send_error(404)
            return
        body = responses[self.path].encode()
        self.send_response(200)
        self.send_header("Content-Type", "text/plain; charset=utf-8")
        self.send_header("Content-Length", str(len(body)))
        self.end_headers()
        self.wfile.write(body)

    def log_message(self, format, *args):
        return


ThreadingHTTPServer(("0.0.0.0", %d), Handler).serve_forever()
`, descriptor.Spec.Runtime.Port)
}

func generatedNodeService(descriptor domain.ServiceDescriptor) string {
	return fmt.Sprintf(`const http = require("node:http");

const responses = new Map([
  ["/healthz", "ok"],
  ["/readyz", "ready"],
  ["/metrics", "service_up 1\n"]
]);

http.createServer((request, response) => {
  if (!responses.has(request.url)) {
    response.writeHead(404).end("not found");
    return;
  }
  response.writeHead(200, {"Content-Type": "text/plain; charset=utf-8"});
  response.end(responses.get(request.url));
}).listen(%d, "0.0.0.0");
`, descriptor.Spec.Runtime.Port)
}

func generatedDeployment(descriptor domain.ServiceDescriptor) string {
	name := descriptor.Metadata.Name
	return fmt.Sprintf("apiVersion: apps/v1\nkind: Deployment\nmetadata:\n  name: %s\n  labels:\n    app.kubernetes.io/name: %s\nspec:\n  replicas: %d\n  selector:\n    matchLabels:\n      app.kubernetes.io/name: %s\n  template:\n    metadata:\n      labels:\n        app.kubernetes.io/name: %s\n    spec:\n      automountServiceAccountToken: false\n      containers:\n        - name: service\n          image: ghcr.io/example/%s@sha256:replace-me\n          ports:\n            - containerPort: %d\n          securityContext:\n            allowPrivilegeEscalation: false\n            readOnlyRootFilesystem: true\n            runAsNonRoot: true\n            capabilities:\n              drop: [\"ALL\"]\n          resources:\n            requests: {cpu: 50m, memory: 64Mi}\n            limits: {cpu: 500m, memory: 256Mi}\n          readinessProbe:\n            httpGet: {path: /readyz, port: %d}\n          livenessProbe:\n            httpGet: {path: /healthz, port: %d}\n---\napiVersion: networking.k8s.io/v1\nkind: NetworkPolicy\nmetadata:\n  name: %s-default-deny\nspec:\n  podSelector: {}\n  policyTypes: [Ingress, Egress]\n", name, name, descriptor.Spec.Runtime.Replicas, name, name, name, descriptor.Spec.Runtime.Port, descriptor.Spec.Runtime.Port, descriptor.Spec.Runtime.Port, name)
}

func generatedWorkflow(template string) string {
	setupAndTest := map[string]string{
		"go-http@1.0.0":     "      - uses: actions/setup-go@v6\n        with:\n          go-version: '1.26.x'\n      - run: go test ./...\n      - run: go vet ./...\n",
		"python-http@1.0.0": "      - uses: actions/setup-python@v6\n        with:\n          python-version: '3.14'\n      - run: python -m py_compile app.py\n",
		"node-http@1.0.0":   "      - uses: actions/setup-node@v5\n        with:\n          node-version: '24'\n      - run: npm test\n",
	}[template]
	return "name: CI\non:\n  pull_request:\n  push:\n    branches: [main]\npermissions:\n  contents: read\njobs:\n  test:\n    runs-on: ubuntu-latest\n    steps:\n      - uses: actions/checkout@v5\n" + setupAndTest
}
