package proxmox

import (
	"github.com/valyala/fasthttp"

	"yoth.dev/proxmox-sdk-go/internal"
)

type Proxmox interface{}

type proxmoxDependency struct {
	httpClient *fasthttp.Client
}

func New(options ...option) Proxmox {
	proxmox := proxmoxDependency{
		httpClient: internal.CreateDefaultHTTPClient(),
	}

	for _, option := range options {
		option.apply(&proxmox)
	}

	return &proxmox
}
