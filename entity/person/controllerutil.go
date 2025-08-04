package person

import (
	"math"

	"github.com/tsinghua-fib-lab/agentsociety-sim-oss/entity"
)

// getLaneMaxV 获取车道最大速度
// 功能：根据车道限速和车辆对限速的认知偏差计算实际限速
// 参数：lane-车道对象
// 返回：车辆认为的车道最大速度（米/秒）
// 说明：考虑车辆对限速的认知偏差，模拟不同驾驶员对限速的理解差异
// 算法说明：
// 1. 获取车道的官方限速
// 2. 乘以车辆对限速的认知偏差系数
// 3. 返回车辆认为的实际限速
func (l *controller) getLaneMaxV(lane entity.ILane) float64 {
	return lane.MaxV() * l.laneMaxVRatio
}

// getLCPhi 计算车辆前轮转角
// 功能：根据车速计算变道时的前轮转角
// 参数：v-车速（米/秒）
// 返回：前轮转角（弧度）
// 算法说明：
// 1. 使用线性插值计算转角：φ = K*v + B
// 2. 当v=0km/h时，转角为30度
// 3. 当v=80km/h≈25m/s时，转角为5度
// 4. 最小转角限制为5度
// 5. 将角度转换为弧度
// 说明：车速越快，变道时前轮转角越小，确保变道稳定性
func (l *controller) getLCPhi(v float64) float64 {
	// 车轮最大转角φ: v=0km/h时，为30度, v=80km/h≈25m/s时，为5度
	const K = (5.0 - 25.0) / (25.0 - 0.0)     // 线性插值斜率
	const B = 30.0                            // 线性插值截距
	return math.Max(K*v+B, 5) * math.Pi / 180 // 限制最小转角为5度并转换为弧度
}
