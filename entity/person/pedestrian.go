package person

import (
	"github.com/samber/lo"
	"github.com/tsinghua-fib-lab/moss-agentsociety-go/entity"
)

const (
	defaultWalkV   = 1.34 // 默认步行速度（米/秒）
	minWalkV       = 0.5  // 最小步行速度（米/秒）
	defaultBikeV   = 4.0  // 默认骑行速度（米/秒）
	minBikeV       = 1.0  // 最小骑行速度（米/秒）
	maxVNoise      = .5   // 速度随机扰动最大值（米/秒）
	shouldNextBias = 1    // 在实际更新位置时相对于orca计算值的增加量
)

// pedestrian 行人实体数据结构
// 功能：管理行人的所有属性和状态，包括速度、位置偏移、链表节点等
type pedestrian struct {
	walkingV           float64 // 行走速度（米/秒）
	bikingV            float64 // 骑行速度（米/秒）
	verticalOffsetRate float64 // 垂直偏移偏好（百分比）
	horizontalOffset   float64 // 水平偏移（米）

	// Lane链表
	node *entity.PedestrianNode // 行人在车道链表中的节点
}

// updatePedestrian 更新行人状态
// 功能：执行行人的主要更新逻辑，包括ORCA计算、位置更新、距离统计等
// 参数：dt-时间步长，pedControlChan-行人控制通道
// 返回：isEnd-是否到达终点
// 算法说明：
// 1. 获取当前路段信息
// 2. 根据是否有外部控制决定速度计算方式
// 3. 更新位置和距离统计
// 4. 处理人行道切换
// 5. 检查是否到达终点
func (p *Person) updatePedestrian(dt float64) (isEnd bool) {
	lane := p.runtime.Lane
	seg := p.multiModalRoute.PedestrianRoute.Current()

	s := p.S()
	v := p.pedestrian.walkingV
	if lane.IsNoEntry() {
		v *= 2 // 红灯，赶快走
	}
	ds := v * dt

	// 将所有新增量加到s上
	if seg.IsForward() {
		s += ds
	} else {
		s -= ds
	}
	// 循环，更新s，修改人的位置，直到人不超出当前车道
	for {
		// 计算多出来的部分（总是为正值）
		shouldNext := s < 0 || s > lane.Length()
		if !shouldNext {
			break
		}
		// 先检查进入下一个segment的话，下一个是否是禁止通行的车道，如果是，则不进去下一个segment
		if !p.multiModalRoute.PedestrianRoute.AtLast() {
			if p.multiModalRoute.PedestrianRoute.Next().Lane.IsNoEntry() {
				p.runtime.V = 0
				return
			}
		}
		// 导航进入下一个segment
		if ok := p.multiModalRoute.PedestrianRoute.Step(); ok {
			// 先计算上一段超出的部分
			if s < 0 {
				s = -s
			} else if s > lane.Length() {
				s -= lane.Length()
			}
			// 更新segment和lane
			seg = p.multiModalRoute.PedestrianRoute.Current()
			lane = seg.Lane
			// 如果是反向，s从另一头计算
			if seg.IsForward() {
				// do nothing
			} else {
				s = lane.Length() - s
			}
		} else {
			isEnd = true // 路径已经走完，标记为结束（相对异常的情况）
			break
		}
	}
	// 如果在最后一个路段，且s超出了终点，标记为结束
	if p.multiModalRoute.PedestrianRoute.AtLast() {
		if seg.IsForward() {
			isEnd = s >= p.multiModalRoute.PedestrianRoute.End.S
		} else {
			isEnd = s <= p.multiModalRoute.PedestrianRoute.End.S
		}
	}
	// 对s坐标进行范围限制
	s = lo.Clamp(s, 0, lane.Length())
	// 如果到达终点，设置为终点位置
	if isEnd {
		p.runtime.Lane = p.multiModalRoute.PedestrianRoute.Last().Lane
		p.runtime.S = p.multiModalRoute.PedestrianRoute.End.S
		// 增量更新车道索引（不再维护数据）
		p.snapshot.Lane.RemovePedestrian(p.pedestrian.node)
		return
	}

	// 检测是否发生和车辆的碰撞，如果发生则撤销这次移动
	xyz := seg.Lane.GetPositionByS(s)

	p.runtime.IsForward = seg.IsForward()
	p.runtime.Lane = seg.Lane
	p.runtime.S = s
	p.runtime.XYZ = xyz
	p.runtime.V = v

	// 增量更新车道索引（维护数据）
	if p.snapshot.Lane != p.runtime.Lane {
		p.snapshot.Lane.RemovePedestrian(p.pedestrian.node)
		// 换一个新的node来避免remove操作和add操作处理同一个对象需要保证先后顺序
		p.pedestrian.node = newPedestrianNode(p.runtime.S, p)
		p.runtime.Lane.AddPedestrian(p.pedestrian.node)
	}
	// 更新统计
	p.m.recordRunning(dt, ds)
	return
}

func newPedestrianNode(key float64, value entity.IPerson) *entity.PedestrianNode {
	return &entity.PedestrianNode{
		S:     key,
		Value: value,
	}
}

func (p *Person) IsForward() bool {
	return p.snapshot.IsForward
}
