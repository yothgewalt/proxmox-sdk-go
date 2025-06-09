package internal_test

import (
	"sync"
	"testing"
	"time"

	"github.com/valyala/fasthttp"

	"yoth.dev/proxmox-sdk-go/internal"
)

func TestWithHttpClient(t *testing.T) {
	customClient := &fasthttp.Client{
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	instance := internal.NewHttpInstance(
		internal.WithHttpClient(customClient),
	)

	if instance.HttpClient != customClient {
		t.Error("Expected custom HttpClient to be set")
	}

	// Verify that the custom client properties are preserved
	if instance.HttpClient.ReadTimeout != 15*time.Second {
		t.Errorf("Expected ReadTimeout 15s, got %v", instance.HttpClient.ReadTimeout)
	}

	if instance.HttpClient.WriteTimeout != 10*time.Second {
		t.Errorf("Expected WriteTimeout 10s, got %v", instance.HttpClient.WriteTimeout)
	}
}

func TestWithWorkerPoolSize(t *testing.T) {
	tests := []struct {
		name string
		size int
	}{
		{"small pool", 10},
		{"medium pool", 50},
		{"large pool", 200},
		{"single worker", 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			instance := internal.NewHttpInstance(
				internal.WithWorkerPoolSize(tt.size),
			)

			// We can't directly test the worker pool size since it's not exposed,
			// but we can test that the option doesn't panic and the instance is created
			if instance == nil {
				t.Fatal("Expected non-nil HttpInstance")
			}

			// Test that the instance can still make requests after setting worker pool size
			if instance.HttpClient == nil {
				t.Error("Expected non-nil HttpClient after setting worker pool size")
			}
		})
	}
}

func TestWithBaseEndpointPath(t *testing.T) {
	tests := []struct {
		name     string
		basePath string
	}{
		{"custom api path", "/custom/api"},
		{"proxmox path", "/api2/json"},
		{"root path", "/"},
		{"nested path", "/api/v1/proxmox"},
		{"path with trailing slash", "/api/v2/"},
		{"empty path", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			instance := internal.NewHttpInstance(
				internal.WithBaseEndpointPath(tt.basePath),
			)

			if instance.GetBaseEndpointPath() != tt.basePath {
				t.Errorf("Expected base endpoint path '%s', got '%s'", tt.basePath, instance.GetBaseEndpointPath())
			}
		})
	}
}

func TestMultipleOptions(t *testing.T) {
	customClient := &fasthttp.Client{
		ReadTimeout:  20 * time.Second,
		WriteTimeout: 15 * time.Second,
	}

	customBasePath := "/api/v3/proxmox"
	workerPoolSize := 75

	instance := internal.NewHttpInstance(
		internal.WithHttpClient(customClient),
		internal.WithBaseEndpointPath(customBasePath),
		internal.WithWorkerPoolSize(workerPoolSize),
	)

	// Test that all options are applied correctly
	if instance.HttpClient != customClient {
		t.Error("Expected custom HttpClient to be set")
	}

	if instance.GetBaseEndpointPath() != customBasePath {
		t.Errorf("Expected base endpoint path '%s', got '%s'", customBasePath, instance.GetBaseEndpointPath())
	}

	// Verify client properties
	if instance.HttpClient.ReadTimeout != 20*time.Second {
		t.Errorf("Expected ReadTimeout 20s, got %v", instance.HttpClient.ReadTimeout)
	}

	if instance.HttpClient.WriteTimeout != 15*time.Second {
		t.Errorf("Expected WriteTimeout 15s, got %v", instance.HttpClient.WriteTimeout)
	}
}

func TestOptionOrder(t *testing.T) {
	// Test that options applied later override earlier ones
	firstClient := &fasthttp.Client{
		ReadTimeout: 10 * time.Second,
	}

	secondClient := &fasthttp.Client{
		ReadTimeout: 20 * time.Second,
	}

	instance := internal.NewHttpInstance(
		internal.WithHttpClient(firstClient),
		internal.WithBaseEndpointPath("/first/api"),
		internal.WithHttpClient(secondClient),
		internal.WithBaseEndpointPath("/second/api"),
	)

	// The last options should take precedence
	if instance.HttpClient != secondClient {
		t.Error("Expected second HttpClient to be set (last option should win)")
	}

	if instance.GetBaseEndpointPath() != "/second/api" {
		t.Errorf("Expected base endpoint path '/second/api', got '%s'", instance.GetBaseEndpointPath())
	}
}

func TestOptionWithNilClient(t *testing.T) {
	// Test behavior when passing nil client
	instance := internal.NewHttpInstance(
		internal.WithHttpClient(nil),
	)

	if instance.HttpClient != nil {
		t.Error("Expected HttpClient to be nil when explicitly set to nil")
	}
}

func TestEmptyOptionsArray(t *testing.T) {
	// Test that passing no options works correctly
	instance := internal.NewHttpInstance()

	if instance == nil {
		t.Fatal("Expected non-nil HttpInstance with no options")
	}

	if instance.HttpClient == nil {
		t.Error("Expected default HttpClient to be created")
	}

	if instance.GetBaseEndpointPath() != "/api2/json" {
		t.Errorf("Expected default base endpoint path '/api2/json', got '%s'", instance.GetBaseEndpointPath())
	}
}

func TestOptionFunctionImplementation(t *testing.T) {
	// Test that option functions implement the Option interface correctly
	var option internal.Option[internal.HttpInstnace]

	// Test WithHttpClient
	customClient := &fasthttp.Client{}
	option = internal.WithHttpClient(customClient)
	if option == nil {
		t.Error("Expected non-nil option from WithHttpClient")
	}

	// Test WithWorkerPoolSize
	option = internal.WithWorkerPoolSize(100)
	if option == nil {
		t.Error("Expected non-nil option from WithWorkerPoolSize")
	}

	// Test WithBaseEndpointPath
	option = internal.WithBaseEndpointPath("/test")
	if option == nil {
		t.Error("Expected non-nil option from WithBaseEndpointPath")
	}
}

func TestConcurrentOptionApplication(t *testing.T) {
	// Test that options can be applied safely in concurrent scenarios
	// This mainly tests that there are no data races

	customClient := &fasthttp.Client{
		ReadTimeout: 25 * time.Second,
	}

	// Create multiple instances concurrently with the same options
	instances := make([]*internal.HttpInstnace, 10)
	var wg sync.WaitGroup

	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()
			instances[index] = internal.NewHttpInstance(
				internal.WithHttpClient(customClient),
				internal.WithBaseEndpointPath("/concurrent/api"),
				internal.WithWorkerPoolSize(50),
			)
		}(i)
	}

	// Wait for all goroutines to complete
	wg.Wait()

	// Verify all instances were created correctly
	for i, instance := range instances {
		if instance == nil {
			t.Errorf("Expected non-nil instance at index %d", i)
			continue
		}

		if instance.HttpClient != customClient {
			t.Errorf("Expected custom HttpClient for instance %d", i)
		}

		if instance.GetBaseEndpointPath() != "/concurrent/api" {
			t.Errorf("Expected base endpoint path '/concurrent/api' for instance %d, got '%s'", i, instance.GetBaseEndpointPath())
		}
	}
}
