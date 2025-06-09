package proxmox

import (
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
