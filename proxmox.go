package proxmox

import (
	"time"

	"github.com/valyala/fasthttp"
)

type Proxmox struct {
	fasthttpClient *fasthttp.Client
}

func New(options ...Option) *Proxmox {
	proxmox := &Proxmox{
		fasthttpClient: getDefaultFastHTTPClient(),
	}

	for _, option := range options {
		option.apply(proxmox)
	}

	return proxmox
}

func getDefaultFastHTTPClient() *fasthttp.Client {
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
