package person

import (
	"fmt"
	"math"

	"git.fiblab.net/general/common/v2/mathutil"
	"github.com/tsinghua-fib-lab/moss-agentsociety-go/entity"
	"github.com/tsinghua-fib-lab/moss-agentsociety-go/entity/person/route"
	"github.com/tsinghua-fib-lab/moss-agentsociety-go/utils/randengine"

	"github.com/samber/lo"
)

const (
	idmTheta           = 4   // IDM模型参数（智能驾驶模型参数）
	platoonMaxDistance = 10  // 编队判定距离（间距小于该值表示完成编队，形成编队的后车将无视信控与车道限速）
	laneMaxVBiasStd    = 0.1 // 车道限速偏差比例的标准差

	// https://jtgl.beijing.gov.cn/jgj/94220/aqcs/139634/index.html
	viewDistanceFactor   = 12 // 在一般情况下，观察距离应等于汽车在12秒内所通过的路程。如果车速为每小时60公里，则观察距离应为200米。
	minViewDistance      = 50 // 最小观察距离（米）
	behindViewDistance   = 3  // 后方观察距离（米）
	decelerationDuration = 20 // 停车提前开始的时间（秒）

	// maxNoiseA 加速度随机扰动最大值
	// 功能：为车辆加速度添加随机扰动，模拟真实驾驶的不确定性
	maxNoiseA = .5

	// zeroAThreshold 加速度零值判定阈值
	// 功能：当加速度绝对值小于此值时认为加速度为零
	zeroAThreshold = .1
)

// controller 车辆控制器
// 功能：管理车辆的所有控制逻辑，包括跟车、变道、速度控制等
type controller struct {
	// 控制器保持的参数

	self          *Person            // 模块所在车辆
	usualBrakingA float64            // 常用制动加速度
	maxBrakingA   float64            // 最大制动加速度
	maxA          float64            // 最大加速度
	maxV          float64            // 最大速度
	laneMaxVRatio float64            // 本车对车道限速认知的偏差百分比，正态分布N(1,0.1)，例如车道限速为50，偏差为10%，则本车认为车道限速为55，限制不超过20%
	length        float64            // 车辆长度
	minGap        float64            // 最小车距
	lcLength      float64            // 变道长度
	headway       float64            // 安全车头时距
	generator     *randengine.Engine // 随机数生成器

	// 状态

	forceLC    bool    // 强制变道标志
	lastLCTime float64 // 上次变道时间

	// 每次update时更新

	route *route.VehicleRoute // 当前路由
	node  *entity.VehicleNode // 当前节点
	v     float64             // 当前速度
	dt    float64             // 时间步长
}

// newController 创建新的车辆控制器
// 功能：根据车辆属性初始化控制器，设置各种控制参数
// 参数：self-车辆实体
// 返回：初始化完成的控制器实例
// 算法说明：
// 1. 验证和修正车辆属性参数
// 2. 从分布中采样缺失的参数
// 3. 设置控制器的所有参数
// 4. 初始化状态变量
func newController(self *Person) *controller {
	// 数据预读
	vehicleAttr := self.vehicleAttr
	e := self.generator
	c := &controller{
		self:          self,
		usualBrakingA: vehicleAttr.UsualBrakingAcceleration,
		maxBrakingA:   vehicleAttr.MaxBrakingAcceleration,
		maxA:          vehicleAttr.MaxAcceleration,
		maxV:          vehicleAttr.MaxSpeed,
		laneMaxVRatio: vehicleAttr.LaneMaxSpeedRecognitionDeviation,
		length:        vehicleAttr.Length,
		minGap:        vehicleAttr.MinGap,
		lcLength:      vehicleAttr.LaneChangeLength,
		headway:       vehicleAttr.Headway,
		generator:     e,
		lastLCTime:    -mathutil.INF,
	}
	return c
}

// envType 环境类型枚举
// 功能：表示车辆所处的不同环境类型
type envType int

const (
	curEnv    envType = iota // 当前环境
	shadowEnv                // 影子环境（变道时）
	leftEnv                  // 左侧环境
	rightEnv                 // 右侧环境
)

// envVehicle 环境中的车辆信息
// 功能：记录环境中其他车辆的信息
type envVehicle struct {
	node     *entity.VehicleNode // 车辆节点
	distance float64             // 距离（米）
}

// envLane 环境中的车道信息
// 功能：记录环境中车道的信息
type envLane struct {
	lane     entity.ILane // 车道
	distance float64      // 距离（米）
}

// env 环境信息结构
// 功能：描述车辆当前所处的完整环境信息
type env struct {
	typ              envType      // 环境类型
	curLane          entity.ILane // 当前车道
	s                float64      // 在车道上的位置
	aheadLanes       []envLane    // 前方车道列表
	aheadVeh         *envVehicle  // 前方车辆
	nextStopDistance float64      // 到下一个停车点的距离
}

// String 生成环境信息的字符串表示
// 功能：将环境信息转换为可读的字符串格式，用于调试
// 返回：包含环境信息的字符串
func (e env) String() string {
	return fmt.Sprintf(
		"curLane=%v, s=%v, aheadLanes=%v, aheadVeh=%v",
		e.curLane, e.s, e.aheadLanes, e.aheadVeh,
	)
}

// getEnv 获取环境信息
// 功能：根据当前车辆位置和提示信息构建完整的环境描述
// 参数：aheadHint-前方车辆提示，curLane-当前车道，s-位置
// 返回：环境信息结构
// 说明：这是环境感知的核心函数，为后续决策提供基础数据
func (l *controller) getEnv(
	aheadHint *entity.VehicleNode,
	curLane entity.ILane,
	s float64,
) (e env) {
	viewDistance := math.Max(l.v*viewDistanceFactor, minViewDistance)
	e.curLane = curLane
	e.s = s
	e.nextStopDistance = math.Inf(0)
	scanDistance := curLane.Length() - e.s // 已经向前探测的距离
	juncIndex := 0
	// ---------------------------------------------
	for scanDistance < viewDistance {
		if curLane.InJunction() {
			var err error
			if curLane, err = curLane.UniqueSuccessor(); err != nil {
				log.Panicf("controller.getEnv: %v", err)
			}
			juncIndex++
		} else {
			// 从route的LaneGroup中找
			curLane, _ = l.route.GetJunctionLaneByPreLane(curLane, juncIndex)
		}
		if curLane == nil {
			break
		}
		e.aheadLanes = append(e.aheadLanes, envLane{
			lane:     curLane,
			distance: scanDistance,
		})
		scanDistance += curLane.Length()
	}
	// ---------------------------------------------
	// 感知前车
	if aheadHint != nil {
		e.aheadVeh = &envVehicle{
			node:     aheadHint,
			distance: aheadHint.S - e.s - aheadHint.L(),
		}
	}
	// 感知障碍物
	if e.aheadVeh == nil {
		// 检查前方车道
		for _, envLane := range e.aheadLanes {
			aheadHint = envLane.lane.FirstVehicle()
			if aheadHint != nil {
				e.aheadVeh = &envVehicle{
					node:     aheadHint,
					distance: envLane.distance + aheadHint.S - aheadHint.L(),
				}
			}
			if e.aheadVeh != nil {
				break
			}
		}
	}
	return
}

func (l *controller) getSideEnvs(
	curLane entity.ILane,
	s float64,
) [2]*env {
	envs := [2]*env{}
	var sideSs [2]float64
	links := l.node.Extra.Links
	for _, side := range []int{entity.LEFT, entity.RIGHT} {
		lane := curLane.NeighborLane(side)
		if lane == nil {
			continue
		}
		sideSs[side] = lane.ProjectFromLane(curLane, s)
		ahead := links[side][entity.AFTER]
		e := l.getEnv(ahead, lane, sideSs[side])
		envs[side] = &e
	}
	if envs[entity.LEFT] != nil {
		envs[entity.LEFT].typ = leftEnv
	}
	if envs[entity.RIGHT] != nil {
		envs[entity.RIGHT].typ = rightEnv
	}
	return envs
}

func (l *controller) update(dt float64) (ac Action) {
	ac.A = mathutil.INF
	ac.AheadVDistance = -1
	// 更新参数
	l.route = l.self.multiModalRoute.VehicleRoute
	l.node = l.self.vehicle.node
	l.v = l.self.runtime.V
	l.dt = dt

	var (
		e        env
		sideEnvs [2]*env
		shadowE  *env
	)

	updateEnvs := func() {
		e = l.getEnv(
			l.self.vehicle.node.Next(),
			l.self.runtime.Lane, l.self.runtime.S,
		)
		e.typ = curEnv
		sideEnvs = l.getSideEnvs(l.self.runtime.Lane, l.self.runtime.S)
		if l.self.IsLC() {
			shadowE = &env{}
			log.Debugf("person: %v, LC: %v", l.self.id, l.self.runtime.LC)
			*shadowE = l.getEnv(
				l.self.vehicle.shadowNode.Next(),
				l.self.runtime.LC.ShadowLane, l.self.runtime.LC.ShadowS,
			)
			shadowE.typ = shadowEnv
		}
		// 前车距离（微观统计数据）
		if e.aheadVeh != nil {
			ac.AheadVDistance = e.aheadVeh.distance
		}
	}

	updateEnvs()

	// ---------------------------------------------
	l.headway = l.self.vehicleAttr.Headway

	// 执行纵向决策（加速度）
	if e.aheadVeh != nil {
		ac.Update(l.policyCarFollow(e.curLane, e.aheadVeh.node, e.aheadVeh.distance))
	} else {
		ac.Update(l.policyCarFollow(e.curLane, nil, mathutil.INF))
	}
	ac.Update(l.policyLane(e.curLane, e.aheadLanes, e.s))
	// 执行变道时的额外纵向决策（加速度），看原车道的前车
	if l.self.IsLC() {
		if shadowE.aheadVeh != nil {
			ac.Update(l.policyCarFollow(shadowE.curLane, shadowE.aheadVeh.node, shadowE.aheadVeh.distance))
		}
		ac.Update(l.policyLane(shadowE.curLane, shadowE.aheadLanes, shadowE.s))
	}
	// 执行横向决策（变道）
	if !l.self.IsLC() && !e.curLane.InJunction() {
		ac.Update(l.planLaneChange(e.curLane, e.s, e.aheadVeh, sideEnvs))
	}
	// 执行变道角度控制
	if l.self.IsLC() {
		ac.LCPhi = l.getLCPhi(l.v)
	}

	// 后处理
	ac.A = lo.Clamp(ac.A, l.maxBrakingA, l.maxA)
	// 加速度添加随机扰动
	noise_acc := maxNoiseA * lo.Clamp(.5*l.generator.NormFloat64(), -1, 1)
	// 过小的加速度不扰动 扰动不改变加速度符号
	if math.Abs(ac.A) >= zeroAThreshold && math.Signbit(ac.A) == math.Signbit(ac.A+noise_acc) {
		ac.A += noise_acc
	}
	return ac
}
