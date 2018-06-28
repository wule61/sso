package system

import (
	"github.com/micro-plat/sso/modules/member"

	"github.com/micro-plat/hydra/component"
	"github.com/micro-plat/hydra/context"
	"github.com/micro-plat/sso/modules/system"
)

type SystemHandler struct {
	container component.IContainer
	sys       system.ISystem
}

func NewSystemHandler(container component.IContainer) (u *SystemHandler) {
	return &SystemHandler{
		container: container,
		sys:       system.NewSystem(container),
	}
}

func (u *SystemHandler) Handle(ctx *context.Context) (r interface{}) {
	sysid := member.Get(ctx).SystemID
	data, err := u.sys.Query(sysid)
	if err != nil {
		return err
	}
	return data
}