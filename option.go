package proxmox

import (
	"github.com/valyala/fasthttp"
)

type Option interface {
	apply(*proxmoxDependency)
}

type optionFunc func(*proxmoxDependency)

func (f optionFunc) apply(p *proxmoxDependency) {
	f(p)
}

func WithHTTPClient(c *fasthttp.Client) Option {
	return optionFunc(func(p *proxmoxDependency) {
		p.httpClient = c
	})
}
