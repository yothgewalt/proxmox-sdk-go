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
	defaultHttpClient := internal.CreateDefaultClient()

	proxmox := proxmoxDependency{
		httpClient: defaultHttpClient,
	}

	for _, option := range options {
		option.apply(&proxmox)
	}

	return &proxmox
}
