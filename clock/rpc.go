package clock

import (
	"context"
	"net/http"

	"connectrpc.com/connect"
	clockv1 "git.fiblab.net/sim/protos/v2/go/city/clock/v1"
	"git.fiblab.net/sim/protos/v2/go/city/clock/v1/clockv1connect"
	"git.fiblab.net/sim/syncer/v3"
)

// Register 将ClockService注册到sidecar
// 功能：注册时钟服务的RPC处理器到sidecar中
// 参数：sidecar-sidecar实例
// 说明：使时钟服务可以通过RPC接口被外部访问，支持分布式仿真
func (c *Clock) Register(sidecar *syncer.Sidecar) {
	sidecar.Register(
		clockv1connect.ClockServiceName,
		func(opts ...connect.HandlerOption) (pattern string, handler http.Handler) {
			return clockv1connect.NewClockServiceHandler(c, opts...)
		},
	)
}

// Now 获取当前仿真时间
// 功能：RPC接口，返回当前仿真天数和时间
// 参数：ctx-上下文，in-请求参数
// 返回：当前仿真天数和时间的响应
// 说明：提供外部系统查询当前仿真时间的接口，支持分布式仿真的时间同步
func (c *Clock) Now(ctx context.Context, in *connect.Request[clockv1.NowRequest]) (*connect.Response[clockv1.NowResponse], error) {
	return connect.NewResponse(&clockv1.NowResponse{
		T: c.T,
	}), nil
}
