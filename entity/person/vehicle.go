package person

import (
	"math"

	"git.fiblab.net/general/common/v2/geometry"
	mapv2 "git.fiblab.net/sim/protos/v2/go/city/map/v2"
	"git.fiblab.net/sim/simulet-go/entity"
)

const (
	closeToEnd = 5 // 车辆到达终点的判定范围（米）
)

// vehicle 车辆实体数据结构
// 功能：管理车辆的所有属性和状态，包括控制、链表节点、控制器等
type vehicle struct {
	// Lane链表相关
	length           float64             // 车辆长度
	node, shadowNode *entity.VehicleNode // 主节点和影子节点（用于变道）
	controller       *controller         // 车辆控制器                                   float64        // 上次位移
}

// updateLaneVehicleNodes 更新车道车辆节点
// 功能：维护车辆在车道链表中的节点，处理车道切换和变道
// 参数：needIndexMaintenance-是否需要维护索引
// 算法说明：
// 1. 比较运行时和快照的车道信息
// 2. 如果需要维护索引：
//   - 处理主车道切换
//   - 处理变道影子节点
//   - 创建新节点避免并发问题
//
// 3. 如果不需要维护索引：
//   - 移除所有节点
//   - 清理变道状态
func (p *Person) updateLaneVehicleNodes(needIndexMaintenance bool) {
	var runtimeParentId, snapshotParentId, runtimeLaneId, snapshotLaneId int32
	if p.runtime.Lane != nil {
		runtimeParentId = p.runtime.Lane.ParentID()
		runtimeLaneId = p.runtime.Lane.ID()
	}
	if p.snapshot.Lane != nil {
		snapshotParentId = p.snapshot.Lane.ParentID()
		snapshotLaneId = p.snapshot.Lane.ID()
	}
	log.Debugf("updateLaneVehicleNodes %v need %v laneId (%v,%v) parentId (%v,%v) step %v",
		p.ID(), needIndexMaintenance,
		snapshotLaneId, runtimeLaneId,
		snapshotParentId, runtimeParentId, p.ctx.Clock().T,
	)
	if needIndexMaintenance {
		// 维护数据
		if p.snapshot.Lane != p.runtime.Lane {
			p.snapshot.Lane.RemoveVehicle(p.vehicle.node)
			// 换一个新的node来避免remove操作和add操作处理同一个对象需要保证先后顺序
			p.vehicle.node = newVehicleNode(p.runtime.S, p)
			p.runtime.Lane.AddVehicle(p.vehicle.node)
		}
		if !p.snapshot.LC.InShadowLane() && !p.runtime.LC.InShadowLane() {
			// do nothing
		} else if p.snapshot.LC.InShadowLane() && !p.runtime.LC.InShadowLane() {
			p.snapshot.LC.ShadowLane.RemoveVehicle(p.vehicle.shadowNode)
		} else if !p.snapshot.LC.InShadowLane() && p.runtime.LC.InShadowLane() {
			p.runtime.LC.ShadowLane.AddVehicle(p.vehicle.shadowNode)
		} else {
			if p.snapshot.LC.ShadowLane != p.runtime.LC.ShadowLane {
				p.snapshot.LC.ShadowLane.RemoveVehicle(p.vehicle.shadowNode)
				p.vehicle.shadowNode = newVehicleNode(p.runtime.LC.ShadowS, p)
				p.runtime.LC.ShadowLane.AddVehicle(p.vehicle.shadowNode)
			}
		}
	} else {
		// 不维护数据
		p.snapshot.Lane.RemoveVehicle(p.vehicle.node)
		if p.snapshot.LC.InShadowLane() {
			p.snapshot.LC.ShadowLane.RemoveVehicle(p.vehicle.shadowNode)
		}
	}
}

// updateVehicle 更新车辆状态
// 功能：执行车辆的主要更新逻辑，包括控制、停车、运动等
// 参数：dt-时间步长，vehControlChan-车辆控制通道，vehRouteChan-车辆路由通道
// 返回：isEnd-是否到达终点
// 算法说明：
// 1. 验证变道状态的一致性
// 2. 更新车辆控制器
// 3. 处理停车状态
// 4. 处理离开停车点
// 5. 强制结束处理
func (p *Person) updateVehicle(dt float64) (isEnd bool) {
	// DEBUG, node一致性
	if p.runtime.LC.InShadowLane() {
		if p.vehicle.shadowNode == nil || p.vehicle.shadowNode.Parent() == nil {
			log.Panicf("vehicle: vehicle %v shadowNode is nil", p.ID())
		}
	}
	p.runtime.Action = p.vehicle.controller.update(dt)
	// 到最后一个step了，不管到没到目的地，都进行清理操作
	forceEnd := p.ctx.Clock().InternalStep+1 == p.ctx.Clock().END_STEP
	p.runtime.forceClearVehicleRuntime(forceEnd)
	skipToEnd := p.refreshRuntime(p.runtime.Action, dt)
	reachTarget := p.checkCloseToEndAndRefreshRuntime(skipToEnd)
	if reachTarget || forceEnd {
		// 增量更新车道索引（不再维护数据）
		p.updateLaneVehicleNodes(false)
		return true
	}
	// 车道链表更新

	// 增量更新车道索引（维护数据）
	p.updateLaneVehicleNodes(true)
	return
}

// 计算本时刻的速度与移动距离
// v(t)=v(t-1)+acc*dt, ds=v(t-1)*dt+acc*dt*dt/2
func computeVAndDistance(v, a, dt float64) (float64, float64) {
	dv := a * dt
	if v+dv < 0 {
		// 刹车到停止
		return 0, v * v / 2 / -a
	}
	return v + dv, (v + dv/2) * dt
}

func (p *Person) refreshRuntime(ac Action, dt float64) (skipToEnd bool) {
	// ATTENTION: 注意v.runtime.Motion不是指针
	v, d := computeVAndDistance(p.V(), ac.A, dt)

	// 阿克曼转向动力学

	laneWidth := p.runtime.Lane.Width()
	if ac.LCTarget != nil {
		laneWidth = (p.runtime.Lane.Width() + ac.LCTarget.Width()) / 2
	}
	if p.runtime.LC.IsLC {
		laneWidth = (p.runtime.Lane.Width() + p.runtime.LC.ShadowLane.Width()) / 2
	}
	maxYaw := math.Min(math.Pi/6, math.Asin(laneWidth/(p.vehicleAttr.Length)))
	dYaw := d / (p.vehicleAttr.Length / 2) * math.Tan(ac.LCPhi)
	lcYaw := .0
	if p.runtime.LC.IsLC {
		lcYaw = p.runtime.LC.Yaw
	}
	oldLCYaw := lcYaw
	lcYaw += dYaw
	if lcYaw > maxYaw {
		lcYaw = oldLCYaw
	}

	// 计算横向距离和纵向偏移
	meanYaw := (oldLCYaw + lcYaw) / 2
	// 1. 横向距离
	dw := d * math.Sin(meanYaw)
	// 2. 纵向偏移
	ds := d * math.Cos(meanYaw)

	// 更新位置
	newRuntime := p.runtime
	if ac.LCTarget != nil {
		// 检查变道目标是否是车道
		if ac.LCTarget.Type() != mapv2.LaneType_LANE_TYPE_DRIVING {
			log.Panicf("vehicle: vehicle %v try to change to non-driving %v from %v, ac=%+v",
				p.ID(), ac.LCTarget, newRuntime.Lane, ac)
		}
		if newRuntime.LC.IsLC {
			// 正在变道，重置变道状态
			// 情况1: 目标车道是当前车道，什么都不做
			// 情况2: 目标车道是ShadowLane，对CompletedRatio取为1-CompletedRatio
			// 情况3: 目标车道是ShadowLane的另一个相邻车道，则将车辆重置回ShadowLane（撤销变道）
			// 情况4: 目标车道是Lane的另一个相邻车道，则将车辆重置到Lane（完成变道）
			if ac.LCTarget == newRuntime.Lane {
				// 情况1
			} else if ac.LCTarget == newRuntime.LC.ShadowLane {
				// 情况2
				newRuntime.LC.CompletedRatio = 1 - newRuntime.LC.CompletedRatio
				newRuntime.LC.ShadowS, newRuntime.S = newRuntime.S, newRuntime.LC.ShadowS
				newRuntime.Lane, newRuntime.LC.ShadowLane = newRuntime.LC.ShadowLane, newRuntime.Lane
			} else if ac.LCTarget == newRuntime.LC.ShadowLane.LeftLane() || ac.LCTarget == newRuntime.LC.ShadowLane.RightLane() {
				// 情况3
				newRuntime.LC = lcRuntime{
					IsLC:           true,
					ShadowLane:     newRuntime.LC.ShadowLane,
					ShadowS:        newRuntime.LC.ShadowS,
					CompletedRatio: 0,
				}
				log.Debugf("vehicle: 情况3 %v LC %v", p.ID(), newRuntime.LC)
				newRuntime.Lane = ac.LCTarget
				newRuntime.S = ac.LCTarget.ProjectFromLane(newRuntime.LC.ShadowLane, newRuntime.LC.ShadowS)
			} else if ac.LCTarget == newRuntime.Lane.LeftLane() || ac.LCTarget == newRuntime.Lane.RightLane() {
				// 情况4
				newRuntime.LC = lcRuntime{
					IsLC:           true,
					ShadowLane:     newRuntime.Lane,
					CompletedRatio: 0,
				}
				log.Debugf("vehicle: 情况4 %v LC %v", p.ID(), newRuntime.LC)
				newRuntime.Lane = ac.LCTarget
				newRuntime.S = ac.LCTarget.ProjectFromLane(newRuntime.Lane, newRuntime.S)
			} else {
				log.Errorf("vehicle: vehicle %v try to change to non-neighbor %v from %v, ac=%+v, ignore it",
					p.ID(), ac.LCTarget, newRuntime.Lane, ac)
			}
		} else {
			// 发起变道，先将当前的车辆位置映射到目标车道上
			// 发起变道
			//  --------------------------------------------
			//   [2] → → (lane_change_length / ds) → → [3]
			//  --↑-----------------------------------------
			//   [1]     (ignore the width)
			//  --------------------------------------------
			// 1: motion.lane + motion.s
			// 2: target_lane + neighbor_s
			// 3: target_lane + target_s
			newRuntime.LC = lcRuntime{
				IsLC:           true,
				ShadowLane:     newRuntime.Lane,
				CompletedRatio: 0,
			}
			log.Debugf("vehicle: 情况else %v LC %v", p.ID(), newRuntime.LC)
			newRuntime.S = ac.LCTarget.ProjectFromLane(newRuntime.Lane, newRuntime.S)
			newRuntime.Lane = ac.LCTarget
		}
	}
	// 向前更新位置
	skipToEnd = p.driveStraightAndRefreshLocation(&newRuntime, ds, dt)
	if newRuntime.LC.IsLC {
		allWidth := (newRuntime.Lane.Width() + newRuntime.LC.ShadowLane.Width()) / 2
		ratio := newRuntime.LC.CompletedRatio + dw/allWidth
		// 处理变道状态
		if ratio >= 1 {
			// 变道已经完成
			newRuntime.clearLaneChange()
		} else {
			newRuntime.LC.CompletedRatio = ratio
			newRuntime.LC.ShadowS = newRuntime.LC.ShadowLane.ProjectFromLane(newRuntime.Lane, newRuntime.S)
			newRuntime.LC.Yaw = lcYaw
		}
	}

	// 更新xy坐标
	xyz := newRuntime.Lane.GetPositionByS(newRuntime.S)
	if newRuntime.LC.IsLC {
		shadowXYZ := newRuntime.LC.ShadowLane.GetPositionByS(newRuntime.LC.ShadowS)
		xyz = geometry.Blend(shadowXYZ, xyz, newRuntime.LC.CompletedRatio)
	}
	newRuntime.XYZ = xyz

	// 更新runtime
	p.runtime = newRuntime
	// 更新车辆速度
	p.runtime.V = v
	return skipToEnd
}

func (p *Person) driveStraightAndRefreshLocation(rt *runtime, ds float64, dt float64) (skipToEnd bool) {
	s := rt.S
	lane := rt.Lane
	s += ds
	if s > lane.Length() {
		if rt.LC.IsLC {
			log.Debugf("vehicle: vehicle %v skipped the change to lane (LC=%+v)",
				p.ID(), rt.LC)
		}
		rt.clearLaneChange()
		for s > lane.Length() {
			s -= lane.Length()
			lane = p.multiModalRoute.VehicleRoute.Next(lane, p.snapshot.S, p.snapshot.V)
			if lane == nil {
				return true
			}
		}
	}
	rt.Lane = lane
	rt.S = s
	return false
}

// 检查车辆是否到达目标地点，是则返回true
func (p *Person) checkCloseToEndAndRefreshRuntime(skipToEnd bool) bool {
	if skipToEnd || (p.runtime.Lane.ParentRoad() == p.multiModalRoute.VehicleRoute.End.Lane.ParentRoad() && p.multiModalRoute.VehicleRoute.End.S-p.runtime.S <= closeToEnd) {
		// 到达目的地，设置motion为目的地的路面位置（供人进入aoi时选择gate）
		p.runtime.Lane = p.multiModalRoute.VehicleRoute.End.Lane
		p.runtime.S = p.multiModalRoute.VehicleRoute.End.S
		p.runtime.V = 0
		p.runtime.clearLaneChange()
		if skipToEnd {
			log.Debugf("skipToEnd: vehicle %v from %+v to %+v",
				p.ID(), p.snapshot, p.runtime,
			)
		}
		return true
	} else {
		return false
	}
}

// getter

// 获取车辆影子所在的Lane
func (p *Person) ShadowLane() entity.ILane {
	return p.snapshot.LC.ShadowLane
}

// 获取车辆影子在Lane上的位置S坐标
func (p *Person) ShadowS() float64 {
	return p.snapshot.LC.ShadowS
}

// 判断车辆是否正在变道
func (p *Person) IsLC() bool {
	return p.snapshot.LC.IsLC
}

func newVehicleNode(key float64, value entity.IPerson) *entity.VehicleNode {
	return &entity.VehicleNode{
		S:     key,
		Value: value,
	}
}
