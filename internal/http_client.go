package internal

import (
	"path"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/bytedance/sonic"
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

// TypedRequest represents a generic request with typed data
type TypedRequest[T any] struct {
	Method  RequestMethod
	URL     string
	Headers map[string]string
	Data    T
	Timeout time.Duration
}

// TypedResponse represents a generic response with typed data
type TypedResponse[T any] struct {
	StatusCode int
	Data       T
	Headers    map[string]string
	RawBody    []byte
	Error      error
}

// RequestEncoder defines how to encode request data
type RequestEncoder[T any] interface {
	Encode(data T) ([]byte, error)
}

// ResponseDecoder defines how to decode response data
type ResponseDecoder[T any] interface {
	Decode(body []byte) (T, error)
}

// JSONRequestEncoder implements JSON encoding for requests using sonic
type JSONRequestEncoder[T any] struct{}

func (e JSONRequestEncoder[T]) Encode(data T) ([]byte, error) {
	return sonic.Marshal(data)
}

// JSONResponseDecoder implements JSON decoding for responses using sonic
type JSONResponseDecoder[T any] struct{}

func (d JSONResponseDecoder[T]) Decode(body []byte) (T, error) {
	var result T
	err := sonic.Unmarshal(body, &result)
	return result, err
}

type HttpInstnace struct {
	HttpClient *fasthttp.Client

	// Worker pool for concurrent request handling
	workerPool *WorkerPool

	// Buffer manager for memory optimization
	bufferManager *RequestBufferManager

	// Object pools for memory optimization
	requestPool  sync.Pool
	responsePool sync.Pool

	// Mutex for thread-safe operations
	mutex sync.RWMutex

	// Request counter for monitoring and rate limiting
	requestCounter int64

	// Base endpoint path for API requests
	baseEndpointPath string
}

// NewHttpInstance creates a new instance of HttpInstnace with default settings
func NewHttpInstance(options ...Option[HttpInstnace]) *HttpInstnace {
	workerPoolConfig := WorkerPoolConfig{
		MaxWorkers: 100, // Default concurrent requests
	}

	connector := HttpInstnace{
		HttpClient:    createDefaultHttpClient(),
		workerPool:    NewWorkerPool(workerPoolConfig),
		bufferManager: NewRequestBufferManager(),

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

	responseChan := make(chan *Response, 1)

	err := h.workerPool.Submit(func() {
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

		httpErr := h.HttpClient.DoTimeout(req, resp, timeout)

		response := &Response{
			StatusCode: resp.StatusCode(),
			Error:      httpErr,
		}

		if resp.Body() != nil {
			response.Body = make([]byte, len(resp.Body()))
			copy(response.Body, resp.Body())
		}

		response.Headers = make(map[string]string)
		resp.Header.VisitAll(func(key, value []byte) {
			response.Headers[string(key)] = string(value)
		})

		responseChan <- response
	})

	if err != nil {
		return &Response{
			Error: err,
		}
	}

	return <-responseChan
}

// GetRequestCount returns the current request counter
func (h *HttpInstnace) GetRequestCount() int64 {
	return atomic.LoadInt64(&h.requestCounter)
}

// SetWorkerPoolSize allows dynamic adjustment of worker pool size
func (h *HttpInstnace) SetWorkerPoolSize(size int) {
	h.mutex.Lock()
	defer h.mutex.Unlock()

	h.workerPool.Resize(size)
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
	if strings.HasPrefix(endpoint, "http://") || strings.HasPrefix(endpoint, "https://") {
		return endpoint
	}

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

// GetBufferManager returns the buffer manager for advanced usage
func (h *HttpInstnace) GetBufferManager() *RequestBufferManager {
	return h.bufferManager
}

// GetWorkerPoolStats returns current worker pool statistics
func (h *HttpInstnace) GetWorkerPoolStats() (activeWorkers, totalTasks int64, maxWorkers int) {
	return h.workerPool.GetStats()
}

// Close gracefully shuts down the HTTP client and its resources
func (h *HttpInstnace) Close() error {
	return h.workerPool.Close()
}

// CloseWithTimeout shuts down the HTTP client with a timeout
func (h *HttpInstnace) CloseWithTimeout(timeout time.Duration) error {
	return h.workerPool.CloseWithTimeout(timeout)
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

// ProxmoxRequestConfig holds configuration for Proxmox-specific requests
type ProxmoxRequestConfig struct {
	Method      RequestMethod
	Endpoint    string
	Headers     map[string]string
	Body        []byte
	ContentType string
	Timeout     time.Duration
}

// ProxmoxJSONRequest performs a Proxmox API request with JSON payload
func (h *HttpInstnace) ProxmoxJSONRequest(method RequestMethod, endpoint string, payload []byte) *Response {
	config := RequestConfig{
		Method:  method,
		URL:     endpoint,
		Body:    payload,
		Timeout: 30 * time.Second,
		Headers: map[string]string{
			"Content-Type": "application/json",
			"Accept":       "application/json",
		},
	}

	return h.MakeRequest(config)
}

// ProxmoxFormRequest performs a Proxmox API request with form-encoded data
func (h *HttpInstnace) ProxmoxFormRequest(method RequestMethod, endpoint string, formData map[string]string) *Response {
	// Use buffer manager to build form data efficiently
	buf := h.bufferManager.GetQueryBuffer()
	defer h.bufferManager.PutQueryBuffer(buf)

	first := true
	for key, value := range formData {
		if !first {
			buf.WriteByte('&')
		}
		buf.WriteString(key)
		buf.WriteByte('=')
		buf.WriteString(value)
		first = false
	}

	config := RequestConfig{
		Method:  method,
		URL:     endpoint,
		Body:    buf.Bytes(),
		Timeout: 30 * time.Second,
		Headers: map[string]string{
			"Content-Type": "application/x-www-form-urlencoded",
			"Accept":       "application/json",
		},
	}

	return h.MakeRequest(config)
}

// ProxmoxBatchRequest performs multiple Proxmox API requests concurrently
func (h *HttpInstnace) ProxmoxBatchRequest(configs []ProxmoxRequestConfig) []*Response {
	requestConfigs := make([]RequestConfig, len(configs))

	for i, proxConfig := range configs {
		headers := make(map[string]string)

		headers["Accept"] = "application/json"

		if proxConfig.ContentType != "" {
			headers["Content-Type"] = proxConfig.ContentType
		}

		for key, value := range proxConfig.Headers {
			headers[key] = value
		}

		timeout := proxConfig.Timeout
		if timeout == 0 {
			timeout = 30 * time.Second
		}

		requestConfigs[i] = RequestConfig{
			Method:  proxConfig.Method,
			URL:     proxConfig.Endpoint,
			Headers: headers,
			Body:    proxConfig.Body,
			Timeout: timeout,
		}
	}

	return h.MakeRequestBatch(requestConfigs)
}

// GetRequestRate returns requests per second over the last interval
func (h *HttpInstnace) GetRequestRate(interval time.Duration) float64 {
	start := atomic.LoadInt64(&h.requestCounter)
	time.Sleep(interval)
	end := atomic.LoadInt64(&h.requestCounter)

	requests := float64(end - start)
	seconds := interval.Seconds()

	if seconds > 0 {
		return requests / seconds
	}
	return 0
}

// ResetRequestCounter resets the request counter (useful for testing/monitoring)
func (h *HttpInstnace) ResetRequestCounter() {
	atomic.StoreInt64(&h.requestCounter, 0)
}

// IsHealthy performs a basic health check of the HTTP client
func (h *HttpInstnace) IsHealthy() bool {
	active, _, max := h.workerPool.GetStats()

	if max > 0 && active >= int64(max) {
		return false
	}

	return true
}

// GetMemoryStats returns memory usage statistics for buffer pools
func (h *HttpInstnace) GetMemoryStats() map[string]interface{} {
	// Note: sync.Pool doesn't expose size directly, so this is basic
	return map[string]interface{}{
		"request_counter":    atomic.LoadInt64(&h.requestCounter),
		"base_endpoint_path": h.GetBaseEndpointPath(),
		"buffer_manager":     "RequestBufferManager initialized",
	}
}

// MakeTypedRequest performs a typed HTTP request with automatic encoding/decoding
func MakeTypedRequest[Req, Resp any](
	httpInstance *HttpInstnace,
	config TypedRequest[Req],
	encoder RequestEncoder[Req],
	decoder ResponseDecoder[Resp],
) *TypedResponse[Resp] {
	atomic.AddInt64(&httpInstance.requestCounter, 1)

	responseChan := make(chan *TypedResponse[Resp], 1)

	err := httpInstance.workerPool.Submit(func() {
		req := httpInstance.requestPool.Get().(*fasthttp.Request)
		resp := httpInstance.responsePool.Get().(*fasthttp.Response)

		defer func() {
			req.Reset()
			httpInstance.requestPool.Put(req)
			resp.Reset()
			httpInstance.responsePool.Put(resp)
		}()

		var body []byte
		var encodeErr error
		if encoder != nil {
			body, encodeErr = encoder.Encode(config.Data)
			if encodeErr != nil {
				responseChan <- &TypedResponse[Resp]{
					Error: encodeErr,
				}
				return
			}
		}

		fullURL := httpInstance.buildFullURL(config.URL)
		req.SetRequestURI(fullURL)
		req.Header.SetMethod(string(config.Method))

		for key, value := range config.Headers {
			req.Header.Set(key, value)
		}

		if len(body) > 0 {
			req.SetBody(body)
		}

		var timeout time.Duration
		if config.Timeout > 0 {
			timeout = config.Timeout
		} else {
			timeout = 30 * time.Second
		}

		httpErr := httpInstance.HttpClient.DoTimeout(req, resp, timeout)

		response := &TypedResponse[Resp]{
			StatusCode: resp.StatusCode(),
			Error:      httpErr,
		}

		if resp.Body() != nil {
			response.RawBody = make([]byte, len(resp.Body()))
			copy(response.RawBody, resp.Body())

			if decoder != nil && httpErr == nil {
				data, decodeErr := decoder.Decode(response.RawBody)
				if decodeErr != nil {
					response.Error = decodeErr
				} else {
					response.Data = data
				}
			}
		}

		response.Headers = make(map[string]string)
		resp.Header.VisitAll(func(key, value []byte) {
			response.Headers[string(key)] = string(value)
		})

		responseChan <- response
	})

	if err != nil {
		return &TypedResponse[Resp]{
			Error: err,
		}
	}

	return <-responseChan
}

// MakeJSONRequest performs a typed JSON request with automatic JSON marshaling/unmarshaling using sonic
func MakeJSONRequest[Req, Resp any](
	httpInstance *HttpInstnace,
	method RequestMethod,
	endpoint string,
	requestData Req,
	headers map[string]string,
) *TypedResponse[Resp] {
	if headers == nil {
		headers = make(map[string]string)
	}
	headers["Content-Type"] = "application/json"
	headers["Accept"] = "application/json"

	config := TypedRequest[Req]{
		Method:  method,
		URL:     endpoint,
		Headers: headers,
		Data:    requestData,
		Timeout: 30 * time.Second,
	}

	encoder := JSONRequestEncoder[Req]{}
	decoder := JSONResponseDecoder[Resp]{}

	return MakeTypedRequest(httpInstance, config, encoder, decoder)
}

// MakeTypedRequestAsync performs an asynchronous typed HTTP request
func MakeTypedRequestAsync[Req, Resp any](
	httpInstance *HttpInstnace,
	config TypedRequest[Req],
	encoder RequestEncoder[Req],
	decoder ResponseDecoder[Resp],
	callback func(*TypedResponse[Resp]),
) {
	go func() {
		response := MakeTypedRequest(httpInstance, config, encoder, decoder)
		if callback != nil {
			callback(response)
		}
	}()
}

// MakeTypedRequestBatch performs multiple typed requests concurrently
func MakeTypedRequestBatch[Req, Resp any](
	httpInstance *HttpInstnace,
	configs []TypedRequest[Req],
	encoder RequestEncoder[Req],
	decoder ResponseDecoder[Resp],
) []*TypedResponse[Resp] {
	responses := make([]*TypedResponse[Resp], len(configs))
	var wg sync.WaitGroup

	for i, config := range configs {
		wg.Add(1)
		go func(index int, cfg TypedRequest[Req]) {
			defer wg.Done()
			responses[index] = MakeTypedRequest(httpInstance, cfg, encoder, decoder)
		}(i, config)
	}

	wg.Wait()
	return responses
}

// ProxmoxTypedJSONRequest performs a typed Proxmox API request with JSON payload using sonic
func ProxmoxTypedJSONRequest[Req, Resp any](
	httpInstance *HttpInstnace,
	method RequestMethod,
	endpoint string,
	requestData Req,
) *TypedResponse[Resp] {
	return MakeJSONRequest[Req, Resp](httpInstance, method, endpoint, requestData, nil)
}

// ConvertJSONResponse converts a regular Response to a typed response using sonic
func ConvertJSONResponse[T any](response *Response) *TypedResponse[T] {
	typedResponse := &TypedResponse[T]{
		StatusCode: response.StatusCode,
		Headers:    response.Headers,
		RawBody:    response.Body,
		Error:      response.Error,
	}

	if response.Error == nil && len(response.Body) > 0 {
		var data T
		if err := sonic.Unmarshal(response.Body, &data); err != nil {
			typedResponse.Error = err
		} else {
			typedResponse.Data = data
		}
	}

	return typedResponse
}

// MarshalToJSON marshals data to JSON using sonic for high performance
func MarshalToJSON[T any](data T) ([]byte, error) {
	return sonic.Marshal(data)
}

// UnmarshalFromJSON unmarshals JSON data using sonic for high performance
func UnmarshalFromJSON[T any](data []byte) (T, error) {
	var result T
	err := sonic.Unmarshal(data, &result)
	return result, err
}
