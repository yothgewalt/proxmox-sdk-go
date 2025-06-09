package internal

import (
	"time"

	"github.com/valyala/fasthttp"
)

type HttpInstnace struct {
	HttpClient *fasthttp.Client
}

func WithClient(client *fasthttp.Client) Option[HttpInstnace] {
	return OptionFunc[HttpInstnace](func(c *HttpInstnace) {
		c.HttpClient = client
	})
}

func NewHttpInstance(options ...Option[HttpInstnace]) *HttpInstnace {
	connector := HttpInstnace{
		HttpClient: createDefaultHttpClient(),
	}

	ApplyOptions(&connector, options...)

	return &connector
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
