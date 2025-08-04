package route

import (
	"git.fiblab.net/general/common/v2/geometry"
	geov2 "git.fiblab.net/sim/protos/v2/go/city/geo/v2"
	"github.com/tsinghua-fib-lab/moss-agentsociety-go/entity"
)

// newRoutePosition 根据protobuf位置创建路由位置
// 功能：将protobuf格式的位置信息转换为内部路由位置结构
// 参数：ctx-任务上下文，pb-protobuf位置信息
// 返回：内部路由位置结构
// 说明：支持AOI位置和车道位置两种类型，如果位置无效则panic
func newRoutePosition(ctx entity.ITaskContext, pb *geov2.Position) entity.RoutePosition {
	p := entity.RoutePosition{}
	if pb.AoiPosition != nil {
		p.Aoi = ctx.AoiManager().Get(pb.AoiPosition.AoiId)
		if pb.XyPosition != nil {
			xy := geometry.NewPointFromPb(pb.XyPosition)
			p.XY = &xy
		}
	} else if pb.LanePosition != nil {
		p.Lane = ctx.LaneManager().Get(pb.LanePosition.LaneId)
		p.S = pb.LanePosition.S
	} else {
		log.Panicf("invalid route position: %+v", pb)
	}
	return p
}

// newPbPosition 将内部路由位置转换为protobuf位置
// 功能：将内部路由位置结构转换为protobuf格式的位置信息
// 参数：rPos-内部路由位置结构
// 返回：protobuf位置信息
// 说明：支持AOI位置和车道位置两种类型，如果位置无效则panic
func newPbPosition(rPos entity.RoutePosition) *geov2.Position {
	pb := &geov2.Position{}
	if rPos.Aoi != nil {
		pb.AoiPosition = &geov2.AoiPosition{
			AoiId: rPos.Aoi.ID(),
		}
		if rPos.XY != nil {
			pb.XyPosition = &geov2.XYPosition{
				X: rPos.XY.X,
				Y: rPos.XY.Y,
				Z: &rPos.XY.Z,
			}
		}
	} else if rPos.Lane != nil {
		pb.LanePosition = &geov2.LanePosition{
			LaneId: rPos.Lane.ID(),
			S:      rPos.S,
		}
	} else {
		log.Panicf("invalid route position: %+v", rPos)
	}
	return pb
}
