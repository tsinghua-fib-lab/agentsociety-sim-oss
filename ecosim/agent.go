package ecosim

import (
	"sync"

	economyv2 "git.fiblab.net/sim/protos/v2/go/city/economy/v2"
)

// Agent 代表经济系统中的个体代理
type Agent struct {
	base *economyv2.Agent
	mu   sync.Mutex
}

// NewAgent 创建新的代理实例
func NewAgent(agent *economyv2.Agent) *Agent {
	return &Agent{
		base: agent,
	}
}

// GetID 获取代理ID
func (a *Agent) GetID() int32 {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.base.Id
}

// GetCurrency 获取代理持有的货币量
func (a *Agent) GetCurrency() float32 {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.base.Currency == nil {
		return 0
	}
	return *a.base.Currency
}

// SetCurrency 设置代理持有的货币量
func (a *Agent) SetCurrency(value float32) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.base.Currency = &value
}

// GetFirmID 获取代理所属公司ID
func (a *Agent) GetFirmID() *int32 {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.base.FirmId
}

// SetFirmID 设置代理所属公司ID
func (a *Agent) SetFirmID(value *int32) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.base.FirmId = value
}

// GetSkill 获取代理的技能水平
func (a *Agent) GetSkill() *float32 {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.base.Skill
}

// SetSkill 设置代理的技能水平
func (a *Agent) SetSkill(value *float32) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.base.Skill = value
}

// GetConsumption 获取代理的消费量
func (a *Agent) GetConsumption() *float32 {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.base.Consumption
}

// SetConsumption 设置代理的消费量
func (a *Agent) SetConsumption(value *float32) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.base.Consumption = value
}

// GetIncome 获取代理的收入
func (a *Agent) GetIncome() *float32 {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.base.Income
}

// SetIncome 设置代理的收入
func (a *Agent) SetIncome(value *float32) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.base.Income = value
}
