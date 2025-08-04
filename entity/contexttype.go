package entity

import (
	routingv2 "git.fiblab.net/sim/protos/v2/go/city/routing/v2"
	"github.com/tsinghua-fib-lab/agentsociety-sim-oss/clock"
	"github.com/tsinghua-fib-lab/agentsociety-sim-oss/utils/config"
)

// 导航模块接口
type IRouter interface {
	// 路径规划（回调版本）
	GetRoute(in *routingv2.GetRouteRequest, process func(res *routingv2.GetRouteResponse)) chan struct{}
	// 路径规划（同步版本）
	GetRouteSync(in *routingv2.GetRouteRequest) *routingv2.GetRouteResponse
}

type ITaskContext interface {
	Clock() *clock.Clock
	LaneManager() ILaneManager
	AoiManager() IAoiManager
	RoadManager() IRoadManager
	JunctionManager() IJunctionManager
	PersonManager() IPersonManager
	RuntimeConfig() *config.RuntimeConfig
	Router() IRouter
}
