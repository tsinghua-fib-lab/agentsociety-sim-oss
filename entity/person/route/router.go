package route

import (
	"git.fiblab.net/sim/simulet-go/entity"
	"git.fiblab.net/sim/simulet-go/utils/input"
)

// New 初始化导航服务
func New(input *input.Input) entity.IRouter {
	return NewLocalRouter(input.Map)
}
