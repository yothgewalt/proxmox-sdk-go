package internal

import "github.com/valyala/fasthttp"

// WithHttpClient sets the HTTP client for the HttpInstnace
func WithHttpClient(client *fasthttp.Client) Option[HttpInstnace] {
	return OptionFunc[HttpInstnace](func(c *HttpInstnace) {
		c.HttpClient = client
	})
}

// WithRequestPoolSize sets the size of the request pool
func WithWorkerPoolSize(size int) Option[HttpInstnace] {
	return OptionFunc[HttpInstnace](func(c *HttpInstnace) {
		c.workerPool = make(chan struct{}, size)
	})
}
