package proxmox

import (
	"github.com/valyala/fasthttp"

	"yoth.dev/proxmox-sdk-go/internal"
)

// WithHTTPClient sets the HTTP client for the Proxmox dependency
func WithHTTPClient(c *fasthttp.Client) internal.Option[proxmoxDependency] {
	return internal.OptionFunc[proxmoxDependency](func(p *proxmoxDependency) {
		p.httpInstance = internal.NewHttpInstance(internal.WithHttpClient(c))
	})
}
