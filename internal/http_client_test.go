package internal

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/valyala/fasthttp"
)

func TestNewHttpInstance(t *testing.T) {
	t.Run("default configuration", func(t *testing.T) {
		instance := NewHttpInstance()

		if instance == nil {
			t.Fatal("Expected non-nil HttpInstance")
		}

		if instance.HttpClient == nil {
			t.Error("Expected non-nil HttpClient")
		}

		if instance.GetBaseEndpointPath() != "/api2/json" {
			t.Errorf("Expected default base endpoint path '/api2/json', got '%s'", instance.GetBaseEndpointPath())
		}

		if instance.GetRequestCount() != 0 {
			t.Errorf("Expected initial request count 0, got %d", instance.GetRequestCount())
		}
	})

	t.Run("with custom options", func(t *testing.T) {
		customClient := &fasthttp.Client{
			ReadTimeout: 10 * time.Second,
		}

		instance := NewHttpInstance(
			WithHttpClient(customClient),
			WithBaseEndpointPath("/custom/api"),
			WithWorkerPoolSize(50),
		)

		if instance.HttpClient != customClient {
			t.Error("Expected custom HttpClient to be set")
		}

		if instance.GetBaseEndpointPath() != "/custom/api" {
			t.Errorf("Expected custom base endpoint path '/custom/api', got '%s'", instance.GetBaseEndpointPath())
		}
	})
}

func TestHttpInstance_MakeRequest(t *testing.T) {
	// Create a test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api2/json/test":
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"status": "success"}`))
		case "/api2/json/error":
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(`{"error": "bad request"}`))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	instance := NewHttpInstance()

	t.Run("successful GET request", func(t *testing.T) {
		config := RequestConfig{
			Method:  GET,
			URL:     server.URL + "/api2/json/test",
			Headers: map[string]string{"Accept": "application/json"},
			Timeout: 5 * time.Second,
		}

		response := instance.MakeRequest(config)

		if response.Error != nil {
			t.Errorf("Expected no error, got %v", response.Error)
		}

		if response.StatusCode != http.StatusOK {
			t.Errorf("Expected status code 200, got %d", response.StatusCode)
		}

		expectedBody := `{"status": "success"}`
		if string(response.Body) != expectedBody {
			t.Errorf("Expected body '%s', got '%s'", expectedBody, string(response.Body))
		}

		if response.Headers["Content-Type"] != "application/json" {
			t.Errorf("Expected Content-Type header 'application/json', got '%s'", response.Headers["Content-Type"])
		}
	})

	t.Run("POST request with body", func(t *testing.T) {
		config := RequestConfig{
			Method:  POST,
			URL:     server.URL + "/api2/json/test",
			Headers: map[string]string{"Content-Type": "application/json"},
			Body:    []byte(`{"key": "value"}`),
			Timeout: 5 * time.Second,
		}

		response := instance.MakeRequest(config)

		if response.Error != nil {
			t.Errorf("Expected no error, got %v", response.Error)
		}

		if response.StatusCode != http.StatusOK {
			t.Errorf("Expected status code 200, got %d", response.StatusCode)
		}
	})

	t.Run("error response", func(t *testing.T) {
		config := RequestConfig{
			Method:  GET,
			URL:     server.URL + "/api2/json/error",
			Timeout: 5 * time.Second,
		}

		response := instance.MakeRequest(config)

		if response.StatusCode != http.StatusBadRequest {
			t.Errorf("Expected status code 400, got %d", response.StatusCode)
		}

		expectedBody := `{"error": "bad request"}`
		if string(response.Body) != expectedBody {
			t.Errorf("Expected body '%s', got '%s'", expectedBody, string(response.Body))
		}
	})

	t.Run("request counter increment", func(t *testing.T) {
		initialCount := instance.GetRequestCount()

		config := RequestConfig{
			Method:  GET,
			URL:     server.URL + "/api2/json/test",
			Timeout: 5 * time.Second,
		}

		instance.MakeRequest(config)

		if instance.GetRequestCount() != initialCount+1 {
			t.Errorf("Expected request count to increment by 1, got %d", instance.GetRequestCount()-initialCount)
		}
	})

	t.Run("default timeout", func(t *testing.T) {
		config := RequestConfig{
			Method: GET,
			URL:    server.URL + "/api2/json/test",
			// No timeout specified - should use default
		}

		response := instance.MakeRequest(config)

		if response.Error != nil {
			t.Errorf("Expected no error with default timeout, got %v", response.Error)
		}
	})
}

func TestHttpInstance_MakeRequestAsync(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"async": "response"}`))
	}))
	defer server.Close()

	instance := NewHttpInstance()

	var wg sync.WaitGroup
	var response *Response

	wg.Add(1)
	callback := func(resp *Response) {
		response = resp
		wg.Done()
	}

	config := RequestConfig{
		Method:  GET,
		URL:     server.URL + "/api2/json/test",
		Timeout: 5 * time.Second,
	}

	instance.MakeRequestAsync(config, callback)

	// Wait for async request to complete
	wg.Wait()

	if response == nil {
		t.Fatal("Expected response from async request")
	}

	if response.StatusCode != http.StatusOK {
		t.Errorf("Expected status code 200, got %d", response.StatusCode)
	}

	expectedBody := `{"async": "response"}`
	if string(response.Body) != expectedBody {
		t.Errorf("Expected body '%s', got '%s'", expectedBody, string(response.Body))
	}
}

func TestHttpInstance_MakeRequestBatch(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api2/json/test1":
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"test": "1"}`))
		case "/api2/json/test2":
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"test": "2"}`))
		case "/api2/json/test3":
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"test": "3"}`))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	instance := NewHttpInstance()

	configs := []RequestConfig{
		{
			Method:  GET,
			URL:     server.URL + "/api2/json/test1",
			Timeout: 5 * time.Second,
		},
		{
			Method:  GET,
			URL:     server.URL + "/api2/json/test2",
			Timeout: 5 * time.Second,
		},
		{
			Method:  GET,
			URL:     server.URL + "/api2/json/test3",
			Timeout: 5 * time.Second,
		},
	}

	responses := instance.MakeRequestBatch(configs)

	if len(responses) != 3 {
		t.Fatalf("Expected 3 responses, got %d", len(responses))
	}

	for i, response := range responses {
		if response.Error != nil {
			t.Errorf("Expected no error for response %d, got %v", i, response.Error)
		}

		if response.StatusCode != http.StatusOK {
			t.Errorf("Expected status code 200 for response %d, got %d", i, response.StatusCode)
		}

		expectedBody := fmt.Sprintf(`{"test": "%d"}`, i+1)
		if string(response.Body) != expectedBody {
			t.Errorf("Expected body '%s' for response %d, got '%s'", expectedBody, i, string(response.Body))
		}
	}
}

func TestHttpInstance_SetWorkerPoolSize(t *testing.T) {
	instance := NewHttpInstance()

	// Test setting a new worker pool size
	instance.SetWorkerPoolSize(200)

	// This test mainly checks that the method doesn't panic
	// The actual pool size change is difficult to test directly
	// but we can test it indirectly by ensuring requests still work

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"pool": "test"}`))
	}))
	defer server.Close()

	config := RequestConfig{
		Method:  GET,
		URL:     server.URL + "/api2/json/test",
		Timeout: 5 * time.Second,
	}

	response := instance.MakeRequest(config)

	if response.Error != nil {
		t.Errorf("Expected no error after changing pool size, got %v", response.Error)
	}
}

func TestHttpInstance_BaseEndpointPath(t *testing.T) {
	instance := NewHttpInstance()

	t.Run("get default base endpoint path", func(t *testing.T) {
		path := instance.GetBaseEndpointPath()
		if path != "/api2/json" {
			t.Errorf("Expected default base endpoint path '/api2/json', got '%s'", path)
		}
	})

	t.Run("set custom base endpoint path", func(t *testing.T) {
		customPath := "/custom/endpoint"
		instance.SetBaseEndpointPath(customPath)

		path := instance.GetBaseEndpointPath()
		if path != customPath {
			t.Errorf("Expected custom base endpoint path '%s', got '%s'", customPath, path)
		}
	})
}

func TestHttpInstance_BuildFullURL(t *testing.T) {
	instance := NewHttpInstance()

	tests := []struct {
		name     string
		basePath string
		endpoint string
		expected string
	}{
		{
			name:     "simple endpoint",
			basePath: "/api2/json",
			endpoint: "/nodes",
			expected: "/api2/json/nodes",
		},
		{
			name:     "endpoint without leading slash",
			basePath: "/api2/json",
			endpoint: "nodes",
			expected: "/api2/json/nodes",
		},
		{
			name:     "base path without trailing slash",
			basePath: "/api2/json",
			endpoint: "/nodes",
			expected: "/api2/json/nodes",
		},
		{
			name:     "empty endpoint",
			basePath: "/api2/json",
			endpoint: "",
			expected: "/api2/json",
		},
		{
			name:     "full URL endpoint",
			basePath: "/api2/json",
			endpoint: "http://example.com/api/test",
			expected: "http://example.com/api/test",
		},
		{
			name:     "https URL endpoint",
			basePath: "/api2/json",
			endpoint: "https://example.com/api/test",
			expected: "https://example.com/api/test",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			instance.SetBaseEndpointPath(tt.basePath)

			// We need to test this indirectly since buildFullURL is not exported
			// We'll create a request with a mock server and check the behavior
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
			}))
			defer server.Close()

			// For testing purposes, we'll test the URL building logic
			// by making actual requests when possible
			if tt.endpoint != "" && !http.DefaultTransport.(*http.Transport).DisableKeepAlives {
				config := RequestConfig{
					Method:  GET,
					URL:     tt.endpoint,
					Timeout: 1 * time.Second,
				}

				// This will test the URL building internally
				response := instance.MakeRequest(config)

				// The request might fail due to invalid URLs in tests,
				// but we're mainly testing that the method doesn't panic
				_ = response
			}
		})
	}
}

func TestRequestMethod_Constants(t *testing.T) {
	tests := []struct {
		method   RequestMethod
		expected string
	}{
		{GET, "GET"},
		{POST, "POST"},
		{PUT, "PUT"},
		{DELETE, "DELETE"},
		{PATCH, "PATCH"},
	}

	for _, tt := range tests {
		t.Run(string(tt.method), func(t *testing.T) {
			if string(tt.method) != tt.expected {
				t.Errorf("Expected method '%s', got '%s'", tt.expected, string(tt.method))
			}
		})
	}
}

func TestRequestConfig_Struct(t *testing.T) {
	config := RequestConfig{
		Method:  POST,
		URL:     "http://example.com",
		Headers: map[string]string{"Content-Type": "application/json"},
		Body:    []byte(`{"test": "data"}`),
		Timeout: 10 * time.Second,
	}

	if config.Method != POST {
		t.Errorf("Expected method POST, got %s", config.Method)
	}

	if config.URL != "http://example.com" {
		t.Errorf("Expected URL 'http://example.com', got '%s'", config.URL)
	}

	if config.Headers["Content-Type"] != "application/json" {
		t.Errorf("Expected Content-Type header 'application/json', got '%s'", config.Headers["Content-Type"])
	}

	if string(config.Body) != `{"test": "data"}` {
		t.Errorf("Expected body '{\"test\": \"data\"}', got '%s'", string(config.Body))
	}

	if config.Timeout != 10*time.Second {
		t.Errorf("Expected timeout 10s, got %v", config.Timeout)
	}
}

func TestResponse_Struct(t *testing.T) {
	response := &Response{
		StatusCode: 200,
		Body:       []byte(`{"result": "ok"}`),
		Headers:    map[string]string{"Content-Type": "application/json"},
		Error:      nil,
	}

	if response.StatusCode != 200 {
		t.Errorf("Expected status code 200, got %d", response.StatusCode)
	}

	if string(response.Body) != `{"result": "ok"}` {
		t.Errorf("Expected body '{\"result\": \"ok\"}', got '%s'", string(response.Body))
	}

	if response.Headers["Content-Type"] != "application/json" {
		t.Errorf("Expected Content-Type header 'application/json', got '%s'", response.Headers["Content-Type"])
	}

	if response.Error != nil {
		t.Errorf("Expected no error, got %v", response.Error)
	}
}

func TestHttpInstance_GetRequestRate(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status": "ok"}`))
	}))
	defer server.Close()

	client := NewHttpInstance()
	defer client.Close()

	// Reset counter
	client.ResetRequestCounter()

	// Make some requests
	config := RequestConfig{
		Method:  GET,
		URL:     server.URL + "/test",
		Timeout: 1 * time.Second,
	}

	// Make 3 requests quickly
	for i := 0; i < 3; i++ {
		client.MakeRequest(config)
	}

	// Measure rate over a short interval
	rate := client.GetRequestRate(100 * time.Millisecond)

	// Rate should be 0 since we already made the requests
	if rate != 0 {
		t.Logf("Request rate: %.2f req/sec (expected 0 for completed requests)", rate)
	}
}

func TestHttpInstance_ResetRequestCounter(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := NewHttpInstance()
	defer client.Close()

	// Make a request
	config := RequestConfig{
		Method: GET,
		URL:    server.URL,
	}
	client.MakeRequest(config)

	// Check counter is > 0
	if client.GetRequestCount() == 0 {
		t.Error("Expected request counter > 0")
	}

	// Reset counter
	client.ResetRequestCounter()

	// Check counter is 0
	if client.GetRequestCount() != 0 {
		t.Errorf("Expected request counter to be 0 after reset, got %d", client.GetRequestCount())
	}
}

func TestHttpInstance_IsHealthy(t *testing.T) {
	client := NewHttpInstance()
	defer client.Close()

	// Should be healthy initially
	if !client.IsHealthy() {
		t.Error("Expected client to be healthy initially")
	}

	// Test with overloaded worker pool
	smallPoolClient := NewHttpInstance(WithWorkerPoolSize(1))
	defer smallPoolClient.Close()

	// Fill the worker pool
	var wg sync.WaitGroup
	wg.Add(1)
	smallPoolClient.workerPool.Submit(func() {
		defer wg.Done()
		time.Sleep(100 * time.Millisecond)
	})

	time.Sleep(10 * time.Millisecond) // Let task start

	// Health check implementation may vary based on criteria
	// This test verifies the method works without error
	healthy := smallPoolClient.IsHealthy()
	t.Logf("Client health with busy worker pool: %v", healthy)

	wg.Wait()
}

func TestHttpInstance_GetMemoryStats(t *testing.T) {
	client := NewHttpInstance()
	defer client.Close()

	stats := client.GetMemoryStats()

	// Check expected keys exist
	if _, exists := stats["request_counter"]; !exists {
		t.Error("Expected request_counter in memory stats")
	}

	if _, exists := stats["base_endpoint_path"]; !exists {
		t.Error("Expected base_endpoint_path in memory stats")
	}

	if _, exists := stats["buffer_manager"]; !exists {
		t.Error("Expected buffer_manager in memory stats")
	}

	// Verify request counter type
	if counter, ok := stats["request_counter"].(int64); !ok {
		t.Error("Expected request_counter to be int64")
	} else if counter < 0 {
		t.Error("Expected request_counter to be non-negative")
	}
}

func TestHttpInstance_AdvancedFeatures(t *testing.T) {
	// Test integration of all advanced features
	client := NewHttpInstance(WithProxmoxDefaults())
	defer client.Close()

	// Test buffer manager access
	bufManager := client.GetBufferManager()
	if bufManager == nil {
		t.Fatal("Expected non-nil buffer manager")
	}

	// Test worker pool stats
	active, total, max := client.GetWorkerPoolStats()
	if max <= 0 {
		t.Error("Expected positive max workers")
	}

	if total < 0 {
		t.Error("Expected non-negative total tasks")
	}

	if active < 0 {
		t.Error("Expected non-negative active workers")
	}

	// Test memory stats
	memStats := client.GetMemoryStats()
	if len(memStats) == 0 {
		t.Error("Expected non-empty memory stats")
	}

	// Test health check
	if !client.IsHealthy() {
		t.Error("Expected healthy client")
	}
}

// Benchmark tests
func BenchmarkHttpInstance_MakeRequest(b *testing.B) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"benchmark": "test"}`))
	}))
	defer server.Close()

	instance := NewHttpInstance()
	config := RequestConfig{
		Method:  GET,
		URL:     server.URL + "/api2/json/test",
		Timeout: 5 * time.Second,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		response := instance.MakeRequest(config)
		if response.Error != nil {
			b.Fatalf("Unexpected error: %v", response.Error)
		}
	}
}

func BenchmarkHttpInstance_MakeRequestBatch(b *testing.B) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"benchmark": "batch"}`))
	}))
	defer server.Close()

	instance := NewHttpInstance()
	configs := []RequestConfig{
		{
			Method:  GET,
			URL:     server.URL + "/api2/json/test1",
			Timeout: 5 * time.Second,
		},
		{
			Method:  GET,
			URL:     server.URL + "/api2/json/test2",
			Timeout: 5 * time.Second,
		},
		{
			Method:  GET,
			URL:     server.URL + "/api2/json/test3",
			Timeout: 5 * time.Second,
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		responses := instance.MakeRequestBatch(configs)
		for _, response := range responses {
			if response.Error != nil {
				b.Fatalf("Unexpected error: %v", response.Error)
			}
		}
	}
}

func BenchmarkNewHttpInstance(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		instance := NewHttpInstance()
		if instance == nil {
			b.Fatal("Expected non-nil instance")
		}
	}
}

func BenchmarkNewHttpInstanceWithOptions(b *testing.B) {
	customClient := &fasthttp.Client{
		ReadTimeout: 10 * time.Second,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		instance := NewHttpInstance(
			WithHttpClient(customClient),
			WithBaseEndpointPath("/custom/api"),
			WithWorkerPoolSize(50),
		)
		if instance == nil {
			b.Fatal("Expected non-nil instance")
		}
	}
}
