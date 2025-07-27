package road

import (
	"fmt"

	mapv2 "git.fiblab.net/sim/protos/v2/go/city/map/v2"
	"git.fiblab.net/sim/simulet-go/entity"
)

// Road 道路实体
// 功能：表示地图中的道路，包含车道集合、路口连接、交通状态等信息
type Road struct {
	ctx entity.ITaskContext

	id           int32
	laneIDs      []int32
	name         string
	drivingLanes []entity.ILane         // 行车道，按从左到右排序
	walkingLanes []entity.ILane         // 人行道，按从左到右排序
	lanes        map[int32]entity.ILane // 车道id->车道指针映射表

	drivingPredecessor entity.IJunction // 前驱路口
	drivingSuccessor   entity.IJunction // 后继路口

	originalMaxV float64 // 道路最大车速均值
}

// newRoad 创建并初始化一个新的Road实例
// 功能：根据基础数据创建Road对象，初始化车道、车速、类型分类等配置
// 参数：ctx-任务上下文，base-基础Road数据，laneManager-车道管理器
// 返回：初始化完成的Road实例
// 说明：按车道类型分类存储，计算平均最大车速
func newRoad(ctx entity.ITaskContext, base *mapv2.Road, laneManager entity.ILaneManager) *Road {
	r := &Road{
		ctx:     ctx,
		id:      base.Id,
		name:    base.Name,
		laneIDs: base.LaneIds,
		lanes:   make(map[int32]entity.ILane),
	}

	// 道路车速、长度
	drivingLaneCount := 0
	r.originalMaxV = .0
	for i, laneID := range r.laneIDs {
		lane := laneManager.Get(laneID)
		r.lanes[laneID] = lane
		lane.SetParentRoadWhenInit(r, i)
		switch lane.Type() {
		case mapv2.LaneType_LANE_TYPE_DRIVING:
			r.drivingLanes = append(r.drivingLanes, lane)
			r.originalMaxV += lane.MaxV()
			drivingLaneCount++
		case mapv2.LaneType_LANE_TYPE_WALKING:
			r.walkingLanes = append(r.walkingLanes, lane)
		case mapv2.LaneType_LANE_TYPE_RAIL_TRANSIT:
		default:
			log.Panicf("Unknown lane type: %d", lane.Type())
		}
	}

	return r
}

// initAfterJunction 在Junction初始化后设置Road的路口连接关系
// 功能：根据车道的连接关系确定Road的前驱和后继路口
// 参数：junctionManager-Junction管理器
// 说明：验证前驱和后继路口的唯一性，确保Road连接关系正确
func (r *Road) initAfterJunction(_ entity.IJunctionManager) {
	// 路口
	for _, lane := range r.drivingLanes {
		for _, pre := range lane.Predecessors() {
			junc := pre.Lane.ParentJunction()
			if junc == nil {
				log.Panicf("Lane %d:%d's predecessor is not in junction", r.id, pre.Lane.ID())
			}
			if r.drivingPredecessor == nil {
				// 设置前驱路口
				r.drivingPredecessor = junc
			} else if r.drivingPredecessor != junc {
				// 检查前驱路口是否唯一
				log.Panicf("Road %d's predecessor is not unique: %d v.s. %d", r.id, r.drivingPredecessor.ID(), junc.ID())
			}
		}
		for _, suc := range lane.Successors() {
			junc := suc.Lane.ParentJunction()
			if junc == nil {
				log.Panicf("Lane %d:%d's successor is not in junction", r.id, suc.Lane.ID())
			}
			if r.drivingSuccessor == nil {
				// 设置后继路口
				r.drivingSuccessor = junc
			} else if r.drivingSuccessor != junc {
				// 检查后继路口是否唯一
				log.Panicf("Road %d's successor is not unique: %d v.s. %d", r.id, r.drivingSuccessor.ID(), junc.ID())
			}
		}
	}
}

// ID 获取Road的唯一标识符
// 功能：返回Road的ID，用于标识和查找特定的Road
// 返回：Road的ID，如果Road为nil则返回-1
func (r *Road) ID() int32 {
	if r == nil {
		return -1
	}
	return r.id
}

// String 获取Road的字符串表示
// 功能：返回Road的字符串描述，用于调试和日志输出
// 返回：Road的字符串表示
func (r *Road) String() string {
	return fmt.Sprintf("Road %d", r.id)
}

// Lanes 获取Road的所有车道映射
// 功能：返回Road内所有车道的映射表，以车道ID为键
// 返回：车道ID到车道对象的映射
func (r *Road) Lanes() map[int32]entity.ILane {
	return r.lanes
}

// RightestDrivingLane 获取最右侧的行车道（最靠近路边）
// 功能：返回最右侧的行车道，通常用于行人过街投影等场景
// 返回：最右侧的行车道，如果无行车道则panic
func (r *Road) RightestDrivingLane() entity.ILane {
	return r.drivingLanes[len(r.drivingLanes)-1]
}

// DrivingPredecessor 获取前驱Junction
// 功能：返回Road的前驱路口，即车辆进入Road的路口
// 返回：前驱路口对象
func (r *Road) DrivingPredecessor() entity.IJunction {
	return r.drivingPredecessor
}

// DrivingSuccessor 获取后继Junction
// 功能：返回Road的后继路口，即车辆离开Road的路口
// 返回：后继路口对象
func (r *Road) DrivingSuccessor() entity.IJunction {
	return r.drivingSuccessor
}

// ProjectToNearestDrivingLane 从步行道投影到最近的行车道
// 功能：将步行道上的位置投影到最近的行车道上，用于行人过街计算
// 参数：walkingLane-步行道，s-步行道上的位置
// 返回：投影后的行车道和位置，如果参数无效则panic
// 说明：投影使用最右侧行车道作为目标车道
func (r *Road) ProjectToNearestDrivingLane(walkingLane entity.ILane, s float64) (entity.ILane, float64) {
	if walkingLane.ParentRoad() != r {
		log.Panicf("Road %d does not contain Lane %d", r.id, walkingLane.ID())
	}
	if walkingLane.Type() != mapv2.LaneType_LANE_TYPE_WALKING {
		log.Panicf("Lane %d is not a walking lane", walkingLane.ID())
	}
	drivingLane := r.RightestDrivingLane()
	drivingS := drivingLane.ProjectFromLane(walkingLane, s)
	return drivingLane, drivingS
}

// ProjectToNearestWalkingLane 从行车道投影到最近的步行道
// 功能：将行车道上的位置投影到最近的步行道上，用于车辆停车计算
// 参数：drivingLane-行车道，s-行车道上的位置
// 返回：投影后的步行道和位置，如果无步行道则返回nil和0
// 说明：投影使用第一个步行道作为目标车道
func (r *Road) ProjectToNearestWalkingLane(drivingLane entity.ILane, s float64) (entity.ILane, float64) {
	if drivingLane.ParentRoad() != r {
		log.Panicf("Road %d does not contain Lane %d", r.id, drivingLane.ID())
	}
	if drivingLane.Type() != mapv2.LaneType_LANE_TYPE_DRIVING {
		log.Panicf("Lane %d is not a driving lane", drivingLane.ID())
	}
	if len(r.walkingLanes) == 0 {
		return nil, 0
	}
	walkingLane := r.walkingLanes[0]
	walkingS := walkingLane.ProjectFromLane(drivingLane, s)
	return walkingLane, walkingS
}

// MaxV 获取道路限速（车道限速的最大值）
// 功能：返回道路的设计最大车速，基于所有行车道的平均限速
// 返回：道路最大车速
func (r *Road) MaxV() float64 {
	return r.originalMaxV
}

// GetAvgDrivingL 获取道路行车道平均长度
// 功能：计算所有行车道的平均长度
// 返回：行车道平均长度
func (r *Road) GetAvgDrivingL() float64 {
	sumL := .0
	for _, l := range r.drivingLanes {
		sumL += l.Length()
	}
	return sumL / float64(len(r.drivingLanes))
}

// Name 获取Road的名称
// 功能：返回Road的名称，用于显示和标识
// 返回：Road的名称
func (r *Road) Name() string {
	return r.name
}
