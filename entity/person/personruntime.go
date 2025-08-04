package person

import (
	"git.fiblab.net/general/common/v2/geometry"
	geov2 "git.fiblab.net/sim/protos/v2/go/city/geo/v2"
	personv2 "git.fiblab.net/sim/protos/v2/go/city/person/v2"
	"github.com/tsinghua-fib-lab/agentsociety-sim-oss/entity"
)

// lcRuntime 变道运行时数据结构
// 功能：记录车辆变道过程中的状态信息，包括变道目标、位置映射、转向角度等
type lcRuntime struct {
	IsLC bool // 变道状态
	// ATTENTION: 重新定义shadow为变道前的车道
	ShadowLane     entity.ILane // 变道前所在车道
	ShadowS        float64      // 映射到变道前所在车道的位置
	Yaw            float64      // 变道过程车头相对于前进方向的偏转角（弧度，总是为正，0代表不转向）
	CompletedRatio float64      // 已完成的变道比例
}

// InShadowLane 检查是否占据阴影车道
// 功能：判断车辆是否仍在原车道中（变道未完成）
// 返回：true表示仍在原车道，false表示已进入新车道
// 说明：当变道完成比例小于阈值时认为仍在原车道
func (lc *lcRuntime) InShadowLane() bool {
	return lc.IsLC && lc.CompletedRatio < lcInOldLaneRatio
}

// runtime 人员运行时数据结构
// 功能：记录人员在模拟过程中的所有运行时状态信息
// 说明：该数据结构需要可以被直接复制，不应产生浅拷贝带来的副作用
type runtime struct {
	// 上一时刻状态，室外由runtime提供，室内由SetSnapshotByAoi触发从runtime复制
	Status personv2.Status

	// 上一时刻的trip是否结束
	IsTripEnd bool

	// 供输出或外部接口调用的人的数据快照，与status对应

	XYZ  geometry.Point // 位置
	V    float64        // 速度
	Lane entity.ILane   // 所在车道id
	S    float64        // 车道上的位置
	Aoi  entity.IAoi    // 所在Aoi

	// 车辆的Runtime

	Action Action    // 车辆行为
	LC     lcRuntime // 以下成员在变道时使用，仅当IsLC == true不为空时有意义

	// 行人的Runtime

	IsForward bool // 是否正向行走
}

// clearLaneChange 清除变道状态
// 功能：重置变道相关的运行时数据，结束变道过程
// 说明：将变道状态重置为空结构体，清除所有变道信息
func (rt *runtime) clearLaneChange() {
	rt.LC = lcRuntime{}
}

// toPbPosition 转换为protobuf位置格式
// 功能：将内部位置数据转换为protobuf格式的位置信息
// 参数：ctx-任务上下文，用于坐标转换
// 返回：protobuf格式的位置信息，包含XY坐标、经纬度和车道/AOI位置
// 说明：同时包含多种坐标系统和位置引用，确保数据的完整性
func (rt *runtime) toPbPosition(ctx entity.ITaskContext) *geov2.Position {
	z := rt.XYZ.Z
	position := &geov2.Position{
		XyPosition: &geov2.XYPosition{X: rt.XYZ.X, Y: rt.XYZ.Y, Z: &z},
	}
	if rt.Lane != nil {
		position.LanePosition = &geov2.LanePosition{LaneId: rt.Lane.ID(), S: rt.S}
	}
	if rt.Aoi != nil {
		position.AoiPosition = &geov2.AoiPosition{AoiId: rt.Aoi.ID()}
	}
	return position
}

// ToPb 转换为protobuf人员运动数据
// 功能：将运行时数据转换为protobuf格式的人员运动信息
// 参数：ctx-任务上下文，self-人员实体
// 返回：protobuf格式的人员运动数据
// 说明：包含位置、速度、加速度、方向、活动等完整信息
func (rt *runtime) ToPb(ctx entity.ITaskContext, self entity.IPerson) *personv2.PersonMotion {
	pb := &personv2.PersonMotion{
		Id:       self.ID(),
		Status:   rt.Status,
		Position: rt.toPbPosition(ctx),
		V:        rt.V,
		A:        rt.Action.A,
		L:        self.Length(),
	}
	return pb
}

// resetByPbPosition 根据protobuf位置重置运行时数据
// 功能：根据给定的位置信息重置人员的运行时状态
// 参数：ctx-任务上下文，pos-protobuf格式的位置信息
// 说明：
// 1. 重置所有运行时数据为初始状态
// 2. 根据位置类型（车道或AOI）设置相应的位置信息
// 3. 计算方向角和位置坐标
// 4. 对于车道位置，需要考虑行人偏移和方向计算
func (rt *runtime) resetByPbPosition(ctx entity.ITaskContext, pos *geov2.Position) {
	*rt = runtime{}
	rt.Status = personv2.Status_STATUS_SLEEP

	if pos.LanePosition != nil {
		rt.Lane = ctx.LaneManager().Get(pos.LanePosition.LaneId)
		rt.S = pos.LanePosition.S
		rt.XYZ = rt.Lane.GetPositionByS(rt.S)
	} else if pos.AoiPosition != nil {
		aoi := ctx.AoiManager().Get(pos.AoiPosition.AoiId)
		rt.Aoi = aoi
		if pos.XyPosition != nil {
			rt.XYZ = geometry.NewPointFromPb(pos.XyPosition)
		} else {
			rt.XYZ = aoi.Centroid()
		}
	} else {
		log.Panic("person: invalid position")
	}
}

// forceClearVehicleRuntime 强制清除车辆运行时数据
// 功能：在特殊情况下强制清除车辆的运行时状态
// 参数：forceEnd-是否强制结束当前状态
// 说明：
// 1. 如果强制结束，清除变道状态和目标
// 2. 用于处理异常情况下的状态重置
// 3. 确保车辆状态的一致性
func (rt *runtime) forceClearVehicleRuntime(forceEnd bool) {
	if forceEnd {
		// 清除变道
		rt.clearLaneChange()
		rt.Action.LCTarget = nil
	}
}
