package junction

import (
	mapv2 "git.fiblab.net/sim/protos/v2/go/city/map/v2"
)

// 依赖倒置，表达junction对信号灯实现的接口需求

// 给交通参与者提供的信控读取接口
type ITrafficLightGetter interface {
	Get() *mapv2.TrafficLight // 当前程序
	Step() int32              // 当前相位
	RemainingTime() float64   // 当前相位剩余时长
	Ok() bool                 // 当前信控开关情况
}

// 信号灯接口
type ITrafficLight interface {
	ITrafficLightGetter
	Prepare()          // 准备阶段，处理各种写入buffer，将信控结果写入到lane中
	Update(dt float64) // 更新阶段，更新信控结果

	Set(tl *mapv2.TrafficLight) error             // 修改信控程序
	Unset()                                       // 删除信控程序（全绿）
	SetPhase(offset int32, remainingTime float64) // 修改信控相位到指定值
	SetOk(ok bool)                                // 设置信控开关情况（true信控工作|false信控失效-全绿）
}
