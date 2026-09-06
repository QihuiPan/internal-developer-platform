package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/user"
	"strings"
	"time"

	"github.com/QihuiPan/internal-developer-platform/internal/domain"
)

var version = "dev"
var commit = "none"
var buildDate = "unknown"

type clientFlags struct {
	address *string
	actor   *string
	role    *string
	token   *string
}

type client struct {
	address string
	actor   string
	role    string
	token   string
	http    *http.Client
}

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(2)
	}
	var err error
	switch os.Args[1] {
	case "create":
		err = create(os.Args[2:])
	case "list":
		err = list(os.Args[2:])
	case "get":
		err = get(os.Args[2:])
	case "download":
		err = download(os.Args[2:])
	case "operations":
		err = operations(os.Args[2:])
	case "operation":
		err = operation(os.Args[2:])
	case "retry":
		err = retry(os.Args[2:])
	case "audit":
		err = audit(os.Args[2:])
	case "version", "--version", "-version":
		fmt.Printf("platformctl %s (commit %s, built %s)\n", version, commit, buildDate)
		return
	case "help", "--help", "-h":
		usage()
		return
	default:
		err = fmt.Errorf("unknown command %q", os.Args[1])
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error:", err)
		os.Exit(1)
	}
}

func create(arguments []string) error {
	set, flags := commandFlags("create")
	file := set.String("file", "", "Descriptor JSON file, or - for standard input")
	name := set.String("name", "payments-notifier", "Service name when --file is not used")
	owner := set.String("owner", "team-payments", "Owning team when --file is not used")
	template := set.String("template", "go-http@1.0.0", "Service template when --file is not used")
	wait := set.Bool("wait", true, "Wait for the operation to finish")
	timeout := set.Duration("timeout", 60*time.Second, "Maximum wait time")
	idempotencyKey := set.String("idempotency-key", "", "Stable retry key; generated when omitted")
	if err := set.Parse(arguments); err != nil {
		return err
	}
	if set.NArg() != 0 {
		return errors.New("create does not accept positional arguments")
	}
	var descriptor domain.ServiceDescriptor
	if *file != "" {
		data, err := readInput(*file)
		if err != nil {
			return err
		}
		decoder := json.NewDecoder(bytes.NewReader(data))
		decoder.DisallowUnknownFields()
		if err := decoder.Decode(&descriptor); err != nil {
			return fmt.Errorf("decode descriptor: %w", err)
		}
	} else {
		descriptor = defaultDescriptor(*name, *owner, *template)
	}
	data, err := json.Marshal(descriptor)
	if err != nil {
		return err
	}
	key := *idempotencyKey
	if key == "" {
		key = fmt.Sprintf("platformctl-%s-%d", descriptor.Metadata.Name, time.Now().UnixNano())
	}
	c := flags.client()
	response, err := c.request(http.MethodPost, "/v1/services", data, map[string]string{"Idempotency-Key": key})
	if err != nil {
		return err
	}
	var result domain.CreateResult
	if err := json.Unmarshal(response, &result); err != nil {
		return fmt.Errorf("decode create response: %w", err)
	}
	if !*wait {
		return printJSON(result)
	}
	completed, err := c.waitForOperation(result.Operation.ID, *timeout)
	if err != nil {
		return err
	}
	if err := printJSON(completed); err != nil {
		return err
	}
	if completed.Status == domain.OperationFailed {
		return fmt.Errorf("operation %s failed: %s", completed.ID, completed.Error)
	}
	return nil
}

func list(arguments []string) error {
	set, flags := commandFlags("list")
	if err := parseNoArguments(set, arguments); err != nil {
		return err
	}
	data, err := flags.client().request(http.MethodGet, "/v1/services", nil, nil)
	return printResponse(data, err)
}

func get(arguments []string) error {
	set, flags := commandFlags("get")
	if err := set.Parse(arguments); err != nil {
		return err
	}
	if set.NArg() != 1 {
		return errors.New("usage: platformctl get [flags] SERVICE_NAME")
	}
	data, err := flags.client().request(http.MethodGet, "/v1/services/"+set.Arg(0), nil, nil)
	return printResponse(data, err)
}

func download(arguments []string) error {
	set, flags := commandFlags("download")
	output := set.String("output", "", "Output ZIP path; defaults to SERVICE_NAME.zip")
	force := set.Bool("force", false, "Replace an existing output file")
	if err := set.Parse(arguments); err != nil {
		return err
	}
	if set.NArg() != 1 {
		return errors.New("usage: platformctl download [flags] SERVICE_NAME")
	}
	name := set.Arg(0)
	path := *output
	if path == "" {
		path = name + ".zip"
	}
	data, err := flags.client().request(http.MethodGet, "/v1/services/"+name+"/archive", nil, nil)
	if err != nil {
		return err
	}
	fileFlags := os.O_WRONLY | os.O_CREATE | os.O_EXCL
	if *force {
		fileFlags = os.O_WRONLY | os.O_CREATE | os.O_TRUNC
	}
	file, err := os.OpenFile(path, fileFlags, 0o640)
	if err != nil {
		if errors.Is(err, os.ErrExist) {
			return fmt.Errorf("output %s already exists; use --force to replace it", path)
		}
		return fmt.Errorf("open archive: %w", err)
	}
	if _, err := file.Write(data); err != nil {
		_ = file.Close()
		return fmt.Errorf("write archive: %w", err)
	}
	if err := file.Close(); err != nil {
		return fmt.Errorf("close archive: %w", err)
	}
	fmt.Printf("Downloaded %s\n", path)
	return nil
}

func operations(arguments []string) error {
	set, flags := commandFlags("operations")
	if err := parseNoArguments(set, arguments); err != nil {
		return err
	}
	data, err := flags.client().request(http.MethodGet, "/v1/operations", nil, nil)
	return printResponse(data, err)
}

func operation(arguments []string) error {
	set, flags := commandFlags("operation")
	if err := set.Parse(arguments); err != nil {
		return err
	}
	if set.NArg() != 1 {
		return errors.New("usage: platformctl operation [flags] OPERATION_ID")
	}
	data, err := flags.client().request(http.MethodGet, "/v1/operations/"+set.Arg(0), nil, nil)
	return printResponse(data, err)
}

func retry(arguments []string) error {
	set, flags := commandFlags("retry")
	reason := set.String("reason", "Operator requested retry", "Required audit reason")
	wait := set.Bool("wait", true, "Wait for the operation to finish")
	timeout := set.Duration("timeout", 60*time.Second, "Maximum wait time")
	if err := set.Parse(arguments); err != nil {
		return err
	}
	if set.NArg() != 1 {
		return errors.New("usage: platformctl retry [flags] OPERATION_ID")
	}
	body, _ := json.Marshal(map[string]string{"reason": *reason})
	c := flags.client()
	data, err := c.request(http.MethodPost, "/v1/operations/"+set.Arg(0)+"/retry", body, nil)
	if err != nil {
		return err
	}
	var value domain.Operation
	if err := json.Unmarshal(data, &value); err != nil {
		return err
	}
	if *wait {
		value, err = c.waitForOperation(value.ID, *timeout)
		if err != nil {
			return err
		}
	}
	if err := printJSON(value); err != nil {
		return err
	}
	if value.Status == domain.OperationFailed {
		return fmt.Errorf("operation %s failed again: %s", value.ID, value.Error)
	}
	return nil
}

func audit(arguments []string) error {
	set, flags := commandFlags("audit")
	if err := parseNoArguments(set, arguments); err != nil {
		return err
	}
	data, err := flags.client().request(http.MethodGet, "/v1/audit-events", nil, nil)
	return printResponse(data, err)
}

func commandFlags(name string) (*flag.FlagSet, clientFlags) {
	set := flag.NewFlagSet(name, flag.ContinueOnError)
	set.SetOutput(os.Stderr)
	flags := clientFlags{
		address: set.String("address", environment("PLATFORM_API_URL", "http://127.0.0.1:8080"), "Platform API base URL"),
		actor:   set.String("actor", environment("PLATFORM_ACTOR", currentUser()), "Audit actor"),
		role:    set.String("role", environment("PLATFORM_ROLE", "developer"), "Role used in demo authentication mode"),
		token:   set.String("token", os.Getenv("PLATFORM_API_TOKEN"), "Bearer token; preferably set PLATFORM_API_TOKEN"),
	}
	return set, flags
}

func (flags clientFlags) client() *client {
	return &client{
		address: strings.TrimRight(*flags.address, "/"),
		actor:   *flags.actor,
		role:    *flags.role,
		token:   *flags.token,
		http:    &http.Client{Timeout: 30 * time.Second},
	}
}

func (c *client) request(method, path string, body []byte, extraHeaders map[string]string) ([]byte, error) {
	request, err := http.NewRequest(method, c.address+path, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	request.Header.Set("Accept", "application/json")
	request.Header.Set("X-Actor", c.actor)
	request.Header.Set("X-Role", c.role)
	if len(body) > 0 {
		request.Header.Set("Content-Type", "application/json")
	}
	if c.token != "" {
		request.Header.Set("Authorization", "Bearer "+c.token)
	}
	for name, value := range extraHeaders {
		request.Header.Set(name, value)
	}
	response, err := c.http.Do(request)
	if err != nil {
		return nil, fmt.Errorf("request %s: %w", path, err)
	}
	defer response.Body.Close()
	data, err := io.ReadAll(io.LimitReader(response.Body, 64<<20))
	if err != nil {
		return nil, err
	}
	if response.StatusCode >= 400 {
		var remoteError struct {
			Code    string `json:"code"`
			Message string `json:"message"`
		}
		if json.Unmarshal(data, &remoteError) == nil && remoteError.Message != "" {
			return nil, fmt.Errorf("API returned HTTP %d (%s): %s", response.StatusCode, remoteError.Code, remoteError.Message)
		}
		return nil, fmt.Errorf("API returned HTTP %d: %s", response.StatusCode, strings.TrimSpace(string(data)))
	}
	return data, nil
}

func (c *client) waitForOperation(id string, timeout time.Duration) (domain.Operation, error) {
	deadline := time.Now().Add(timeout)
	for {
		data, err := c.request(http.MethodGet, "/v1/operations/"+id, nil, nil)
		if err != nil {
			return domain.Operation{}, err
		}
		var value domain.Operation
		if err := json.Unmarshal(data, &value); err != nil {
			return domain.Operation{}, err
		}
		if value.Status == domain.OperationSucceeded || value.Status == domain.OperationFailed {
			return value, nil
		}
		if time.Now().After(deadline) {
			return domain.Operation{}, fmt.Errorf("operation %s did not finish within %s", id, timeout)
		}
		time.Sleep(250 * time.Millisecond)
	}
}

func defaultDescriptor(name, owner, template string) domain.ServiceDescriptor {
	return domain.ServiceDescriptor{
		APIVersion: "platform.demo/v1",
		Kind:       "Service",
		Metadata:   domain.Metadata{Name: name, Owner: owner},
		Spec: domain.ServiceSpec{
			Template:      template,
			Runtime:       domain.RuntimeSpec{Port: 8080, Replicas: 2},
			Resources:     []domain.ResourceSpec{{Type: "postgres", Plan: "small"}},
			Environments:  []string{"dev", "staging", "production"},
			Observability: domain.ObservabilitySpec{AvailabilitySLO: 99.9},
		},
	}
}

func readInput(path string) ([]byte, error) {
	if path == "-" {
		return io.ReadAll(io.LimitReader(os.Stdin, 1<<20))
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read descriptor: %w", err)
	}
	return data, nil
}

func printResponse(data []byte, err error) error {
	if err != nil {
		return err
	}
	var value any
	if err := json.Unmarshal(data, &value); err != nil {
		return err
	}
	return printJSON(value)
}

func printJSON(value any) error {
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	return encoder.Encode(value)
}

func parseNoArguments(set *flag.FlagSet, arguments []string) error {
	if err := set.Parse(arguments); err != nil {
		return err
	}
	if set.NArg() != 0 {
		return fmt.Errorf("%s does not accept positional arguments", set.Name())
	}
	return nil
}

func currentUser() string {
	value, err := user.Current()
	if err == nil && value.Username != "" {
		parts := strings.FieldsFunc(value.Username, func(character rune) bool { return character == '\\' || character == '/' })
		return parts[len(parts)-1]
	}
	return "platformctl-user"
}

func environment(name, fallback string) string {
	if value := os.Getenv(name); value != "" {
		return value
	}
	return fallback
}

func usage() {
	fmt.Fprintln(os.Stderr, `platformctl manages an Internal Developer Platform instance.

Usage:
  platformctl COMMAND [flags]

Commands:
  create       Create a service and wait for reconciliation
  list         List service catalogue entries
  get          Get one service
  download     Download a generated service repository ZIP
  operations   List operations
  operation    Get one operation
  retry        Retry a failed operation
  audit        List audit events (platform_admin or auditor)
  version      Print build information

Run platformctl COMMAND -h for command flags.`)
}
