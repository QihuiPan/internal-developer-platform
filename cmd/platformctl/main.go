package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/QihuiPan/internal-developer-platform/internal/domain"
)

func main() {
	address := flag.String("address", "http://localhost:8080", "Platform API base URL")
	name := flag.String("name", "payments-notifier", "Service name")
	owner := flag.String("owner", "team-payments", "Owning team")
	flag.Parse()
	descriptor := domain.ServiceDescriptor{
		APIVersion: "platform.demo/v1", Kind: "Service", Metadata: domain.Metadata{Name: *name, Owner: *owner},
		Spec: domain.ServiceSpec{Template: "go-http@1.0.0", Runtime: domain.RuntimeSpec{Port: 8080, Replicas: 2}, Resources: []domain.ResourceSpec{{Type: "postgres", Plan: "small"}}, Environments: []string{"dev", "staging", "production"}, Observability: domain.ObservabilitySpec{AvailabilitySLO: 99.9}},
	}
	body, err := json.Marshal(descriptor)
	if err != nil {
		fatal(err)
	}
	request, err := http.NewRequest(http.MethodPost, *address+"/v1/services", bytes.NewReader(body))
	if err != nil {
		fatal(err)
	}
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Idempotency-Key", fmt.Sprintf("platformctl-%s-%d", *name, time.Now().Unix()))
	request.Header.Set("X-Actor", os.Getenv("USER"))
	if request.Header.Get("X-Actor") == "" {
		request.Header.Set("X-Actor", "platformctl-user")
	}
	request.Header.Set("X-Role", "developer")
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		fatal(err)
	}
	defer response.Body.Close()
	data, err := io.ReadAll(response.Body)
	if err != nil {
		fatal(err)
	}
	fmt.Printf("HTTP %d\n%s", response.StatusCode, data)
	if response.StatusCode >= 400 {
		os.Exit(1)
	}
}

func fatal(err error) { fmt.Fprintln(os.Stderr, err); os.Exit(1) }
