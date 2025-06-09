package internal

import (
	"time"

	"github.com/valyala/fasthttp"
)

// WithHttpClient sets the HTTP client for the HttpInstnace
func WithHttpClient(client *fasthttp.Client) Option[HttpInstnace] {
	return OptionFunc[HttpInstnace](func(c *HttpInstnace) {
		c.HttpClient = client
	})
}

// WithWorkerPoolSize sets the size of the worker pool
func WithWorkerPoolSize(size int) Option[HttpInstnace] {
	return OptionFunc[HttpInstnace](func(c *HttpInstnace) {
		if size > 0 {
			c.workerPool.Resize(size)
		}
	})
}

// WithWorkerPoolConfig sets the worker pool configuration
func WithWorkerPoolConfig(config WorkerPoolConfig) Option[HttpInstnace] {
	return OptionFunc[HttpInstnace](func(c *HttpInstnace) {
		c.workerPool = NewWorkerPool(config)
	})
}

// WithBaseEndpointPath sets a custom base endpoint path for the HttpInstance
func WithBaseEndpointPath(basePath string) Option[HttpInstnace] {
	return OptionFunc[HttpInstnace](func(c *HttpInstnace) {
		c.baseEndpointPath = basePath
	})
}

// WithBufferManager sets a custom buffer manager
func WithBufferManager(manager *RequestBufferManager) Option[HttpInstnace] {
	return OptionFunc[HttpInstnace](func(c *HttpInstnace) {
		c.bufferManager = manager
	})
}

// WithProxmoxDefaults applies optimized defaults for Proxmox API usage
func WithProxmoxDefaults() Option[HttpInstnace] {
	return OptionFunc[HttpInstnace](func(c *HttpInstnace) {
		c.workerPool = NewWorkerPool(WorkerPoolConfig{
			MaxWorkers:  50,
			TaskTimeout: 30 * time.Second,
		})

		c.bufferManager = NewRequestBufferManager()

		// Proxmox API endpoint
		c.baseEndpointPath = "/api2/json"

		c.HttpClient = &fasthttp.Client{
			ReadTimeout:  30 * time.Second,
			WriteTimeout: 15 * time.Second,

			MaxIdleConnDuration: 120 * time.Second,
			MaxConnDuration:     300 * time.Second,

			ReadBufferSize:      8192,
			WriteBufferSize:     4096,
			MaxResponseBodySize: 50 << 20,

			DisableHeaderNamesNormalizing: false,
			DisablePathNormalizing:        false,
		}
	})
}
