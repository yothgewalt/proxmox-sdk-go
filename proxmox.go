package proxmox

import (
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
