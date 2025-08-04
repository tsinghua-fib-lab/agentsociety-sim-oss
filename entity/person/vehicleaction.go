package person

import (
	"github.com/tsinghua-fib-lab/agentsociety-sim-oss/entity"
)

// Action 车辆动作结构体
// 功能：描述车辆的控制动作，包括加速度、变道目标等
type Action struct {
	A        float64      // 加速度（米/秒²）
	LCTarget entity.ILane // 变道目标车道
	LCPhi    float64      // 变道过程的前轮角度（弧度）

	AheadVDistance float64 // 到前方车辆的距离（米）
}

// Update 更新车辆动作
// 功能：采用取最小的方式设置加速度，处理多个动作的冲突
// 参数：others-其他动作列表
// 算法说明：
// 1. 对于加速度，取所有动作中的最小值（最保守的制动）
// 2. 对于变道目标，如果存在冲突则记录错误
// 3. 优先使用第一个有效的变道目标
func (a *Action) Update(others ...Action) {
	for _, o := range others {
		if o.A < a.A {
			a.A = o.A
		}
		if o.LCTarget != nil {
			if a.LCTarget != nil {
				log.Error("start lane change conflict")
			}
			a.LCTarget = o.LCTarget
			a.LCPhi = o.LCPhi
		}
	}
}

// SetBrakeAcc 设置制动加速度
// 功能：根据制动距离和当前速度计算所需的制动加速度
// 参数：brakeDistance-制动距离（米），v-当前速度（米/秒）
// 算法说明：
// 1. 检查车辆是否静止（静止车辆无法制动）
// 2. 检查制动距离是否有效
// 3. 使用运动学公式计算制动加速度：a = -v²/(2*d)
func (a *Action) SetBrakeAcc(brakeDistance, v float64) {
	if v == 0 {
		log.Error("unmoving vehicle cannot brake")
	} else if brakeDistance <= 0 {
		// log.Error("brake target already pass")
	} else {
		a.A = -v * v / brakeDistance / 2
	}
}

// startLaneChange 开始变道
// 功能：设置变道目标车道和变道角度
// 参数：lcTarget-变道目标车道，lcPhi-变道角度（弧度）
// 说明：用于初始化变道动作的参数
func (a *Action) startLaneChange(lcTarget entity.ILane, lcPhi float64) {
	a.LCTarget = lcTarget
	a.LCPhi = lcPhi
}
