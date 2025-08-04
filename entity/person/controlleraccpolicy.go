package person

import (
	"git.fiblab.net/general/common/v2/mathutil"
	mapv2 "git.fiblab.net/sim/protos/v2/go/city/map/v2"
	"github.com/tsinghua-fib-lab/agentsociety-sim-oss/entity"
)

// policyCarFollow 策略1：前车跟车策略
// 功能：根据前车信息计算跟车加速度
// 参数：curLane-当前车道，ahead-前车节点，distance-与前车距离
// 返回：ac-计算得到的加速度动作
// 算法说明：
// 1. 获取前车速度：如果前车存在则获取其速度，否则为0
// 2. 调用跟车模型：使用IDM模型计算跟车加速度
// 3. 考虑车道限速：使用当前车道的最大速度限制
// 说明：这是最基本的跟车策略，基于智能驾驶模型(IDM)实现
func (l *controller) policyCarFollow(
	curLane entity.ILane,
	ahead *entity.VehicleNode, distance float64,
) (ac Action) {
	var aheadV float64
	if ahead != nil {
		aheadV = ahead.V()
	}
	ac.A = l.selfFollow(aheadV, distance, l.getLaneMaxV(curLane))
	return
}

// policyLane 策略2：车道相关策略
// 功能：处理车道相关的各种约束和情况
// 参数：curLane-当前车道，aheadLanes-前方车道环境，s-当前位置
// 返回：ac-计算得到的加速度动作
// 算法说明：
// 1. 初始化加速度为无穷大（表示无约束）
// 2. 红灯停车检查：如果未完全进入车道且遇到红灯则停车
// 3. 路口人行道处理：检查人行道占用情况，决定停车或减速
// 4. 前方车道检查：检查前方车道的各种限制条件
// 5. 信号灯处理：根据信号灯状态决定是否停车
// 说明：处理车道上的各种交通规则和约束条件
func (l *controller) policyLane(curLane entity.ILane, aheadLanes []envLane, s float64) (ac Action) {
	ac.A = mathutil.INF

	// 下一车道
	if len(aheadLanes) == 0 {
		return
	}
	for _, envLane := range aheadLanes {
		// 假设要在路口停车，加速度是多少
		// ATTENTION: 增加2米的空间
		stopA := l.stop(envLane.distance, l.getLaneMaxV(curLane), l.minGap+2)
		if envLane.lane.InJunction() {
			// 需要开始判断路口信控情况
			switch state, _, remainingTime := envLane.lane.Light(); state {
			case mapv2.LightState_LIGHT_STATE_RED:
				// 红灯减速停车
				ac.Update(Action{
					A: stopA,
				})
			case mapv2.LightState_LIGHT_STATE_YELLOW:
				// 黄灯，倒计时结束前不可过线，减速停车
				if remainingTime*l.v <= envLane.distance {
					ac.Update(Action{
						A: stopA,
					})
				}
			default:
				// 绿灯或没灯，跳过
			}
		}
	}
	return
}
