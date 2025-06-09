package proxmox

import (
	"github.com/valyala/fasthttp"

	"yoth.dev/proxmox-sdk-go/internal"
)

type Proxmox struct {
	fasthttpClient *fasthttp.Client
}

func New(options ...Option) *Proxmox {
	proxmox := &Proxmox{
		fasthttpClient: internal.CreateDefaultHTTPClient(),
	}

	for _, option := range options {
		option.apply(proxmox)
	}

	return proxmox
}
