package internal

import (
	"time"

	"github.com/valyala/fasthttp"
)

type connectorOption interface {
	apply(*Connector)
}

type optionFunc func(*Connector)

func (f optionFunc) apply(c *Connector) {
	f(c)
}

func WithClient(client *fasthttp.Client) connectorOption {
	return optionFunc(func(c *Connector) {
		c.Client = client
	})
}

type Connector struct {
	Client *fasthttp.Client
}

func NewConnector(options ...connectorOption) *Connector {
	connector := Connector{
		Client: CreateDefaultClient(),
	}

	for _, option := range options {
		option.apply(&connector)
	}

	return &connector
}

func CreateDefaultClient() *fasthttp.Client {
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
