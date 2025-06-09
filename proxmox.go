package proxmox

import (
	"github.com/valyala/fasthttp"

	"yoth.dev/proxmox-sdk-go/internal"
)

type Proxmox interface{}

type proxmoxDependency struct {
	httpInstance *internal.HttpInstnace
}

func New(options ...internal.Option[proxmoxDependency]) Proxmox {
	defaultHttpInstance := internal.NewHttpInstance()

	proxmox := proxmoxDependency{
		httpInstance: defaultHttpInstance,
	}

	internal.ApplyOptions(&proxmox, options...)

	return &proxmox
}

func WithHTTPClient(c *fasthttp.Client) internal.Option[proxmoxDependency] {
	return internal.OptionFunc[proxmoxDependency](func(p *proxmoxDependency) {
		p.httpInstance = internal.NewHttpInstance(internal.WithHttpClient(c))
	})
}
