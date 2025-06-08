package proxmox

import (
	"github.com/valyala/fasthttp"
)

type Option interface {
	apply(*Proxmox)
}

type optionFunc func(*Proxmox)

func (f optionFunc) apply(p *Proxmox) {
	f(p)
}

func WithFastHTTPClient(c *fasthttp.Client) Option {
	return optionFunc(func(p *Proxmox) {
		p.fasthttpClient = c
	})
}
