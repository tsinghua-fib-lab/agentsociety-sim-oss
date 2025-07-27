package junction

import (
	"errors"

	mapv2 "git.fiblab.net/sim/protos/v2/go/city/map/v2"
	"git.fiblab.net/sim/simulet-go/entity"
	"git.fiblab.net/sim/simulet-go/entity/junction/trafficlight"
	"git.fiblab.net/sim/simulet-go/utils/randengine"
	"github.com/samber/lo"
)

var (
	ErrDisabledTrafficLight = errors.New("traffic light is disabled for the junction")
)

type laneGroupKey struct {
	InRoad  entity.IRoad
	OutRoad entity.IRoad
}

type laneGroupValue struct {
	InAngle  float64
	OutAngle float64
	Lanes    []entity.ILane
}

type Junction struct {
	ctx entity.ITaskContext

	id                int32
	laneIDs           []int32
	trafficLight      ITrafficLight          // 信号灯模块
	lanes             map[int32]entity.ILane // 车道id->车道指针映射表
	drivingLanes      []entity.ILane         // 行车道
	drivingLaneGroups map[laneGroupKey]*laneGroupValue
	preDrivingLanes   []entity.ILane       // 前驱行车道
	phases            [][]mapv2.LightState // 最大压力信控的可用相位
	fixedProgram      *mapv2.TrafficLight

	generator *randengine.Engine
}

// newJunction 创建并初始化一个新的Junction实例
// 功能：根据基础数据创建Junction对象，初始化车道、信号灯、车道组、碰撞检测等配置
// 参数：ctx-任务上下文，base-基础Junction数据，laneManager-车道管理器，roadManager-道路管理器
// 返回：初始化完成的Junction实例
func newJunction(
	ctx entity.ITaskContext,
	base *mapv2.Junction,
	laneManager entity.ILaneManager,
	roadManager entity.IRoadManager,
) *Junction {
	// 初始化Junction基础结构
	j := &Junction{
		ctx:               ctx,
		id:                base.Id,
		laneIDs:           base.LaneIds,
		lanes:             make(map[int32]entity.ILane),
		drivingLanes:      make([]entity.ILane, 0),
		drivingLaneGroups: make(map[laneGroupKey]*laneGroupValue),
		preDrivingLanes:   make([]entity.ILane, 0),
		phases:            make([][]mapv2.LightState, 0),
		fixedProgram:      base.FixedProgram,
		generator:         randengine.New(uint64(base.Id)),
	}

	// 初始化车道映射和信号灯设置
	lanes := make([]entity.ILaneTrafficLightSetter, 0)
	for _, laneID := range j.laneIDs {
		lane := laneManager.Get(laneID)
		lane.SetParentJunctionWhenInit(j)
		j.lanes[laneID] = lane
		lanes = append(lanes, lane)
	}

	// 初始化车道组映射
	for _, g := range base.DrivingLaneGroups {
		inRoad := roadManager.Get(g.InRoadId)
		outRoad := roadManager.Get(g.OutRoadId)
		key := laneGroupKey{
			InRoad:  inRoad,
			OutRoad: outRoad,
		}
		value := &laneGroupValue{
			InAngle:  g.InAngle,
			OutAngle: g.OutAngle,
			Lanes:    make([]entity.ILane, len(g.LaneIds)),
		}
		for i, laneID := range g.LaneIds {
			l := j.lanes[laneID]
			value.Lanes[i] = l
			if l.Type() == mapv2.LaneType_LANE_TYPE_DRIVING {
				j.drivingLanes = append(j.drivingLanes, l)
			}
		}
		j.drivingLaneGroups[key] = value
	}

	// 初始化前驱行车道
	for _, l := range j.drivingLanes {
		pre, err := l.UniquePredecessor()
		if err != nil {
			log.Panicf("get unique predecessor error: %v", err)
		}
		j.preDrivingLanes = append(j.preDrivingLanes, pre)
	}
	j.preDrivingLanes = lo.Uniq(j.preDrivingLanes)

	// 转换可用相位数据
	j.phases = lo.Map(base.Phases, func(p *mapv2.AvailablePhase, _ int) []mapv2.LightState {
		return p.States
	})

	// 信号灯初始化逻辑
	if ctx.RuntimeConfig().C.PreferFixedLight && j.fixedProgram != nil && len(j.fixedProgram.Phases) > 0 {
		// 使用固定信号灯程序
		j.trafficLight = trafficlight.NewLocalTrafficLight(ctx, j.id, lanes)
		if err := j.trafficLight.Set(j.fixedProgram); err != nil {
			log.Panicf("set fixed program error: %v", err)
		}
	} else {
		// 使用最大压力信号灯
		if len(j.phases) > 0 {
			j.trafficLight = trafficlight.NewMaxPressureTrafficLight(j.id, lanes, j.phases)
		}
	}

	return j
}

// prepare 准备阶段，处理信号灯的准备工作
// 功能：执行信号灯的准备工作，处理各种写入缓冲区操作，更新排队情况等统计信息
func (j *Junction) prepare() {
	if j.trafficLight != nil {
		j.trafficLight.Prepare()
	}
}

// update 更新阶段，执行Junction的模拟逻辑
// 功能：执行信号灯的更新逻辑，更新信号灯状态
// 参数：dt-时间步长
func (j *Junction) update(dt float64) {
	if j.trafficLight != nil {
		j.trafficLight.Update(dt)
	}
}

// ID 获取Junction的唯一标识符
// 功能：返回Junction的ID，用于标识和查找特定的Junction
// 返回：Junction的ID，如果Junction为nil则返回-1
func (j *Junction) ID() int32 {
	if j == nil {
		return -1
	}
	return j.id
}

// Lanes 获取Junction内的所有车道映射
// 功能：返回Junction内所有车道的映射表，以车道ID为键
// 返回：车道ID到车道对象的映射
func (j *Junction) Lanes() map[int32]entity.ILane {
	return j.lanes
}

// DrivingLaneGroup 根据入道路和出道路获取Junction内的行车道组与角度
// 功能：根据指定的入道路和出道路查找对应的行车道组信息
// 参数：inRoad-入道路，outRoad-出道路
// 返回：车道列表、入角度、出角度、是否找到
func (j *Junction) DrivingLaneGroup(inRoad, outRoad entity.IRoad) (lanes []entity.ILane, inAngle, outAngle float64, ok bool) {
	key := laneGroupKey{
		InRoad:  inRoad,
		OutRoad: outRoad,
	}
	value, ok := j.drivingLaneGroups[key]
	if !ok {
		return
	}
	return value.Lanes, value.InAngle, value.OutAngle, true
}

// HasTrafficLight 判断是否有信号灯
// 功能：检查当前Junction是否有可用的信号灯
// 返回：true表示有信号灯且正常工作，false表示没有信号灯或信号灯失效
func (j *Junction) HasTrafficLight() bool {
	return j.trafficLight != nil && j.trafficLight.Ok()
}

// SetTrafficLight 设置信号灯程序
// 功能：为Junction设置新的信号灯程序
// 参数：tl-信号灯程序数据
// 返回：设置结果，如果信号灯被禁用则返回错误
func (j *Junction) SetTrafficLight(tl *mapv2.TrafficLight) error {
	if j.trafficLight == nil {
		// 信控被禁用，无法设置信号灯
		return ErrDisabledTrafficLight
	}
	return j.trafficLight.Set(tl)
}

// unsetTrafficLight 取消信号灯程序
// 功能：取消当前Junction的信号灯程序，使其变为全绿灯状态
// 返回：操作结果，如果信号灯被禁用则返回错误
func (j *Junction) unsetTrafficLight() error {
	if j.trafficLight == nil {
		// 信控被禁用，无法设置信号灯
		return ErrDisabledTrafficLight
	}
	j.trafficLight.Unset()
	return nil
}

// setPhase 设置信号灯相位
// 功能：设置信号灯到指定的相位和剩余时间
// 参数：offset-相位偏移，remainingTime-剩余时间
// 返回：设置结果，如果信号灯被禁用则返回错误
func (j *Junction) setPhase(offset int32, remainingTime float64) error {
	if j.trafficLight == nil {
		// 信控被禁用，无法设置信号灯
		return ErrDisabledTrafficLight
	}
	j.trafficLight.SetPhase(offset, remainingTime)
	return nil
}

// setStatus 设置信号灯状态
// 功能：设置信号灯的开关状态
// 参数：ok-信号灯状态，true表示正常工作，false表示失效（全绿灯）
// 返回：设置结果，如果信号灯被禁用则返回错误
func (j *Junction) setStatus(ok bool) error {
	if j.trafficLight == nil {
		// 信控被禁用，无法设置信号灯
		return ErrDisabledTrafficLight
	}
	j.trafficLight.SetOk(ok)
	return nil
}
