package ecosim

import (
	"sync"

	economyv2 "git.fiblab.net/sim/protos/v2/go/city/economy/v2"
)

// Firm 代表企业实体
type Firm struct {
	mu   sync.RWMutex
	base *economyv2.Firm
}

// NewFirm 创建新的企业实例
func NewFirm(firm *economyv2.Firm) *Firm {
	return &Firm{
		base: firm,
	}
}

// GetID 获取企业ID
func (f *Firm) GetID() int32 {
	f.mu.RLock()
	defer f.mu.RUnlock()
	return f.base.Id
}

// GetCurrency 获取企业持有的货币量
func (f *Firm) GetCurrency() float32 {
	f.mu.RLock()
	defer f.mu.RUnlock()
	return f.base.Currency
}

// SetCurrency 设置企业持有的货币量
func (f *Firm) SetCurrency(value float32) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.base.Currency = value
}

// GetBase 获取底层proto消息
func (f *Firm) GetBase() *economyv2.Firm {
	f.mu.RLock()
	defer f.mu.RUnlock()
	return f.base
}

// GetPrice 获取价格
func (f *Firm) GetPrice() float32 {
	f.mu.RLock()
	defer f.mu.RUnlock()
	return f.base.Price
}

// SetPrice 设置价格
func (f *Firm) SetPrice(value float32) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.base.Price = value
}

// GetInventory 获取库存
func (f *Firm) GetInventory() int32 {
	f.mu.RLock()
	defer f.mu.RUnlock()
	return f.base.Inventory
}

// SetInventory 设置库存
func (f *Firm) SetInventory(value int32) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.base.Inventory = value
}

// GetDemand 获取需求量
func (f *Firm) GetDemand() float32 {
	f.mu.RLock()
	defer f.mu.RUnlock()
	return f.base.Demand
}

// SetDemand 设置需求量
func (f *Firm) SetDemand(value float32) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.base.Demand = value
}

// GetSales 获取销售量
func (f *Firm) GetSales() float32 {
	f.mu.RLock()
	defer f.mu.RUnlock()
	return f.base.Sales
}

// SetSales 设置销售量
func (f *Firm) SetSales(value float32) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.base.Sales = value
}

// GetEmployees 获取员工列表
func (f *Firm) GetEmployees() []int32 {
	f.mu.RLock()
	defer f.mu.RUnlock()
	return f.base.Employees
}

// SetEmployees 设置员工列表
func (f *Firm) SetEmployees(value []int32) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.base.Employees = value
}

// NBS 代表国家统计局实体
type NBS struct {
	mu   sync.RWMutex
	base *economyv2.NBS
}

// NewNBS 创建新的国家统计局实例
func NewNBS(nbs *economyv2.NBS) *NBS {
	return &NBS{
		base: nbs,
	}
}

// GetID 获取统计局ID
func (n *NBS) GetID() int32 {
	n.mu.RLock()
	defer n.mu.RUnlock()
	return n.base.Id
}

// GetCurrency 获取统计局持有的货币量
func (n *NBS) GetCurrency() float32 {
	n.mu.RLock()
	defer n.mu.RUnlock()
	return n.base.Currency
}

// SetCurrency 设置统计局持有的货币量
func (n *NBS) SetCurrency(value float32) {
	n.mu.Lock()
	defer n.mu.Unlock()
	n.base.Currency = value
}

// GetBase 获取底层proto消息
func (n *NBS) GetBase() *economyv2.NBS {
	n.mu.RLock()
	defer n.mu.RUnlock()
	return n.base
}

// GetNominalGDP 获取名义GDP
func (n *NBS) GetNominalGDP() map[string]float32 {
	n.mu.RLock()
	defer n.mu.RUnlock()
	return n.base.NominalGdp
}

// SetNominalGDP 设置名义GDP
func (n *NBS) SetNominalGDP(value map[string]float32) {
	n.mu.Lock()
	defer n.mu.Unlock()
	n.base.NominalGdp = value
}

// GetRealGDP 获取实际GDP
func (n *NBS) GetRealGDP() map[string]float32 {
	n.mu.RLock()
	defer n.mu.RUnlock()
	return n.base.RealGdp
}

// SetRealGDP 设置实际GDP
func (n *NBS) SetRealGDP(value map[string]float32) {
	n.mu.Lock()
	defer n.mu.Unlock()
	n.base.RealGdp = value
}

// GetUnemployment 获取失业率
func (n *NBS) GetUnemployment() map[string]float32 {
	n.mu.RLock()
	defer n.mu.RUnlock()
	return n.base.Unemployment
}

// SetUnemployment 设置失业率
func (n *NBS) SetUnemployment(value map[string]float32) {
	n.mu.Lock()
	defer n.mu.Unlock()
	n.base.Unemployment = value
}

// GetWages 获取工资水平
func (n *NBS) GetWages() map[string]float32 {
	n.mu.RLock()
	defer n.mu.RUnlock()
	return n.base.Wages
}

// SetWages 设置工资水平
func (n *NBS) SetWages(value map[string]float32) {
	n.mu.Lock()
	defer n.mu.Unlock()
	n.base.Wages = value
}

// GetPrices 获取价格水平
func (n *NBS) GetPrices() map[string]float32 {
	n.mu.RLock()
	defer n.mu.RUnlock()
	return n.base.Prices
}

// SetPrices 设置价格水平
func (n *NBS) SetPrices(value map[string]float32) {
	n.mu.Lock()
	defer n.mu.Unlock()
	n.base.Prices = value
}

// GetWorkingHours 获取工作时长
func (n *NBS) GetWorkingHours() map[string]float32 {
	n.mu.RLock()
	defer n.mu.RUnlock()
	return n.base.WorkingHours
}

// SetWorkingHours 设置工作时长
func (n *NBS) SetWorkingHours(value map[string]float32) {
	n.mu.Lock()
	defer n.mu.Unlock()
	n.base.WorkingHours = value
}

// GetDepression 获取抑郁指数
func (n *NBS) GetDepression() map[string]float32 {
	n.mu.RLock()
	defer n.mu.RUnlock()
	return n.base.Depression
}

// SetDepression 设置抑郁指数
func (n *NBS) SetDepression(value map[string]float32) {
	n.mu.Lock()
	defer n.mu.Unlock()
	n.base.Depression = value
}

// GetConsumptionCurrency 获取消费货币
func (n *NBS) GetConsumptionCurrency() map[string]float32 {
	n.mu.RLock()
	defer n.mu.RUnlock()
	return n.base.ConsumptionCurrency
}

// SetConsumptionCurrency 设置消费货币
func (n *NBS) SetConsumptionCurrency(value map[string]float32) {
	n.mu.Lock()
	defer n.mu.Unlock()
	n.base.ConsumptionCurrency = value
}

// GetIncomeCurrency 获取收入货币
func (n *NBS) GetIncomeCurrency() map[string]float32 {
	n.mu.RLock()
	defer n.mu.RUnlock()
	return n.base.IncomeCurrency
}

// SetIncomeCurrency 设置收入货币
func (n *NBS) SetIncomeCurrency(value map[string]float32) {
	n.mu.Lock()
	defer n.mu.Unlock()
	n.base.IncomeCurrency = value
}

// GetLocusControl 获取控制点
func (n *NBS) GetLocusControl() map[string]float32 {
	n.mu.RLock()
	defer n.mu.RUnlock()
	return n.base.LocusControl
}

// SetLocusControl 设置控制点
func (n *NBS) SetLocusControl(value map[string]float32) {
	n.mu.Lock()
	defer n.mu.Unlock()
	n.base.LocusControl = value
}

// Government 代表政府实体
type Government struct {
	mu   sync.RWMutex
	base *economyv2.Government
}

// NewGovernment 创建新的政府实例
func NewGovernment(gov *economyv2.Government) *Government {
	return &Government{
		base: gov,
	}
}

// GetID 获取政府ID
func (g *Government) GetID() int32 {
	g.mu.RLock()
	defer g.mu.RUnlock()
	return g.base.Id
}

// GetCurrency 获取政府持有的货币量
func (g *Government) GetCurrency() float32 {
	g.mu.RLock()
	defer g.mu.RUnlock()
	return g.base.Currency
}

// SetCurrency 设置政府持有的货币量
func (g *Government) SetCurrency(value float32) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.base.Currency = value
}

// GetBase 获取底层proto消息
func (g *Government) GetBase() *economyv2.Government {
	g.mu.RLock()
	defer g.mu.RUnlock()
	return g.base
}

// GetBracketRates 获取税率
func (g *Government) GetBracketRates() []float32 {
	g.mu.RLock()
	defer g.mu.RUnlock()
	return g.base.BracketRates
}

// SetBracketRates 设置税率
func (g *Government) SetBracketRates(value []float32) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.base.BracketRates = value
}

// GetBracketCutoffs 获取税率档位切分点
func (g *Government) GetBracketCutoffs() []float32 {
	g.mu.RLock()
	defer g.mu.RUnlock()
	return g.base.BracketCutoffs
}

// SetBracketCutoffs 设置税率档位切分点
func (g *Government) SetBracketCutoffs(value []float32) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.base.BracketCutoffs = value
}

// Bank 代表银行实体
type Bank struct {
	mu   sync.RWMutex
	base *economyv2.Bank
}

// NewBank 创建新的银行实例
func NewBank(bank *economyv2.Bank) *Bank {
	return &Bank{
		base: bank,
	}
}

// GetID 获取银行ID
func (b *Bank) GetID() int32 {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.base.Id
}

// GetCurrency 获取银行持有的货币量
func (b *Bank) GetCurrency() float32 {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.base.Currency
}

// SetCurrency 设置银行持有的货币量
func (b *Bank) SetCurrency(value float32) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.base.Currency = value
}

// GetBase 获取底层proto消息
func (b *Bank) GetBase() *economyv2.Bank {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.base
}

// GetInterestRate 获取利率
func (b *Bank) GetInterestRate() float32 {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.base.InterestRate
}

// SetInterestRate 设置利率
func (b *Bank) SetInterestRate(value float32) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.base.InterestRate = value
}
