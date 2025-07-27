package clock

import (
	"fmt"

	"git.fiblab.net/sim/protos/v2/go/city/clock/v1/clockv1connect"
	"git.fiblab.net/sim/simulet-go/utils/config"
)

// Clock 仿真时钟管理器
// 功能：管理仿真系统的时间推进，支持子循环机制以提高仿真精度
// 说明：维护当前仿真时间、步数等信息，提供时间格式化和RPC服务
type Clock struct {
	clockv1connect.UnimplementedClockServiceHandler

	DT         float64 // 每个实际模拟步时间间隔（秒）
	SUBLOOP    int32   // 每个实际模拟步内部循环次数
	START_STEP int32   // 起始步
	END_STEP   int32   // 结束步，模拟区间[START, END)

	T            float64 // 当前时间（秒）
	InternalStep int32   // 当前内部步数
}

// New 根据配置创建新的时钟实例
// 功能：根据全局配置初始化时钟信息，支持子循环机制
// 参数：stepConfig-控制步配置，包含时间间隔、子循环数等信息
// 返回：初始化完成的时钟实例
// 算法说明：
// 1. 获取子循环数（默认为1）
// 2. 计算实际时间步长：dt = interval / subloop
// 3. 计算起始和结束步数（考虑子循环缩放）
// 4. 初始化时钟状态
// 说明：子循环机制允许在保持输出兼容性的同时提高仿真精度
func New(stepConfig config.ControlStep) *Clock {
	subloop := int32(1)
	dt := stepConfig.Interval / float64(subloop)
	startStep := stepConfig.Start * (subloop)
	endStep := (stepConfig.Start + stepConfig.Total) * (subloop)

	c := &Clock{
		DT:         dt,
		SUBLOOP:    subloop,
		START_STEP: startStep,
		END_STEP:   endStep,
	}
	c.Init()
	return c
}

// Init 初始化时钟状态
// 功能：设置仿真天数和重置时钟状态
// 参数：day-仿真天数
// 说明：重置内部步数为起始步，重新计算当前时间
func (c *Clock) Init() {
	c.InternalStep = c.START_STEP
	c.T = float64(c.InternalStep) * c.DT
}

// ExternalStep 获取用于输出的步数值
// 功能：将内部步数转换为外部步数（按原始interval计算）
// 返回：外部步数
// 说明：保持与原有输出格式的兼容性，外部看到的仍然是按interval间隔的数据
func (c *Clock) ExternalStep() int32 {
	return c.InternalStep / c.SUBLOOP
}

// ExternalStartStep 获取外部起始步数
// 功能：计算用于输出的起始步数
// 返回：外部起始步数
// 说明：将内部起始步数转换为外部步数格式
func (c *Clock) ExternalStartStep() int32 {
	return c.START_STEP / c.SUBLOOP
}

// NoInSubloop 检查是否不在子循环内
// 功能：判断当前是否为可以进行输出的时刻
// 返回：true表示可以进行输出，false表示在子循环内
// 说明：只有当一个完整的时钟周期内的所有子循环都完成时才允许输出
func (c *Clock) NoInSubloop() bool {
	return c.InternalStep%c.SUBLOOP == 0
}

// String 获取时钟的字符串表示
// 功能：将当前时间格式化为可读的字符串
// 返回：格式化的时间字符串（Day X: HH:MM:SS）
// 算法说明：
// 1. 将总秒数转换为小时、分钟、秒
// 2. 格式化为标准时间格式
func (c *Clock) String() string {
	t := c.T
	h := int(t / 3600)
	t -= float64(h * 3600)
	m := int(t / 60)
	t -= float64(m * 60)
	s := int(t)
	return fmt.Sprintf("%02d:%02d:%02d", h, m, s)
}

// GetHourMinuteSecond 获取当前时间的小时、分钟、秒
// 功能：将当前时间分解为小时、分钟、秒三个部分
// 返回：小时、分钟、秒（秒为浮点数，支持亚秒级精度）
// 算法说明：
// 1. 计算小时数：总秒数除以3600
// 2. 计算分钟数：剩余秒数除以60
// 3. 计算秒数：最终剩余秒数（浮点数）
func (c *Clock) GetHourMinuteSecond() (int, int, float64) {
	hour := int(c.T) / 3600
	minute := int(c.T) % 3600 / 60
	second := c.T - float64(hour*3600+minute*60)
	return hour, minute, second
}
