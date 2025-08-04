// 提供Max Pressure信号灯控制算法
// 不会按照原来的相位顺序切换，而是在每个相位结束后计算所有相位的pressure，选取pressure最大的相位
package trafficlight

import (
	"errors"
	"flag"

	"git.fiblab.net/general/common/v2/mathutil"
	mapv2 "git.fiblab.net/sim/protos/v2/go/city/map/v2"
	"github.com/samber/lo"
	"github.com/tsinghua-fib-lab/agentsociety-sim-oss/entity"
	"github.com/tsinghua-fib-lab/agentsociety-sim-oss/utils/container"
)

var (
	yellowTime          = flag.Float64("tl.mp_yellow_time", 3, "最大压力法黄灯时间")
	pedestrianClearTime = flag.Float64("tl.mp_pedestrian_clear_time", 5, "最大压力法行人清空时间")
	allRedTime          = flag.Float64("tl.mp_all_red_time", 3, "最大压力法全红时间")
	phaseTime           = flag.Float64("tl.mp_phase_time", 15, "最大压力法相位时间")
	maxRepeatCount      = flag.Int("tl.mp_max_repeat_count", 6, "最大压力法每个相位最多重复的次数")
)

var (
	ErrMaxPressure = errors.New("mp: cannot set traffic light with traffic light algorithm")
)

// mpTlRuntime 最大压力信号灯运行时数据结构
// 功能：存储最大压力算法的运行时状态，包括相位信息、时间控制、过渡状态等
type mpTlRuntime struct {
	phases           [][]mapv2.LightState // 可供最大压力算法选择的相位列表（如果nil，则没有信控）
	index            int                  // 当前相位
	repeatCount      int                  // 当前相位重复的次数
	totalTime        float64              // 当前相位总时长
	remainingT       float64              // 当前相位剩余时间
	transitionPhases [][]mapv2.LightState // 过渡相位 包含行人清空、黄灯和全红等相位
	transitionTimes  []float64            // 过渡相位持续时长

	nextIndex int // 黄灯状态后的下一个相位
}

// mpTrafficLight 最大压力信号灯控制器
// 功能：实现基于最大压力算法的自适应信号灯控制，根据车道压力动态选择最优相位
type mpTrafficLight struct {
	junctionID         int32                            // 所属junction ID
	lanes              []entity.ILaneTrafficLightSetter // 车道数据
	snapshotRemainingT float64                          // 上一次的剩余时间
	runtime            mpTlRuntime                      // 运行时数据
	ok                 bool                             // 信号灯状态，true为开启，false为关闭
	okBuffer           bool                             // 信号灯状态buffer，用于交互式接口写入
}

// NewMaxPressureTrafficLight 创建Max Pressure算法信号灯控制器
// 功能：初始化最大压力信号灯控制器，设置基础参数和可用相位
// 参数：junctionID-路口ID，lanes-车道列表，phases-可用相位列表
// 返回：初始化完成的最大压力信号灯控制器实例
func NewMaxPressureTrafficLight(junctionID int32, lanes []entity.ILaneTrafficLightSetter, phases [][]mapv2.LightState) *mpTrafficLight {
	return &mpTrafficLight{
		junctionID: junctionID,
		lanes:      lanes,
		runtime:    mpTlRuntime{phases: phases},
		ok:         true,
		okBuffer:   true,
	}
}

// Prepare 准备阶段，处理信号灯的准备工作
// 功能：更新信号灯状态，将当前相位信息写入车道，处理全绿灯和过渡相位情况
// 说明：至少需要两个相位才有信控，否则保持全绿灯状态
func (l *mpTrafficLight) Prepare() {
	// 更新信号灯状态
	l.ok = l.okBuffer
	l.snapshotRemainingT = l.runtime.remainingT
	// 写入lane中数据
	// 至少两个相位才有信控
	if len(l.runtime.phases) < 2 || !l.ok {
		// 无信控，全绿
		for _, lane := range l.lanes {
			lane.SetLight(mapv2.LightState_LIGHT_STATE_GREEN, mathutil.INF, mathutil.INF)
		}
	} else {
		// 设置相位与时间
		if len(l.runtime.transitionPhases) > 0 {
			phase := l.runtime.transitionPhases[0]
			nextPhase := l.runtime.phases[l.runtime.nextIndex]
			// 过渡相位
			if len(l.runtime.transitionPhases) > 1 {
				nextPhase = l.runtime.transitionPhases[1]
			}
			for i, lane := range l.lanes {
				// 如果下个相位还是绿灯，则把下个相位的时间也加上
				if phase[i] == mapv2.LightState_LIGHT_STATE_GREEN && nextPhase[i] == mapv2.LightState_LIGHT_STATE_GREEN {
					lane.SetLight(phase[i], l.runtime.totalTime+*phaseTime, l.runtime.remainingT+*phaseTime)
				} else {
					lane.SetLight(phase[i], l.runtime.totalTime, l.runtime.remainingT)
				}
			}
		} else {
			phase := l.runtime.phases[l.runtime.index]
			for i, lane := range l.lanes {
				lane.SetLight(phase[i], l.runtime.totalTime, l.runtime.remainingT)
			}
		}
	}
}

// Update 更新阶段，执行最大压力算法的核心逻辑
// 功能：根据车道压力动态选择最优相位，处理相位切换和过渡状态
// 参数：dt-时间步长
// 算法说明：
// 1. 计算所有车道的压力值
// 2. 为每个相位计算总压力（绿灯车道压力之和）
// 3. 选择压力最大的相位作为下一个相位
// 4. 如果最大压力相位未变化且未达到最大重复次数，则延长当前相位
// 5. 生成过渡相位（行人清空、黄灯、全红）
func (l *mpTrafficLight) Update(dt float64) {
	if len(l.runtime.phases) < 2 || !l.ok {
		return
	}

	l.runtime.remainingT -= dt
	if l.runtime.remainingT > 0 {
		// 当前相位没走完，啥事都不干
		return
	}
	if len(l.runtime.transitionPhases) == 1 {
		// 切换相位（过渡相位->下一相位）进入下一相位
		l.runtime.index = l.runtime.nextIndex
		l.runtime.remainingT += *phaseTime
		l.runtime.transitionPhases = nil
	} else if len(l.runtime.transitionPhases) > 1 {
		// 切换相位（过渡相位->下一个过渡相位）
		l.runtime.transitionTimes = l.runtime.transitionTimes[1:]
		l.runtime.transitionPhases = l.runtime.transitionPhases[1:]
		l.runtime.remainingT += l.runtime.transitionTimes[0]
	} else {
		// 切换相位（正常灯->根据最大压力计算下一相位并生成黄灯相位）
		// 找到最大压力的相位
		lanePressure := lo.Map(l.lanes, func(l entity.ILaneTrafficLightSetter, _ int) float64 {
			return l.GetPressure()
		})
		pressureHeap := container.NewPriorityQueue[int]()
		for i, phase := range l.runtime.phases {
			// 统计所有绿灯junction lane的压力和
			pressure := 0.
			for j, state := range phase {
				if state == mapv2.LightState_LIGHT_STATE_GREEN {
					pressure += lanePressure[j]
				}
			}
			pressureHeap.Push(i, -pressure) // 小顶堆，压力越大越靠前
		}
		pressureHeap.Heapify()
		// 如果最大压力的相位没有变化，延时直至达到最长时间（并切换到第二大压力的相位）
		// 如果有变化，进入黄灯状态
		maxIndex, _ := pressureHeap.HeapPop()
		if maxIndex == l.runtime.index {
			// 没变化，先检查是否达到最大延时次数
			if l.runtime.repeatCount >= *maxRepeatCount {
				// 达到最大延时次数，切换到第二大压力的相位
				maxIndex, _ = pressureHeap.HeapPop()
			} else {
				l.runtime.remainingT += *phaseTime
				l.runtime.repeatCount++
			}
		}
		if maxIndex != l.runtime.index {
			// 有变化
			l.runtime.nextIndex = maxIndex
			l.runtime.repeatCount = 1
			// 行人清空相位
			clearPhase := make([]mapv2.LightState, len(l.lanes))
			// 黄灯相位，把当前为绿灯、下一时刻为红灯的变为黄灯
			yellowPhase := make([]mapv2.LightState, len(l.lanes))
			hasClearPhase := false
			// 全红相位
			allRedPhase := make([]mapv2.LightState, len(l.lanes))
			hasAllRedPhase := false
			nextPhase := l.runtime.phases[maxIndex]
			copy(yellowPhase, l.runtime.phases[l.runtime.index])
			copy(clearPhase, l.runtime.phases[l.runtime.index])
			copy(allRedPhase, nextPhase)
			for i, state := range yellowPhase {
				if state == mapv2.LightState_LIGHT_STATE_GREEN && nextPhase[i] == mapv2.LightState_LIGHT_STATE_RED {
					yellowPhase[i] = mapv2.LightState_LIGHT_STATE_YELLOW
					if l.lanes[i].IsWalkLane() {
						hasClearPhase = true
						clearPhase[i] = mapv2.LightState_LIGHT_STATE_YELLOW
					}
				}
				if state == mapv2.LightState_LIGHT_STATE_RED && nextPhase[i] == mapv2.LightState_LIGHT_STATE_GREEN && !l.lanes[i].IsWalkLane() {
					allRedPhase[i] = mapv2.LightState_LIGHT_STATE_RED
					hasAllRedPhase = true
				}
			}
			// 顺序 最大压力信控相位1--行人清空相位--黄灯相位--车道全红相位--最大压力信控相位2
			l.runtime.transitionPhases = make([][]mapv2.LightState, 0)
			l.runtime.transitionTimes = make([]float64, 0)
			if hasClearPhase {
				l.runtime.transitionPhases = append(l.runtime.transitionPhases, clearPhase)
				l.runtime.transitionTimes = append(l.runtime.transitionTimes, *pedestrianClearTime)
			}
			l.runtime.transitionPhases = append(l.runtime.transitionPhases, yellowPhase)
			l.runtime.transitionTimes = append(l.runtime.transitionTimes, *yellowTime)
			if hasAllRedPhase {
				l.runtime.transitionPhases = append(l.runtime.transitionPhases, allRedPhase)
				l.runtime.transitionTimes = append(l.runtime.transitionTimes, *allRedTime)
			}
			l.runtime.remainingT += l.runtime.transitionTimes[0]
		}
	}
	if l.runtime.remainingT <= 0 {
		log.Warnf("traffic light %d remaining time %f <= 0", l.junctionID, l.runtime.remainingT)
	}
	// 更新相位后的totalTime即为remainingTime
	l.runtime.totalTime = l.runtime.remainingT
}

// Get 获取当前信号灯程序
// 功能：返回当前信号灯程序，最大压力算法不支持外部程序设置
// 返回：始终返回nil，因为最大压力算法不保存外部程序
func (l *mpTrafficLight) Get() *mapv2.TrafficLight {
	return nil
}

// Set 设置信号灯程序
// 功能：设置信号灯程序，最大压力算法不支持外部程序设置
// 参数：tl-信号灯程序
// 返回：错误信息，最大压力算法不支持此操作
func (l *mpTrafficLight) Set(tl *mapv2.TrafficLight) error {
	return ErrMaxPressure
}

// Unset 取消信号灯程序
// 功能：取消当前信号灯程序，最大压力算法不支持此操作
func (l *mpTrafficLight) Unset() {}

// SetPhase 设置信号灯相位
// 功能：设置信号灯相位，最大压力算法不支持外部相位设置
// 参数：offset-相位偏移，remainingTime-剩余时间
func (l *mpTrafficLight) SetPhase(offset int32, remainingTime float64) {}

// SetOk 设置信号灯状态
// 功能：设置信号灯的开关状态
// 参数：ok-信号灯状态，true表示正常工作，false表示失效（全绿灯）
func (l *mpTrafficLight) SetOk(ok bool) {
	l.okBuffer = ok
}

// Step 获取当前相位索引
// 功能：返回当前相位索引，最大压力算法返回-1表示动态相位
// 返回：当前相位索引，最大压力算法返回-1
func (l *mpTrafficLight) Step() int32 {
	return -1
}

// RemainingTime 获取当前相位剩余时间
// 功能：返回当前相位的剩余时间
// 返回：当前相位的剩余时间
func (l *mpTrafficLight) RemainingTime() float64 {
	return l.snapshotRemainingT
}

// Ok 获取信号灯状态
// 功能：返回信号灯是否正常工作
// 返回：true表示正常工作，false表示失效
func (l *mpTrafficLight) Ok() bool {
	return l.ok
}
