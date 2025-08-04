package task

import (
	"flag"
	"sync"
)

const (
	SelfName = "city" // 本程序在模拟任务集群中的名字
)

var (
	heartBeatInterval = flag.Int("log.heartbeat_interval", 100, "心跳日志间隔步数")
)

// prepare 准备阶段，每步执行一次
// 功能：在每个仿真步骤开始时进行准备工作
// 算法说明：
// 1. 更新时钟：增加内部步数并计算当前时间
// 2. 日志文件轮转：切换到新的日志文件
// 3. 心跳日志：定期输出系统状态信息
// 4. 并行准备：并发执行各个管理器的准备操作
//   - 人员管理器：准备节点和人员数据
//   - 车道管理器：准备车道数据
//   - 路口管理器：准备路口数据
//   - AOI管理器：准备区域数据
//   - 出租车管理器：准备出租车数据
//
// 说明：确保所有系统组件在更新阶段前都处于正确状态
func (ctx *Context) prepare() {
	log.Debugf("step %d complete, +1", ctx.clock.InternalStep)
	ctx.clock.InternalStep++
	log.Debugf("step %d complete, +1 ok", ctx.clock.InternalStep)
	ctx.clock.T = float64(ctx.clock.InternalStep) * ctx.clock.DT

	if ctx.clock.InternalStep%int32(*heartBeatInterval) == 0 {
		hour, minute, second := ctx.clock.GetHourMinuteSecond()
		log.Infof(
			"STEP: %d(%d:%d:%.2f)",
			ctx.clock.InternalStep,
			hour, minute, second,
		)
	}

	// Prepare
	var wg sync.WaitGroup

	{
		wg.Add(1)
		go func() {
			defer wg.Done()
			ctx.personManager.PrepareNode()

			var subWg sync.WaitGroup
			subWg.Add(1)
			go func() {
				defer subWg.Done()
				ctx.personManager.Prepare() // person
			}()
			subWg.Add(1)
			go func() {
				defer subWg.Done()
				ctx.laneManager.Prepare() // lane
			}()
			subWg.Wait()
			subWg.Add(1)
			go func() {
				defer subWg.Done()
				ctx.junctionManager.Prepare() // junction
			}()
			subWg.Wait()
		}()
		wg.Add(1)
		go func() {
			defer wg.Done()
			ctx.aoiManager.Prepare() // aoi
		}()
		wg.Wait()
	}
}

// update 更新阶段，每步执行一次
// 功能：在每个仿真步骤中执行主要的仿真逻辑
// 算法说明：
// 1. 统计输出：在非子循环步骤中输出各种统计数据
//   - 车辆微观统计：记录车辆详细状态
//   - 车道统计：记录车道状态信息
//   - 道路统计：记录道路状态信息
//
// 2. 并行更新：并发执行各个管理器的更新操作
//   - 人员管理器：更新人员状态和行为
//   - AOI管理器：更新区域状态
//   - 路口管理器：更新信号灯状态
//   - 车道管理器：更新车道状态
//   - 出租车管理器：更新出租车状态
//
// 3. 输出处理：在非子循环步骤中处理各种输出
//   - 通用输出：复杂格式的输出数据
//   - 简单输出：简化格式的输出数据
//
// 说明：这是仿真的核心阶段，执行所有实体的状态更新
func (ctx *Context) update() {
	var wg sync.WaitGroup

	// Update
	{
		wg.Add(1)
		go func() {
			defer wg.Done()
			ctx.personManager.Update(ctx.clock.DT) // person
		}()
		wg.Add(1)
		go func() {
			defer wg.Done()
			ctx.aoiManager.Update(ctx.clock.DT) // aoi
		}()
		wg.Add(1)
		go func() {
			defer wg.Done()
			ctx.junctionManager.Update(ctx.clock.DT) // junction
		}()
		wg.Add(1)
		go func() {
			defer wg.Done()
			ctx.laneManager.Update() // lane
		}()
	}
	wg.Wait()
}

// Run 运行
func (ctx *Context) Run() {
	// 初始化
	ctx.Init()
	// init syncer
	ctx.sidecar.Step(false)
	for {
		ctx.prepare()
		// 通知准备阶段完成
		log.Debugf("step %d: prepare complete and call NotifyStepReady", ctx.clock.InternalStep)
		ctx.sidecar.NotifyStepReady()
		log.Debugf("step %d: NotifyStepReady complete", ctx.clock.InternalStep)
		ctx.update()
		log.Debugf("step %d: update complete", ctx.clock.InternalStep)
		close := false
		if ctx.clock.InternalStep+1 >= ctx.clock.END_STEP {
			close = ctx.sidecar.Step(true)
		} else {
			close = ctx.sidecar.Step(false)
		}
		if close || ctx.closed.Load() {
			break
		}
	}
	log.Infof("engine complete")
	ctx.Close()
}
