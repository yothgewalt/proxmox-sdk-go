package proxmox

import (
	"github.com/valyala/fasthttp"

	"yoth.dev/proxmox-sdk-go/internal"
)

// WithBaseEndpointPath sets a custom base endpoint path for the Proxmox API
func WithBaseEndpointPath(basePath string) internal.Option[proxmoxDependency] {
	return internal.OptionFunc[proxmoxDependency](func(p *proxmoxDependency) {
		p.httpInstance = internal.NewHttpInstance(internal.WithBaseEndpointPath(basePath))
	})
}

// WithHTTPClient sets the HTTP client for the Proxmox dependency
func WithHTTPClient(c *fasthttp.Client) internal.Option[proxmoxDependency] {
	return internal.OptionFunc[proxmoxDependency](func(p *proxmoxDependency) {
		p.httpInstance = internal.NewHttpInstance(internal.WithHttpClient(c))
	})
}

// WithHTTPClientAndBaseEndpoint sets both the HTTP client and base endpoint path
func WithHTTPClientAndBaseEndpoint(client *fasthttp.Client, basePath string) internal.Option[proxmoxDependency] {
	return internal.OptionFunc[proxmoxDependency](func(p *proxmoxDependency) {
		p.httpInstance = internal.NewHttpInstance(
			internal.WithHttpClient(client),
			internal.WithBaseEndpointPath(basePath),
		)
	})
}
