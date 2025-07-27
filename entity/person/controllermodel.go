package person

import (
	"math"

	"git.fiblab.net/general/common/v2/mathutil"
	"github.com/samber/lo"
)

// followImpl 跟车模型核心实现
// 功能：实现智能驾驶模型(IDM)的跟车逻辑
// 参数：selfV-本车速度，targetV-目标速度，aheadV-前车速度，distance-车距，minGap-最小车距，headway-安全车头时距
// 返回：计算得到的加速度（米/秒²）
// 算法说明：
// 1. 检查是否发生碰撞（距离小于等于0）
// 2. 使用IDM模型计算期望车距：s_star = minGap + max(0, v*headway + v*(v-v_ahead)/(2*sqrt(a*b)))
// 3. 计算加速度：a = maxA * (1 - (v/targetV)^4 - (s_star/distance)^2)
// 4. 限制加速度在制动和加速范围内
// 说明：IDM模型是经典的跟车模型，能够模拟真实驾驶行为
func (l *controller) followImpl(
	selfV, targetV, aheadV, distance, minGap, headway float64,
) float64 {
	var acc float64
	if distance <= 0 {
		// 车辆已经发生碰撞，紧急制动
		acc = -mathutil.INF
	} else {
		// https://en.wikipedia.org/wiki/Intelligent_driver_model
		// 计算期望车距：s_star = minGap + max(0, v*headway + v*(v-v_ahead)/(2*sqrt(a*b)))
		s_star := minGap + math.Max(
			0,
			selfV*headway+selfV*(selfV-aheadV)/2/math.Sqrt(-l.usualBrakingA*l.maxA),
		)
		// IDM加速度公式：a = maxA * (1 - (v/targetV)^4 - (s_star/distance)^2)
		acc = l.maxA * (1 - math.Pow(selfV/targetV, idmTheta) - math.Pow(s_star/distance, 2))
	}
	return lo.Clamp(acc, l.maxBrakingA, l.maxA) // 限制加速度在合理范围内
}

// follow 跟车模型
// 功能：使用控制器默认参数调用跟车模型
// 参数：selfV-本车速度，targetV-目标速度，aheadV-前车速度，distance-车距
// 返回：计算得到的加速度（米/秒²）
// 说明：使用控制器中预设的最小车距和安全车头时距参数
func (l *controller) follow(
	selfV, targetV, aheadV, distance float64,
) float64 {
	return l.followImpl(selfV, targetV, aheadV, distance, l.minGap, l.headway)
}

// selfFollow 跟车模型（使用控制器自身的参数）
// 功能：使用控制器自身的速度和目标速度进行跟车计算
// 参数：aheadV-前车速度，distance-车距，laneMaxV-车道最大速度
// 返回：计算得到的加速度（米/秒²）
// 说明：目标速度为车道限速和车辆最大速度的较小值，确保安全
func (l *controller) selfFollow(aheadV, distance, laneMaxV float64) float64 {
	return l.follow(l.v, math.Min(l.maxV, laneMaxV), aheadV, distance)
}

// stop 在指定距离内刹停
// 功能：计算在指定距离内停车所需的加速度
// 参数：distance-停车距离，laneMaxV-车道最大速度，minGap-最小车距
// 返回：计算得到的加速度（米/秒²）
// 算法说明：
// 1. 停车时不需要考虑跟车的安全车头时距
// 2. 使用时间步长作为预判时间
// 3. 前车速度设为0（停车目标）
// 说明：用于计算停车所需的制动加速度，确保在指定距离内安全停车
func (l *controller) stop(distance, laneMaxV, minGap float64) float64 {
	// 停车的话，要先预判dt时间，而不需要按照跟车的headway进行计算
	return l.followImpl(l.v, math.Min(l.maxV, laneMaxV), 0, distance, minGap, l.dt)
}
