package route

import (
	"github.com/tsinghua-fib-lab/agentsociety-sim-oss/entity"
	"github.com/tsinghua-fib-lab/agentsociety-sim-oss/utils/input"
)

// New 初始化导航服务
func New(input *input.Input) entity.IRouter {
	return NewLocalRouter(input.Map)
}
