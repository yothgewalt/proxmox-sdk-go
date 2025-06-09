package examples

import (
	"fmt"
	"log"
	"time"

	"github.com/valyala/fasthttp"
	"yoth.dev/proxmox-sdk-go/internal"
)

// ExampleBasicWorkerPool demonstrates basic worker pool usage
func ExampleBasicWorkerPool() {
	// Create HTTP client with default worker pool
	client := internal.NewHttpInstance()
	defer client.Close() // Always close to cleanup resources

	// Make a simple request
	response := client.MakeRequest(internal.RequestConfig{
		Method:  internal.GET,
		URL:     "https://api.proxmox.com/api2/json/nodes",
		Timeout: 10 * time.Second,
	})

	if response.Error != nil {
		log.Printf("Request failed: %v", response.Error)
		return
	}

	fmt.Printf("Response: %d - %s\n", response.StatusCode, string(response.Body))
}

// ExampleCustomWorkerPool demonstrates custom worker pool configuration
func ExampleCustomWorkerPool() {
	// Create HTTP client with custom worker pool settings
	client := internal.NewHttpInstance(
		internal.WithWorkerPoolConfig(internal.WorkerPoolConfig{
			MaxWorkers:  200, // High concurrency for bulk operations
			TaskTimeout: 60 * time.Second,
		}),
		internal.WithBaseEndpointPath("/api2/json"),
	)
	defer client.Close()

	// Get worker pool statistics
	active, total, max := client.GetWorkerPoolStats()
	fmt.Printf("Worker Pool Stats - Active: %d, Total: %d, Max: %d\n", active, total, max)

	// Make multiple concurrent requests
	configs := []internal.RequestConfig{
		{Method: internal.GET, URL: "/nodes", Timeout: 5 * time.Second},
		{Method: internal.GET, URL: "/cluster/status", Timeout: 5 * time.Second},
		{Method: internal.GET, URL: "/access/users", Timeout: 5 * time.Second},
	}

	responses := client.MakeRequestBatch(configs)
	for i, resp := range responses {
		if resp.Error != nil {
			log.Printf("Request %d failed: %v", i, resp.Error)
		} else {
			fmt.Printf("Request %d: Status %d\n", i, resp.StatusCode)
		}
	}
}

// ExampleProxmoxOptimized demonstrates Proxmox-optimized configuration
func ExampleProxmoxOptimized() {
	// Create HTTP client optimized for Proxmox API
	client := internal.NewHttpInstance(
		internal.WithProxmoxDefaults(), // Apply Proxmox-optimized settings
	)
	defer client.Close()

	// Use Proxmox-specific request methods
	response := client.ProxmoxJSONRequest(
		internal.GET,
		"/nodes/localhost/status",
		nil, // No payload for GET request
	)

	if response.Error != nil {
		log.Printf("Proxmox request failed: %v", response.Error)
		return
	}

	fmt.Printf("Proxmox Response: %s\n", string(response.Body))
}

// ExampleBufferUsage demonstrates efficient buffer usage
func ExampleBufferUsage() {
	client := internal.NewHttpInstance()
	defer client.Close()

	// Get the buffer manager for advanced usage
	bufManager := client.GetBufferManager()

	// Example: Building a complex JSON payload efficiently
	jsonBuf := bufManager.GetJSONBuffer()
	defer bufManager.PutJSONBuffer(jsonBuf)

	// Build JSON manually (in practice, you'd use json.Marshal)
	jsonBuf.WriteString(`{"node": "localhost", "action": "start"}`)

	response := client.MakeRequest(internal.RequestConfig{
		Method: internal.POST,
		URL:    "/nodes/localhost/qemu/100/status",
		Headers: map[string]string{
			"Content-Type": "application/json",
		},
		Body:    jsonBuf.Bytes(),
		Timeout: 15 * time.Second,
	})

	if response.Error != nil {
		log.Printf("Request failed: %v", response.Error)
	}
}

// ExampleFormDataRequest demonstrates form-encoded requests
func ExampleFormDataRequest() {
	client := internal.NewHttpInstance(internal.WithProxmoxDefaults())
	defer client.Close()

	// Use the convenience method for form requests
	formData := map[string]string{
		"username": "admin",
		"password": "secret",
		"realm":    "pam",
	}

	response := client.ProxmoxFormRequest(
		internal.POST,
		"/access/ticket",
		formData,
	)

	if response.Error != nil {
		log.Printf("Authentication failed: %v", response.Error)
		return
	}

	fmt.Printf("Auth Response: %s\n", string(response.Body))
}

// ExampleBatchProxmoxRequests demonstrates efficient batch processing
func ExampleBatchProxmoxRequests() {
	client := internal.NewHttpInstance(internal.WithProxmoxDefaults())
	defer client.Close()

	// Prepare multiple Proxmox API requests
	requests := []internal.ProxmoxRequestConfig{
		{
			Method:      internal.GET,
			Endpoint:    "/nodes",
			ContentType: "application/json",
			Timeout:     10 * time.Second,
		},
		{
			Method:      internal.GET,
			Endpoint:    "/cluster/resources",
			ContentType: "application/json",
			Timeout:     10 * time.Second,
		},
		{
			Method:      internal.GET,
			Endpoint:    "/access/users",
			ContentType: "application/json",
			Timeout:     10 * time.Second,
		},
	}

	// Execute all requests concurrently with optimized resource usage
	responses := client.ProxmoxBatchRequest(requests)

	for i, resp := range responses {
		if resp.Error != nil {
			log.Printf("Batch request %d failed: %v", i, resp.Error)
		} else {
			fmt.Printf("Batch request %d: Status %d, Body length: %d\n",
				i, resp.StatusCode, len(resp.Body))
		}
	}
}

// ExampleCustomHTTPClient demonstrates using a custom HTTP client
func ExampleCustomHTTPClient() {
	// Create a custom fasthttp client
	customClient := &fasthttp.Client{
		ReadTimeout:         45 * time.Second,
		WriteTimeout:        20 * time.Second,
		MaxIdleConnDuration: 300 * time.Second,
		MaxConnsPerHost:     50, // Higher connection limit
	}

	// Use the custom client with optimized worker pool
	client := internal.NewHttpInstance(
		internal.WithHttpClient(customClient),
		internal.WithWorkerPoolSize(150), // Custom worker pool size
		internal.WithBaseEndpointPath("/api2/json"),
	)
	defer client.Close()

	// The client now uses your custom HTTP settings
	response := client.MakeRequest(internal.RequestConfig{
		Method:  internal.GET,
		URL:     "/version",
		Timeout: 30 * time.Second,
	})

	if response.Error == nil {
		fmt.Printf("Custom client response: %s\n", string(response.Body))
	}
}

// ExampleAsyncRequests demonstrates asynchronous request handling
func ExampleAsyncRequests() {
	client := internal.NewHttpInstance(internal.WithProxmoxDefaults())
	defer client.Close()

	// Create a channel to collect async responses
	responseChan := make(chan *internal.Response, 3)

	// Make multiple async requests
	endpoints := []string{"/nodes", "/cluster/status", "/access/domains"}

	for _, endpoint := range endpoints {
		client.MakeRequestAsync(
			internal.RequestConfig{
				Method:  internal.GET,
				URL:     endpoint,
				Timeout: 10 * time.Second,
			},
			func(resp *internal.Response) {
				responseChan <- resp
			},
		)
	}

	// Collect responses
	for i := 0; i < len(endpoints); i++ {
		select {
		case resp := <-responseChan:
			if resp.Error != nil {
				log.Printf("Async request failed: %v", resp.Error)
			} else {
				fmt.Printf("Async response: Status %d\n", resp.StatusCode)
			}
		case <-time.After(15 * time.Second):
			log.Printf("Timeout waiting for async response")
		}
	}
}

// ExampleGracefulShutdown demonstrates proper resource cleanup
func ExampleGracefulShutdown() {
	client := internal.NewHttpInstance(internal.WithProxmoxDefaults())

	// Make some requests
	response := client.MakeRequest(internal.RequestConfig{
		Method:  internal.GET,
		URL:     "/version",
		Timeout: 5 * time.Second,
	})

	if response.Error == nil {
		fmt.Printf("Response received: %d\n", response.StatusCode)
	}

	// Gracefully shutdown with timeout
	err := client.CloseWithTimeout(5 * time.Second)
	if err != nil {
		log.Printf("Shutdown timed out: %v", err)
	} else {
		fmt.Println("Client shutdown gracefully")
	}
}

// ExampleDynamicWorkerPoolResize demonstrates runtime worker pool adjustment
func ExampleDynamicWorkerPoolResize() {
	client := internal.NewHttpInstance()
	defer client.Close()

	// Start with default worker pool
	active, total, max := client.GetWorkerPoolStats()
	fmt.Printf("Initial - Active: %d, Total: %d, Max: %d\n", active, total, max)

	// Increase worker pool for high-load period
	client.SetWorkerPoolSize(300)

	// Make many concurrent requests
	configs := make([]internal.RequestConfig, 50)
	for i := range configs {
		configs[i] = internal.RequestConfig{
			Method:  internal.GET,
			URL:     fmt.Sprintf("/test-endpoint-%d", i),
			Timeout: 5 * time.Second,
		}
	}

	responses := client.MakeRequestBatch(configs)
	fmt.Printf("Processed %d requests\n", len(responses))

	// Get updated stats
	active, total, max = client.GetWorkerPoolStats()
	fmt.Printf("After batch - Active: %d, Total: %d, Max: %d\n", active, total, max)

	// Reduce worker pool size after high-load period
	client.SetWorkerPoolSize(50)
	fmt.Println("Worker pool size reduced for normal operations")
}
