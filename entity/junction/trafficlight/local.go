package trafficlight

import (
	"fmt"

	"git.fiblab.net/general/common/v2/mathutil"
	mapv2 "git.fiblab.net/sim/protos/v2/go/city/map/v2"
	"github.com/tsinghua-fib-lab/agentsociety-sim-oss/entity"
)

// localTlRuntime 本地信号灯运行时数据结构
// 功能：存储固定相位信号灯的运行时状态，包括程序、相位索引、时间控制等
type localTlRuntime struct {
	tl           *mapv2.TrafficLight
	tlStep       int32
	tlTotalTime  float64
	tlRemainingT float64
}

// localTrafficLight 本地固定相位信号灯控制器
// 功能：实现基于固定程序的信号灯控制，按照预设的相位顺序和时间进行切换
type localTrafficLight struct {
	ctx entity.ITaskContext

	JunctionID int32                            // 所属junction ID
	lanes      []entity.ILaneTrafficLightSetter // 车道数据

	timeBeforeChange [][]float64     // 下一次信号灯变化时间（相位切换时不一定所有的信号灯都变）
	snapshot         localTlRuntime  // snapshot，用于保存输出的数据
	runtime          localTlRuntime  // 运行时数据
	buffer           *localTlRuntime // 数据buffer，用于交互式接口写入(optional)
	ok               bool            // 信号灯状态，true为开启，false为关闭
	okBuffer         bool            // 信号灯状态buffer，用于交互式接口写入
}

// NewLocalTrafficLight 创建固定相位信号灯控制器
// 功能：初始化本地信号灯控制器，设置基础参数和车道映射
// 参数：ctx-任务上下文，junctionID-路口ID，lanes-车道列表
// 返回：初始化完成的本地信号灯控制器实例
func NewLocalTrafficLight(ctx entity.ITaskContext, junctionID int32, lanes []entity.ILaneTrafficLightSetter) *localTrafficLight {
	return &localTrafficLight{
		ctx:              ctx,
		JunctionID:       junctionID,
		lanes:            lanes,
		timeBeforeChange: make([][]float64, 0),
		runtime:          localTlRuntime{},
		ok:               true,
		okBuffer:         true,
	}
}

// Prepare 准备阶段，处理信号灯的准备工作
// 功能：更新信号灯状态，将当前相位信息写入车道，处理全绿灯和固定相位情况
// 说明：如果没有信号灯程序或信号灯关闭，则保持全绿灯状态
func (l *localTrafficLight) Prepare() {
	// 更新信号灯状态
	l.ok = l.okBuffer
	// 写入snapshot
	l.snapshot = l.runtime
	// 写入lane中数据
	if l.snapshot.tl == nil || !l.ok {
		for _, lane := range l.lanes {
			lane.SetLight(mapv2.LightState_LIGHT_STATE_GREEN, mathutil.INF, mathutil.INF)
		}
	} else {
		p := l.snapshot.tl.Phases[l.snapshot.tlStep]
		for i, lane := range l.lanes {
			lane.SetLight(
				p.States[i],
				l.snapshot.tlTotalTime+l.timeBeforeChange[i][l.snapshot.tlStep],  // total time
				l.snapshot.tlRemainingT+l.timeBeforeChange[i][l.snapshot.tlStep], // remaining time
			)
		}
	}
}

// Update 更新阶段，执行固定相位信号灯的核心逻辑
// 功能：按照预设程序进行相位切换，处理时间计算和全红相位保持逻辑
// 参数：dt-时间步长
// 算法说明：
// 1. 处理buffer中的新程序设置
// 2. 计算每个车道在后续相位中的状态变化时间
// 3. 根据剩余时间进行相位切换
// 4. 支持全红相位保持功能（当有车辆时延长全红时间）
func (l *localTrafficLight) Update(dt float64) {
	if l.buffer != nil {
		l.runtime = *l.buffer
		l.buffer = nil
		// 初始化步骤
		if l.runtime.tl != nil {
			numPhases := len(l.runtime.tl.Phases)
			numLanes := len(l.lanes)

			for laneIndex := 0; laneIndex < numLanes; laneIndex++ {
				time := make([]float64, numPhases)

				// 检查所有状态是否相同
				allTheSame := true
				lastState := l.runtime.tl.Phases[numPhases-1].States[laneIndex]

				// 遍历所有相位（从后往前），计算每个相位的持续时间
				for phaseIndex := numPhases - 2; phaseIndex >= 0; phaseIndex-- {
					state := l.runtime.tl.Phases[phaseIndex+1].States[laneIndex]
					if state == lastState {
						time[phaseIndex] = time[phaseIndex+1] + l.runtime.tl.Phases[phaseIndex+1].Duration
					} else {
						allTheSame = false
					}
					lastState = state
				}

				// 如果所有状态相同，则设置为无限时间
				if allTheSame {
					for idx := range time {
						time[idx] = mathutil.INF
					}
				} else {
					// 调整时间以考虑第一个相位和最后一个相位的相邻关系
					t0 := time[0] + l.runtime.tl.Phases[0].Duration
					lastState = l.runtime.tl.Phases[numPhases-1].States[laneIndex]

					// 确保第一个相位和最后一个相位的状态一致时，更新时间
					if lastState == l.runtime.tl.Phases[0].States[laneIndex] {
						for phaseIndex := numPhases - 1; phaseIndex >= 0; phaseIndex-- {
							if lastState != l.runtime.tl.Phases[phaseIndex].States[laneIndex] {
								break
							}
							time[phaseIndex] += t0
						}
					}
				}

				// 将计算结果存储到车道状态变化时间列表中
				l.timeBeforeChange = append(l.timeBeforeChange, time)
			}
		}
	}
	if l.runtime.tl == nil || !l.ok {
		return
	}

	l.runtime.tlRemainingT -= dt
	// 切换相位
	if l.runtime.tlRemainingT <= 0 {
		l.runtime.tlRemainingT = 0
		l.runtime.tlTotalTime = 0
		// 正常切换相位逻辑
		for {
			l.runtime.tlStep = (l.runtime.tlStep + 1) % int32(len(l.runtime.tl.Phases))
			l.runtime.tlRemainingT += l.runtime.tl.Phases[l.runtime.tlStep].Duration
			if l.runtime.tlRemainingT > 0 {
				l.runtime.tlTotalTime = l.runtime.tlRemainingT
				break
			}
		}
	}
}

// Get 获取当前信号灯程序
// 功能：返回当前正在执行的信号灯程序
// 返回：当前信号灯程序，如果没有程序则返回nil
func (l *localTrafficLight) Get() *mapv2.TrafficLight {
	return l.snapshot.tl
}

// Set 设置信号灯程序
// 功能：设置新的信号灯程序，验证程序的有效性
// 参数：tl-信号灯程序
// 返回：设置结果，如果程序无效则返回错误
// 说明：程序设置会延迟到下一个更新周期生效
func (l *localTrafficLight) Set(tl *mapv2.TrafficLight) error {
	if tl.JunctionId != l.JunctionID {
		return fmt.Errorf("set junction %d with wrong traffic light id %d", l.JunctionID, tl.JunctionId)
	}
	if l.lanes == nil {
		return fmt.Errorf("no lane data in junction %d", l.JunctionID)
	}
	if tl.Phases == nil {
		return fmt.Errorf("set with empty traffic light")
	}
	for _, p := range tl.Phases {
		if len(p.States) != len(l.lanes) {
			return fmt.Errorf("number of lanes %d and traffic light states %d does not match", len(l.lanes), len(p.States))
		}
	}

	phaseIndex := l.JunctionID % int32(len(tl.Phases))
	l.buffer = &localTlRuntime{
		tl: tl, tlStep: phaseIndex, tlRemainingT: tl.Phases[phaseIndex].Duration,
	}
	return nil
}

// Unset 取消信号灯程序
// 功能：取消当前信号灯程序，使信号灯变为全绿灯状态
// 说明：取消操作会延迟到下一个更新周期生效
func (l *localTrafficLight) Unset() {
	l.buffer = &localTlRuntime{tl: nil, tlStep: 0, tlRemainingT: 0}
}

// SetPhase 设置信号灯相位
// 功能：设置当前相位索引和剩余时间
// 参数：offset-相位偏移，remainingT-剩余时间
// 说明：相位设置会延迟到下一个更新周期生效
func (l *localTrafficLight) SetPhase(offset int32, remainingT float64) {
	if l.runtime.tl == nil { // 当前没有信控程序
		return
	}
	if l.buffer != nil {
		l.buffer.tlRemainingT = remainingT
		l.buffer.tlStep = offset
	} else {
		l.buffer = &localTlRuntime{
			tl: l.runtime.tl, tlStep: offset, tlRemainingT: remainingT,
		}
	}
}

// SetOk 设置信号灯状态
// 功能：设置信号灯的开关状态
// 参数：ok-信号灯状态，true表示正常工作，false表示失效（全绿灯）
func (l *localTrafficLight) SetOk(ok bool) {
	l.okBuffer = ok
}

// Step 获取当前相位索引
// 功能：返回当前相位索引
// 返回：当前相位索引
func (l *localTrafficLight) Step() int32 {
	return l.snapshot.tlStep
}

// RemainingTime 获取当前相位剩余时间
// 功能：返回当前相位的剩余时间
// 返回：当前相位的剩余时间
func (l *localTrafficLight) RemainingTime() float64 {
	return l.snapshot.tlRemainingT
}

// Ok 获取信号灯状态
// 功能：返回信号灯是否正常工作
// 返回：true表示正常工作，false表示失效
func (l *localTrafficLight) Ok() bool {
	return l.ok
}
