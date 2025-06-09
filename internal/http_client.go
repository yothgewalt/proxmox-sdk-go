package internal

import (
	"path"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/valyala/fasthttp"
)

// RequestMethod represents HTTP methods
type RequestMethod string

const (
	GET    RequestMethod = "GET"
	POST   RequestMethod = "POST"
	PUT    RequestMethod = "PUT"
	DELETE RequestMethod = "DELETE"
	PATCH  RequestMethod = "PATCH"
)

// RequestConfig holds configuration for HTTP requests
type RequestConfig struct {
	Method  RequestMethod
	URL     string
	Headers map[string]string
	Body    []byte
	Timeout time.Duration
}

// Response represents HTTP response with error handling
type Response struct {
	StatusCode int
	Body       []byte
	Headers    map[string]string
	Error      error
}

type HttpInstnace struct {
	HttpClient *fasthttp.Client
	// Worker pool for concurrent request handling
	workerPool chan struct{}

	// Object pools for memory optimization
	requestPool  sync.Pool
	responsePool sync.Pool

	// Mutex for thread-safe operations
	mutex sync.RWMutex

	// Request counter for monitoring and rate limiting
	requestCounter int64

	// Base endpoint path for API requests
	// Reason: Provides consistent API path management and allows customization
	baseEndpointPath string
}

// NewHttpInstance creates a new instance of HttpInstnace with default settings
func NewHttpInstance(options ...Option[HttpInstnace]) *HttpInstnace {
	connector := HttpInstnace{
		HttpClient: createDefaultHttpClient(),
		workerPool: make(chan struct{}, 100),
		// Default base endpoint path for Proxmox API
		baseEndpointPath: "/api2/json",

		requestPool: sync.Pool{
			New: func() interface{} {
				return &fasthttp.Request{}
			},
		},
		responsePool: sync.Pool{
			New: func() interface{} {
				return &fasthttp.Response{}
			},
		},
	}

	ApplyOptions(&connector, options...)

	return &connector
}

// MakeRequest performs a HTTP request with proper resource management
func (h *HttpInstnace) MakeRequest(config RequestConfig) *Response {
	atomic.AddInt64(&h.requestCounter, 1)

	h.workerPool <- struct{}{}
	defer func() { <-h.workerPool }()

	req := h.requestPool.Get().(*fasthttp.Request)
	resp := h.responsePool.Get().(*fasthttp.Response)

	defer func() {
		req.Reset()
		h.requestPool.Put(req)
		resp.Reset()
		h.responsePool.Put(resp)
	}()

	fullURL := h.buildFullURL(config.URL)
	req.SetRequestURI(fullURL)
	req.Header.SetMethod(string(config.Method))

	for key, value := range config.Headers {
		req.Header.Set(key, value)
	}

	if len(config.Body) > 0 {
		req.SetBody(config.Body)
	}

	var timeout time.Duration
	if config.Timeout > 0 {
		timeout = config.Timeout
	} else {
		timeout = 30 * time.Second
	}

	err := h.HttpClient.DoTimeout(req, resp, timeout)

	response := &Response{
		StatusCode: resp.StatusCode(),
		Error:      err,
	}

	if resp.Body() != nil {
		response.Body = make([]byte, len(resp.Body()))
		copy(response.Body, resp.Body())
	}

	response.Headers = make(map[string]string)
	resp.Header.VisitAll(func(key, value []byte) {
		response.Headers[string(key)] = string(value)
	})

	return response
}

// GetRequestCount returns the current request counter
func (h *HttpInstnace) GetRequestCount() int64 {
	return atomic.LoadInt64(&h.requestCounter)
}

// SetWorkerPoolSize allows dynamic adjustment of worker pool size
func (h *HttpInstnace) SetWorkerPoolSize(size int) {
	h.mutex.Lock()
	defer h.mutex.Unlock()

	h.workerPool = make(chan struct{}, size)
}

// MakeRequestAsync performs an asynchronous HTTP request
func (h *HttpInstnace) MakeRequestAsync(config RequestConfig, callback func(*Response)) {
	go func() {
		response := h.MakeRequest(config)
		if callback != nil {
			callback(response)
		}
	}()
}

// MakeRequestBatch performs multiple requests concurrently with result aggregation
func (h *HttpInstnace) MakeRequestBatch(configs []RequestConfig) []*Response {
	responses := make([]*Response, len(configs))
	var wg sync.WaitGroup

	for i, config := range configs {
		wg.Add(1)
		go func(index int, cfg RequestConfig) {
			defer wg.Done()
			responses[index] = h.MakeRequest(cfg)
		}(i, config)
	}

	wg.Wait()
	return responses
}

// buildFullURL constructs the complete URL by combining base endpoint path with the given endpoint
func (h *HttpInstnace) buildFullURL(endpoint string) string {
	// Handle cases where endpoint might already be a full URL
	if strings.HasPrefix(endpoint, "http://") || strings.HasPrefix(endpoint, "https://") {
		return endpoint
	}

	// Clean and join paths to avoid double slashes
	cleanBasePath := strings.TrimSuffix(h.baseEndpointPath, "/")
	cleanEndpoint := strings.TrimPrefix(endpoint, "/")

	if cleanEndpoint == "" {
		return cleanBasePath
	}

	return path.Join(cleanBasePath, cleanEndpoint)
}

// GetBaseEndpointPath returns the current base endpoint path
func (h *HttpInstnace) GetBaseEndpointPath() string {
	h.mutex.RLock()
	defer h.mutex.RUnlock()
	return h.baseEndpointPath
}

// SetBaseEndpointPath updates the base endpoint path dynamically
func (h *HttpInstnace) SetBaseEndpointPath(basePath string) {
	h.mutex.Lock()
	defer h.mutex.Unlock()
	h.baseEndpointPath = basePath
}

func createDefaultHttpClient() *fasthttp.Client {
	return &fasthttp.Client{
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 10 * time.Second,

		MaxIdleConnDuration: 90 * time.Second,
		MaxConnDuration:     300 * time.Second,
		MaxConnsPerHost:     10,

		ReadBufferSize:      4096,
		WriteBufferSize:     4096,
		MaxResponseBodySize: 10 << 20,

		TLSConfig: nil,

		DisableHeaderNamesNormalizing: false,
		DisablePathNormalizing:        false,
	}
}
