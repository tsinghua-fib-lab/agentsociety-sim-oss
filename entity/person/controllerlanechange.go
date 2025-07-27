package person

import (
	"math"

	"git.fiblab.net/general/common/v2/mathutil"
	"git.fiblab.net/sim/simulet-go/entity"
)

const (
	lcLengthFactor     = 5   // 变道长度与当前车速的关系（即几秒完成变道）
	lcInOldLaneRatio   = 0.5 // 变道完成度小于该值时，认为还在原车道
	lcSafeBrakingABias = 1
	lcLaneEnd          = 20 // 车道最末端禁止主动变道的距离
)

// planLaneChange 变道规划主函数
// 功能：根据当前环境和策略决定是否进行变道
// 参数：curLane-当前车道，s-当前位置，ahead-前方车辆，sideEnvs-侧方环境，enableProactiveLaneChange-是否启用主动变道
// 返回：ac-变道动作
// 算法说明：
// 1. 强制变道检查：如果距离目标车道过远，进入强制变道模式
// 2. 走错路处理：如果剩余距离不足，重新规划路径
// 3. 主动变道决策：根据MOBIL或SUMO算法决定是否变道
// 4. 变道执行：执行具体的变道动作
// 说明：这是变道决策的核心函数，处理各种变道场景
func (l *controller) planLaneChange(
	curLane entity.ILane, s float64, ahead *envVehicle,
	sideEnvs [2]*env,
) (ac Action) {
	ac.A = mathutil.INF
	reverseS := curLane.Length() - s
	links := l.node.Extra.Links
	envs := sideEnvs
	maxV := l.getLaneMaxV(curLane)
	// 变道目标
	lcLength := math.Max(l.v*lcLengthFactor, l.length) // 变道距离至少保留2个车长
	lc := l.route.GetLCScan(curLane, l.self.snapshot.S, l.self.snapshot.V)
	if !lc.InCandidate && (reverseS-lc.DeltaLCDistance <= lcLength*float64(lc.Count)) {
		// 如果距离不足，进入强制变道模式（且无法从路由上延迟变道）
		l.forceLC = true
	} else if lc.InCandidate && l.forceLC {
		// 已经在目标车道组内，退出强制变道模式
		l.forceLC = false
	}
	if l.forceLC {
		e := envs[lc.Side]
		// 加塞的处理逻辑
		if e == nil {
			log.Panicf("VehicleRoute: bad force lc target %+v, %v, %+v", lc, curLane, l.route)
		}
		target := e.curLane
		l.lastLCTime = l.self.ctx.Clock().T
		// 执行纵向控制策略
		sn := e.s
		if e.aheadVeh != nil {
			ac.Update(l.policyCarFollow(e.curLane, e.aheadVeh.node, e.aheadVeh.distance))
		}
		ac.Update(l.policyLane(e.curLane, e.aheadLanes, e.s))
		// 变道中考虑后车，在强制变道中采用尽可能减速的方式进行后车处理
		// 强制变道，必须过去，所以越慢越好
		if back := links[lc.Side][entity.BEFORE]; back != nil {
			v3 := back.V()
			s3 := back.S
			an3 := l.follow(v3, maxV, l.v, sn-l.length-s3)
			// 判决规则: 如果后车会追尾本车，本车刹车停下来等后车过去
			// TODO: 不太合理
			if an3 < math.Min(l.usualBrakingA+lcSafeBrakingABias, -1) {
				ac.Update(Action{A: l.maxBrakingA})
				// 变道，但不旋转车身
				ac.startLaneChange(target, 0)
				return
			}
		}
		// 正常强制变道，减速慢行
		if ac.LCTarget == nil {
			ac.Update(Action{A: l.usualBrakingA})
			ac.startLaneChange(target, 0)
		}
		return
	}

	// 前方车道距离过近
	if reverseS < lcLaneEnd {
		return
	}
	// 距离上次变道时间过短
	if l.self.ctx.Clock().T-l.lastLCTime < l.generator.Float64()*2+4 {
		return
	}
	// 没有变道的可能
	if envs[entity.LEFT] == nil && envs[entity.RIGHT] == nil {
		return
	}
	// MOBIL变道算法
	// -----------------------
	//      [3]   [n0] [4]  现在假设0->n0的变道(n = next)
	// -----------------------
	//  [2]      [0]    [1]
	// -----------------------
	//     []   	 []
	// -----------------------
	// 要求变道后：
	// 1. [3]不会追尾[target]，即[3]的预期加速度（刹车）不能小于安全加速度（取max和一般值的平均）
	// 2. 整体加速度提升大于阈值: \Delta_a0 + p(\Delta_a2+\Delta_a3) > a_threshold
	// threshold根据变道车道是否在导航路线上有所不同

	// 对于其他车的属性，采用本车的值去推断
	v1, s1 := mathutil.INF, mathutil.INF
	if ahead != nil && ahead.node != nil {
		v1 = ahead.node.V()
		s1 = ahead.node.S - ahead.node.L()
	}
	a0 := l.selfFollow(v1, s1-s, maxV)
	deltaA2 := 0.0
	if vehNode2 := l.node.Prev(); vehNode2 != nil {
		// 如果2号车存在，计算2号车的预期加速度变化值
		v2 := vehNode2.V()
		s2 := vehNode2.S
		deltaA2 = l.follow(v2, maxV, v1, s1-s2) - l.follow(v2, maxV, l.v, s-l.length-s2)
	}
	deltas := [2]float64{}
	an0s := [2]float64{}
	for _, side := range [2]int{entity.LEFT, entity.RIGHT} {
		e := envs[side]
		if e == nil {
			continue
		}
		target := e.curLane
		if target == nil {
			// 无法变道
			continue
		}
		if lc.InCandidate {
			// 如果已经在目标车道组内，但要变道到目标车道组外，不允许
			if lc.Neighbors[side] == 0 {
				continue
			}
		} else {
			// 如果不在目标车道组内，变道后距离目标车道集合更远，不允许
			if side != lc.Side {
				continue
			}
		}
		// 本车变道后的预期加速度
		v4, s4 := mathutil.INF, mathutil.INF
		if node := links[side][entity.AFTER]; node != nil {
			v4 = node.V()
			s4 = node.S - node.L()
		}
		sn0 := e.s
		an0 := l.selfFollow(v4, s4-sn0, maxV)
		an0s[side] = an0
		deltaA0 := an0 - a0
		// 3号车变道后的预期加速度
		deltaA3 := 0.0
		if vehNode3 := links[side][entity.BEFORE]; vehNode3 != nil {
			v3 := vehNode3.V()
			s3 := vehNode3.S
			an3 := l.follow(v3, maxV, l.v, sn0-l.length-s3)
			// 判决规则1: 如果3号车会追尾target，那么不变道
			if an3 < l.usualBrakingA+lcSafeBrakingABias {
				continue
			}
			deltaA3 = an3 - l.follow(v3, maxV, v4, s4-s3)
		}
		// 主判决规则
		// 参考封硕Nature子刊的处理方式
		if delta := deltaA0 + 0.1*(deltaA2+deltaA3); delta > 0 {
			deltas[side] = delta
		}
	}
	u := deltas[entity.LEFT] + deltas[entity.RIGHT]
	pLC := 2e-8
	if u >= 1 {
		pLC = 0.9
	} else if u > 0 {
		pLC = (0.9 - 2e-8) * u
	} else {
		// u <= 0, 意味着deltas[entity.LEFT] = deltas[entity.RIGHT] = 0
		// 为了保证0值体现车道不存在，进行修正
		if envs[entity.LEFT] != nil {
			deltas[entity.LEFT] = 1
		}
		if envs[entity.RIGHT] != nil {
			deltas[entity.RIGHT] = 1
		}
	}
	// 按概率决定是否变道
	if l.generator.PTrue(pLC) {
		// 再按照deltas的大小来按概率决定变道方向
		side := int(l.generator.DiscreteDistribution(deltas[:]))
		e := envs[side]
		// 执行变道逻辑
		target := e.curLane
		ac = Action{A: an0s[side]}
		ac.Update(l.policyLane(e.curLane, e.aheadLanes, e.s))
		l.lastLCTime = l.self.ctx.Clock().T
		ac.startLaneChange(target, 0)
	}
	return
}
