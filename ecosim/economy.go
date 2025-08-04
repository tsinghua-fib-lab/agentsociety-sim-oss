package ecosim

import (
	"fmt"
	"os"
	"sync"

	economyv2 "git.fiblab.net/sim/protos/v2/go/city/economy/v2"
	"google.golang.org/protobuf/proto"
)

// EconomySim 代表经济模拟系统
type EconomySim struct {
	agents map[int32]*Agent
	firms  map[int32]*Firm
	nbs    map[int32]*NBS
	govs   map[int32]*Government
	banks  map[int32]*Bank
	mu     sync.Mutex
}

// SimError 自定义错误类型
type SimError struct {
	Message string
}

func (e *SimError) Error() string {
	return e.Message
}

// NewEconomySim 创建新的经济模拟系统实例
func NewEconomySim() *EconomySim {
	return &EconomySim{
		agents: make(map[int32]*Agent),
		firms:  make(map[int32]*Firm),
		nbs:    make(map[int32]*NBS),
		govs:   make(map[int32]*Government),
		banks:  make(map[int32]*Bank),
	}
}

// AddAgent 添加新代理
func (e *EconomySim) AddAgent(agent *economyv2.Agent) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	if _, exists := e.agents[agent.Id]; exists {
		return fmt.Errorf("agent %d already exists", agent.Id)
	}

	e.agents[agent.Id] = NewAgent(agent)
	return nil
}

// RemoveAgent 移除代理
func (e *EconomySim) RemoveAgent(agentID int32) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	if _, exists := e.agents[agentID]; !exists {
		return fmt.Errorf("agent %d not found", agentID)
	}

	delete(e.agents, agentID)
	return nil
}

// AddFirm 添加新企业
func (e *EconomySim) AddFirm(firm *economyv2.Firm) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	if _, exists := e.firms[firm.Id]; exists {
		return fmt.Errorf("firm %d already exists", firm.Id)
	}
	e.firms[firm.Id] = NewFirm(firm)
	return nil
}

// AddNBS 添加新的国家统计局
func (e *EconomySim) AddNBS(nbs *economyv2.NBS) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	if _, exists := e.nbs[nbs.Id]; exists {
		return fmt.Errorf("NBS %d already exists", nbs.Id)
	}
	e.nbs[nbs.Id] = NewNBS(nbs)
	return nil
}

// AddGovernment 添加新政府
func (e *EconomySim) AddGovernment(gov *economyv2.Government) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	if _, exists := e.govs[gov.Id]; exists {
		return fmt.Errorf("government %d already exists", gov.Id)
	}
	e.govs[gov.Id] = NewGovernment(gov)
	return nil
}

// AddBank 添加新银行
func (e *EconomySim) AddBank(bank *economyv2.Bank) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	if _, exists := e.banks[bank.Id]; exists {
		return fmt.Errorf("bank %d already exists", bank.Id)
	}
	e.banks[bank.Id] = NewBank(bank)
	return nil
}

// RemoveFirm 移除企业
func (e *EconomySim) RemoveFirm(firmID int32) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	if _, exists := e.firms[firmID]; !exists {
		return fmt.Errorf("firm %d not found", firmID)
	}
	delete(e.firms, firmID)
	return nil
}

// RemoveNBS 移除国家统计局
func (e *EconomySim) RemoveNBS(nbsID int32) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	if _, exists := e.nbs[nbsID]; !exists {
		return fmt.Errorf("NBS %d not found", nbsID)
	}
	delete(e.nbs, nbsID)
	return nil
}

// RemoveGovernment 移除政府
func (e *EconomySim) RemoveGovernment(govID int32) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	if _, exists := e.govs[govID]; !exists {
		return fmt.Errorf("government %d not found", govID)
	}
	delete(e.govs, govID)
	return nil
}

// RemoveBank 移除银行
func (e *EconomySim) RemoveBank(bankID int32) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	if _, exists := e.banks[bankID]; !exists {
		return fmt.Errorf("bank %d not found", bankID)
	}
	delete(e.banks, bankID)
	return nil
}

// UpdateFirm 更新企业
func (e *EconomySim) UpdateFirm(firm *economyv2.Firm) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	if _, exists := e.firms[firm.Id]; !exists {
		return fmt.Errorf("firm %d not found", firm.Id)
	}
	e.firms[firm.Id] = NewFirm(firm)
	return nil
}

// UpdateNBS 更新国家统计局
func (e *EconomySim) UpdateNBS(nbs *economyv2.NBS) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	if _, exists := e.nbs[nbs.Id]; !exists {
		return fmt.Errorf("NBS %d not found", nbs.Id)
	}
	e.nbs[nbs.Id] = NewNBS(nbs)
	return nil
}

// UpdateGovernment 更新政府
func (e *EconomySim) UpdateGovernment(gov *economyv2.Government) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	if _, exists := e.govs[gov.Id]; !exists {
		return fmt.Errorf("government %d not found", gov.Id)
	}
	e.govs[gov.Id] = NewGovernment(gov)
	return nil
}

// UpdateBank 更新银行
func (e *EconomySim) UpdateBank(bank *economyv2.Bank) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	if _, exists := e.banks[bank.Id]; !exists {
		return fmt.Errorf("bank %d not found", bank.Id)
	}
	e.banks[bank.Id] = NewBank(bank)
	return nil
}

// GetOrg 获取组织
func (e *EconomySim) GetOrg(orgID int32) (interface{}, error) {
	e.mu.Lock()
	defer e.mu.Unlock()

	if org, exists := e.firms[orgID]; exists {
		return org, nil
	}
	if org, exists := e.nbs[orgID]; exists {
		return org, nil
	}
	if org, exists := e.govs[orgID]; exists {
		return org, nil
	}
	if org, exists := e.banks[orgID]; exists {
		return org, nil
	}
	return nil, &SimError{Message: fmt.Sprintf("organization %d not found", orgID)}
}

// UpdateOrg 更新组织
func (e *EconomySim) UpdateOrg(org interface{}) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	switch o := org.(type) {
	case *economyv2.Firm:
		if _, exists := e.firms[o.Id]; !exists {
			return &SimError{Message: fmt.Sprintf("firm %d not found", o.Id)}
		}
		e.firms[o.Id] = NewFirm(o)
	case *economyv2.NBS:
		if _, exists := e.nbs[o.Id]; !exists {
			return &SimError{Message: fmt.Sprintf("NBS %d not found", o.Id)}
		}
		e.nbs[o.Id] = NewNBS(o)
	case *economyv2.Government:
		if _, exists := e.govs[o.Id]; !exists {
			return &SimError{Message: fmt.Sprintf("government %d not found", o.Id)}
		}
		e.govs[o.Id] = NewGovernment(o)
	case *economyv2.Bank:
		if _, exists := e.banks[o.Id]; !exists {
			return &SimError{Message: fmt.Sprintf("bank %d not found", o.Id)}
		}
		e.banks[o.Id] = NewBank(o)
	default:
		return &SimError{Message: "unsupported organization type"}
	}
	return nil
}

// GetAgent 获取代理
func (e *EconomySim) GetAgent(agentID int32) (*Agent, error) {
	e.mu.Lock()
	defer e.mu.Unlock()

	agent, exists := e.agents[agentID]
	if !exists {
		return nil, fmt.Errorf("agent %d not found", agentID)
	}

	return agent, nil
}

// UpdateAgent 更新代理
func (e *EconomySim) UpdateAgent(agent *economyv2.Agent) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	existingAgent, exists := e.agents[agent.Id]
	if !exists {
		return fmt.Errorf("agent %d not found", agent.Id)
	}

	existingAgent.base = agent
	return nil
}

// CalculateTaxesDue 计算应缴税额
func (e *EconomySim) CalculateTaxesDue(governmentID int32, agentIDs []int32, incomes []float32, enableRedistribution bool) (float32, []float32, error) {
	e.mu.Lock()
	defer e.mu.Unlock()

	// 获取政府实例
	gov, exists := e.govs[governmentID]
	if !exists {
		return 0, nil, fmt.Errorf("government %d not found", governmentID)
	}

	// 获取税率档位和切分点
	bracketCutoffs := gov.GetBracketCutoffs()
	if len(bracketCutoffs) == 0 {
		bracketCutoffs = DefaultBracketCutoffs
	}

	bracketRates := gov.GetBracketRates()
	if len(bracketRates) == 0 {
		bracketRates = DefaultBracketRates
	}

	// 检查输入参数长度是否匹配
	if len(agentIDs) != len(incomes) {
		return 0, nil, fmt.Errorf("length of agent IDs and incomes must match")
	}

	var totalTax float32
	updatedIncomes := make([]float32, 0, len(incomes))

	// 计算每个代理的税收和更新收入
	for i, agentID := range agentIDs {
		// 检查代理是否存在
		agent, exists := e.agents[agentID]
		if !exists {
			return 0, nil, fmt.Errorf("agent %d not found", agentID)
		}

		// 计算税收
		tax := taxesDue(incomes[i], bracketCutoffs, bracketRates)
		totalTax += tax

		// 更新收入和代理货币
		currentIncome := incomes[i] - tax
		updatedIncomes = append(updatedIncomes, currentIncome)
		currentCurrency := agent.GetCurrency()
		agent.SetCurrency(currentCurrency + currentIncome)
	}

	// 处理再分配
	if enableRedistribution {
		// 计算每人分得的金额
		var lumpSum float32
		if len(agentIDs) > 0 {
			lumpSum = totalTax / float32(len(agentIDs))
		}

		// 更新每个代理的货币
		for _, agentID := range agentIDs {
			agent := e.agents[agentID]
			currentCurrency := agent.GetCurrency()
			agent.SetCurrency(currentCurrency + lumpSum)
		}
	} else {
		// 更新政府货币
		currentCurrency := gov.GetCurrency()
		gov.SetCurrency(currentCurrency + totalTax)
	}

	return totalTax, updatedIncomes, nil
}

// CalculateConsumption 计算消费
func (e *EconomySim) CalculateConsumption(firmIDs []int32, agentID int32, demands []int32, consumptionAccumulation bool) (float32, bool, error) {
	e.mu.Lock()
	defer e.mu.Unlock()

	// 检查参数
	if len(firmIDs) != len(demands) {
		return 0, false, fmt.Errorf("number of firms and demands must match")
	}

	// 获取代理实例
	agent, exists := e.agents[agentID]
	if !exists {
		return 0, false, fmt.Errorf("agent %d not found", agentID)
	}

	// 获取代理的货币量
	agentCurrency := agent.GetCurrency()

	// 计算总消费
	var totalConsumption float32
	var success bool = true

	type salesInfo struct {
		firmID      int32
		actualSales int32
		cost        float32
	}
	var sales []salesInfo

	// 计算每个企业的销售情况
	for i, firmID := range firmIDs {
		firm, exists := e.firms[firmID]
		if !exists {
			return 0, false, fmt.Errorf("firm %d not found", firmID)
		}

		demand := demands[i]
		price := firm.GetPrice()
		inventory := firm.GetInventory()

		// 检查库存是否足够
		var actualSales int32
		if inventory >= demand {
			actualSales = demand
		} else {
			actualSales = inventory
			success = false
		}

		cost := float32(actualSales) * price
		if cost > agentCurrency {
			actualSales = int32(agentCurrency / price)
			cost = float32(actualSales) * price
			success = false
		}

		if actualSales > 0 {
			sales = append(sales, salesInfo{
				firmID:      firmID,
				actualSales: actualSales,
				cost:        cost,
			})
			totalConsumption += cost
			agentCurrency -= cost
		}
	}

	// 如果不累积消费，则更新代理和企业的状态
	if !consumptionAccumulation {
		// 更新代理的货币量
		agent.SetCurrency(agentCurrency)

		// 更新代理的消费量
		currentConsumption := float32(0)
		if consumption := agent.GetConsumption(); consumption != nil {
			currentConsumption = *consumption
		}
		newConsumption := currentConsumption + totalConsumption
		agent.SetConsumption(&newConsumption)

		// 更新企业的状态
		for _, sale := range sales {
			firm := e.firms[sale.firmID]
			firm.SetCurrency(firm.GetCurrency() + sale.cost)
			firm.SetInventory(firm.GetInventory() - sale.actualSales)
			firm.SetDemand(firm.GetDemand() + float32(sale.actualSales))
			firm.SetSales(firm.GetSales() + float32(sale.actualSales))
		}
	}

	return totalConsumption, success, nil
}

// CalculateInterest 计算利息
func (e *EconomySim) CalculateInterest(bankID int32, agentIDs []int32) (float32, []float32, error) {
	e.mu.Lock()
	defer e.mu.Unlock()

	// 获取银行实例
	bank, exists := e.banks[bankID]
	if !exists {
		return 0, nil, fmt.Errorf("bank %d not found", bankID)
	}

	// 获取利率
	interestRate := bank.GetInterestRate()
	if interestRate <= 0 {
		return 0, nil, fmt.Errorf("invalid interest rate for bank %d", bankID)
	}

	// 计算每个代理的利息
	var totalInterest float32
	updatedCurrencies := make([]float32, len(agentIDs))

	for i, agentID := range agentIDs {
		agent, exists := e.agents[agentID]
		if !exists {
			return 0, nil, fmt.Errorf("agent %d not found", agentID)
		}

		currency := agent.GetCurrency()
		interest := currency * interestRate
		totalInterest += interest

		// 更新代理的货币量
		newCurrency := currency + interest
		agent.SetCurrency(newCurrency)
		updatedCurrencies[i] = newCurrency
	}

	// 检查银行是否有足够的货币支付利息
	bankCurrency := bank.GetCurrency()
	if bankCurrency < totalInterest {
		return 0, nil, fmt.Errorf("bank %d does not have enough currency to pay interest", bankID)
	}

	// 更新银行的货币量
	bank.SetCurrency(bankCurrency - totalInterest)

	return totalInterest, updatedCurrencies, nil
}

// GetFirmIDs 获取所有企业ID
func (e *EconomySim) GetFirmIDs() []int32 {
	e.mu.Lock()
	defer e.mu.Unlock()

	var ids []int32
	for id := range e.firms {
		ids = append(ids, id)
	}
	return ids
}

// GetNBSIDs 获取所有国家统计局ID
func (e *EconomySim) GetNBSIDs() []int32 {
	e.mu.Lock()
	defer e.mu.Unlock()

	var ids []int32
	for id := range e.nbs {
		ids = append(ids, id)
	}
	return ids
}

// GetGovernmentIDs 获取所有政府ID
func (e *EconomySim) GetGovernmentIDs() []int32 {
	e.mu.Lock()
	defer e.mu.Unlock()

	var ids []int32
	for id := range e.govs {
		ids = append(ids, id)
	}
	return ids
}

// GetBankIDs 获取所有银行ID
func (e *EconomySim) GetBankIDs() []int32 {
	e.mu.Lock()
	defer e.mu.Unlock()

	var ids []int32
	for id := range e.banks {
		ids = append(ids, id)
	}
	return ids
}

// SaveEntities 保存经济实体状态
func (e *EconomySim) SaveEntities(filePath string) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	// 创建实体列表
	entities := &economyv2.EconomyEntities{
		Firms:       make([]*economyv2.Firm, 0),
		Nbs:         make([]*economyv2.NBS, 0),
		Governments: make([]*economyv2.Government, 0),
		Banks:       make([]*economyv2.Bank, 0),
		Agents:      make([]*economyv2.Agent, 0),
	}

	// 保存组织
	for _, firm := range e.firms {
		entities.Firms = append(entities.Firms, firm.GetBase())
	}
	for _, nbs := range e.nbs {
		entities.Nbs = append(entities.Nbs, nbs.GetBase())
	}
	for _, gov := range e.govs {
		entities.Governments = append(entities.Governments, gov.GetBase())
	}
	for _, bank := range e.banks {
		entities.Banks = append(entities.Banks, bank.GetBase())
	}

	// 保存代理
	for _, agent := range e.agents {
		entities.Agents = append(entities.Agents, agent.base)
	}

	// 序列化并保存到文件
	data, err := proto.Marshal(entities)
	if err != nil {
		return fmt.Errorf("failed to marshal entities: %v", err)
	}

	if err := os.WriteFile(filePath, data, 0644); err != nil {
		return fmt.Errorf("failed to write file: %v", err)
	}

	return nil
}

// LoadEntities 加载经济实体状态
func (e *EconomySim) LoadEntities(filePath string) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	// 从文件读取数据
	data, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to read file: %v", err)
	}

	// 反序列化数据
	entities := &economyv2.EconomyEntities{}
	if err := proto.Unmarshal(data, entities); err != nil {
		return fmt.Errorf("failed to unmarshal entities: %v", err)
	}

	// 清空当前状态
	e.firms = make(map[int32]*Firm)
	e.nbs = make(map[int32]*NBS)
	e.govs = make(map[int32]*Government)
	e.banks = make(map[int32]*Bank)
	e.agents = make(map[int32]*Agent)

	// 加载组织
	for _, firm := range entities.Firms {
		e.firms[firm.Id] = NewFirm(firm)
	}
	for _, nbs := range entities.Nbs {
		e.nbs[nbs.Id] = NewNBS(nbs)
	}
	for _, gov := range entities.Governments {
		e.govs[gov.Id] = NewGovernment(gov)
	}
	for _, bank := range entities.Banks {
		e.banks[bank.Id] = NewBank(bank)
	}

	// 加载代理
	for _, agent := range entities.Agents {
		e.agents[agent.Id] = NewAgent(agent)
	}

	return nil
}

// GetFirm 获取企业
func (e *EconomySim) GetFirm(firmID int32) (*Firm, error) {
	e.mu.Lock()
	defer e.mu.Unlock()

	firm, exists := e.firms[firmID]
	if !exists {
		return nil, fmt.Errorf("firm %d not found", firmID)
	}
	return firm, nil
}

// GetNBS 获取国家统计局
func (e *EconomySim) GetNBS(nbsID int32) (*NBS, error) {
	e.mu.Lock()
	defer e.mu.Unlock()

	nbs, exists := e.nbs[nbsID]
	if !exists {
		return nil, fmt.Errorf("NBS %d not found", nbsID)
	}
	return nbs, nil
}

// GetGovernment 获取政府
func (e *EconomySim) GetGovernment(govID int32) (*Government, error) {
	e.mu.Lock()
	defer e.mu.Unlock()

	gov, exists := e.govs[govID]
	if !exists {
		return nil, fmt.Errorf("government %d not found", govID)
	}
	return gov, nil
}

// GetBank 获取银行
func (e *EconomySim) GetBank(bankID int32) (*Bank, error) {
	e.mu.Lock()
	defer e.mu.Unlock()

	bank, exists := e.banks[bankID]
	if !exists {
		return nil, fmt.Errorf("bank %d not found", bankID)
	}
	return bank, nil
}

// DeltaUpdateFirm 增量更新企业
func (e *EconomySim) DeltaUpdateFirm(firmID int32, deltaInventory *int32, deltaPrice, deltaCurrency *float32, deltaDemand, deltaSales *float32, addEmployees, removeEmployees []int32) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	firm, exists := e.firms[firmID]
	if !exists {
		return fmt.Errorf("firm %d not found", firmID)
	}

	if deltaInventory != nil {
		firm.SetInventory(firm.GetInventory() + *deltaInventory)
	}
	if deltaPrice != nil {
		firm.SetPrice(firm.GetPrice() + *deltaPrice)
	}
	if deltaCurrency != nil {
		firm.SetCurrency(firm.GetCurrency() + *deltaCurrency)
	}
	if deltaDemand != nil {
		firm.SetDemand(firm.GetDemand() + *deltaDemand)
	}
	if deltaSales != nil {
		firm.SetSales(firm.GetSales() + *deltaSales)
	}

	// 更新员工列表
	if len(addEmployees) > 0 || len(removeEmployees) > 0 {
		currentEmployees := firm.GetEmployees()

		// 创建一个map来存储当前所有员工ID，用于快速查找和去重
		employeeMap := make(map[int32]bool, len(currentEmployees))
		for _, empID := range currentEmployees {
			employeeMap[empID] = true
		}

		// 移除员工 - 使用map优化
		if len(removeEmployees) > 0 {
			// 标记要移除的员工
			for _, empID := range removeEmployees {
				delete(employeeMap, empID)
			}
		}

		// 添加员工 - 使用map避免重复
		if len(addEmployees) > 0 {
			for _, empID := range addEmployees {
				employeeMap[empID] = true
			}
		}

		// 从map重建员工列表
		newEmployees := make([]int32, 0, len(employeeMap))
		for empID := range employeeMap {
			newEmployees = append(newEmployees, empID)
		}

		firm.SetEmployees(newEmployees)
	}

	return nil
}

// DeltaUpdateNBS 增量更新国家统计局
func (e *EconomySim) DeltaUpdateNBS(nbsID int32, deltaNominalGDP, deltaRealGDP, deltaUnemployment, deltaWages, deltaPrices, deltaWorkingHours, deltaDepression, deltaConsumptionCurrency, deltaIncomeCurrency, deltaLocusControl map[string]float32, deltaCurrency *float32, addCitizenIDs, removeCitizenIDs []int32) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	nbs, exists := e.nbs[nbsID]
	if !exists {
		return fmt.Errorf("NBS %d not found", nbsID)
	}

	// 更新时间序列数据
	if deltaNominalGDP != nil {
		current := nbs.GetNominalGDP()
		for k, v := range deltaNominalGDP {
			current[k] += v
		}
		nbs.SetNominalGDP(current)
	}
	if deltaRealGDP != nil {
		current := nbs.GetRealGDP()
		for k, v := range deltaRealGDP {
			current[k] += v
		}
		nbs.SetRealGDP(current)
	}
	if deltaUnemployment != nil {
		current := nbs.GetUnemployment()
		for k, v := range deltaUnemployment {
			current[k] += v
		}
		nbs.SetUnemployment(current)
	}
	if deltaWages != nil {
		current := nbs.GetWages()
		for k, v := range deltaWages {
			current[k] += v
		}
		nbs.SetWages(current)
	}
	if deltaPrices != nil {
		current := nbs.GetPrices()
		for k, v := range deltaPrices {
			current[k] += v
		}
		nbs.SetPrices(current)
	}
	if deltaWorkingHours != nil {
		current := nbs.GetWorkingHours()
		for k, v := range deltaWorkingHours {
			current[k] += v
		}
		nbs.SetWorkingHours(current)
	}
	if deltaDepression != nil {
		current := nbs.GetDepression()
		for k, v := range deltaDepression {
			current[k] += v
		}
		nbs.SetDepression(current)
	}
	if deltaConsumptionCurrency != nil {
		current := nbs.GetConsumptionCurrency()
		for k, v := range deltaConsumptionCurrency {
			current[k] += v
		}
		nbs.SetConsumptionCurrency(current)
	}
	if deltaIncomeCurrency != nil {
		current := nbs.GetIncomeCurrency()
		for k, v := range deltaIncomeCurrency {
			current[k] += v
		}
		nbs.SetIncomeCurrency(current)
	}
	if deltaLocusControl != nil {
		current := nbs.GetLocusControl()
		for k, v := range deltaLocusControl {
			current[k] += v
		}
		nbs.SetLocusControl(current)
	}

	if deltaCurrency != nil {
		nbs.SetCurrency(nbs.GetCurrency() + *deltaCurrency)
	}

	// 处理公民ID列表的添加和删除
	if len(addCitizenIDs) > 0 || len(removeCitizenIDs) > 0 {
		// 创建当前公民ID的map，用于快速查找
		citizenMap := make(map[int32]bool)
		for _, citizenID := range nbs.GetBase().CitizenIds {
			citizenMap[citizenID] = true
		}

		// 移除公民ID
		if len(removeCitizenIDs) > 0 {
			for _, citizenID := range removeCitizenIDs {
				delete(citizenMap, citizenID)
			}
		}

		// 添加公民ID
		if len(addCitizenIDs) > 0 {
			for _, citizenID := range addCitizenIDs {
				citizenMap[citizenID] = true
			}
		}

		// 从map重建公民ID列表
		newCitizenIDs := make([]int32, 0, len(citizenMap))
		for citizenID := range citizenMap {
			newCitizenIDs = append(newCitizenIDs, citizenID)
		}

		nbs.GetBase().CitizenIds = newCitizenIDs
	}

	return nil
}

// DeltaUpdateGovernment 增量更新政府
func (e *EconomySim) DeltaUpdateGovernment(govID int32, deltaBracketCutoffs, deltaBracketRates []float32, deltaCurrency *float32, addCitizenIDs, removeCitizenIDs []int32) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	gov, exists := e.govs[govID]
	if !exists {
		return fmt.Errorf("government %d not found", govID)
	}

	if deltaBracketCutoffs != nil {
		current := gov.GetBracketCutoffs()
		for i := range current {
			if i < len(deltaBracketCutoffs) {
				current[i] += deltaBracketCutoffs[i]
			}
		}
		gov.SetBracketCutoffs(current)
	}

	if deltaBracketRates != nil {
		current := gov.GetBracketRates()
		for i := range current {
			if i < len(deltaBracketRates) {
				current[i] += deltaBracketRates[i]
			}
		}
		gov.SetBracketRates(current)
	}

	if deltaCurrency != nil {
		gov.SetCurrency(gov.GetCurrency() + *deltaCurrency)
	}

	// 处理公民ID列表的添加和删除
	if len(addCitizenIDs) > 0 || len(removeCitizenIDs) > 0 {
		// 创建当前公民ID的map，用于快速查找
		citizenMap := make(map[int32]bool)
		for _, citizenID := range gov.GetBase().CitizenIds {
			citizenMap[citizenID] = true
		}

		// 移除公民ID
		if len(removeCitizenIDs) > 0 {
			for _, citizenID := range removeCitizenIDs {
				delete(citizenMap, citizenID)
			}
		}

		// 添加公民ID
		if len(addCitizenIDs) > 0 {
			for _, citizenID := range addCitizenIDs {
				citizenMap[citizenID] = true
			}
		}

		// 从map重建公民ID列表
		newCitizenIDs := make([]int32, 0, len(citizenMap))
		for citizenID := range citizenMap {
			newCitizenIDs = append(newCitizenIDs, citizenID)
		}

		gov.GetBase().CitizenIds = newCitizenIDs
	}

	return nil
}

// DeltaUpdateBank 增量更新银行
func (e *EconomySim) DeltaUpdateBank(bankID int32, deltaInterestRate, deltaCurrency *float32, addCitizenIDs, removeCitizenIDs []int32) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	bank, exists := e.banks[bankID]
	if !exists {
		return fmt.Errorf("bank %d not found", bankID)
	}

	if deltaInterestRate != nil {
		bank.SetInterestRate(bank.GetInterestRate() + *deltaInterestRate)
	}

	if deltaCurrency != nil {
		bank.SetCurrency(bank.GetCurrency() + *deltaCurrency)
	}

	// 处理公民ID列表的添加和删除
	if len(addCitizenIDs) > 0 || len(removeCitizenIDs) > 0 {
		// 创建当前公民ID的map，用于快速查找
		citizenMap := make(map[int32]bool)
		for _, citizenID := range bank.GetBase().CitizenIds {
			citizenMap[citizenID] = true
		}

		// 移除公民ID
		if len(removeCitizenIDs) > 0 {
			for _, citizenID := range removeCitizenIDs {
				delete(citizenMap, citizenID)
			}
		}

		// 添加公民ID
		if len(addCitizenIDs) > 0 {
			for _, citizenID := range addCitizenIDs {
				citizenMap[citizenID] = true
			}
		}

		// 从map重建公民ID列表
		newCitizenIDs := make([]int32, 0, len(citizenMap))
		for citizenID := range citizenMap {
			newCitizenIDs = append(newCitizenIDs, citizenID)
		}

		bank.GetBase().CitizenIds = newCitizenIDs
	}

	return nil
}

// DeltaUpdateAgent 增量更新代理
func (e *EconomySim) DeltaUpdateAgent(update *economyv2.AgentDeltaUpdate) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	agent, exists := e.agents[update.AgentId]
	if !exists {
		return fmt.Errorf("agent %d not found", update.AgentId)
	}

	if update.DeltaCurrency != nil {
		agent.SetCurrency(agent.GetCurrency() + *update.DeltaCurrency)
	}

	if update.NewFirmId != nil {
		agent.SetFirmID(update.NewFirmId)
	}

	if update.DeltaSkill != nil {
		currentSkill := float32(0)
		if agent.GetSkill() != nil {
			currentSkill = *agent.GetSkill()
		}
		newSkill := currentSkill + *update.DeltaSkill
		agent.SetSkill(&newSkill)
	}

	if update.DeltaConsumption != nil {
		currentConsumption := float32(0)
		if agent.GetConsumption() != nil {
			currentConsumption = *agent.GetConsumption()
		}
		newConsumption := currentConsumption + *update.DeltaConsumption
		agent.SetConsumption(&newConsumption)
	}

	if update.DeltaIncome != nil {
		currentIncome := float32(0)
		if agent.GetIncome() != nil {
			currentIncome = *agent.GetIncome()
		}
		newIncome := currentIncome + *update.DeltaIncome
		agent.SetIncome(&newIncome)
	}

	return nil
}

// CalculateRealGDP 计算实际GDP
func (e *EconomySim) CalculateRealGDP(nbsID int32) (float32, error) {
	e.mu.Lock()
	defer e.mu.Unlock()

	nbs, exists := e.nbs[nbsID]
	if !exists {
		return 0, fmt.Errorf("NBS %d not found", nbsID)
	}

	// 获取名义GDP和价格水平
	nominalGDP := nbs.GetNominalGDP()
	prices := nbs.GetPrices()

	// 计算实际GDP
	var realGDP float32
	for timestamp, gdp := range nominalGDP {
		if price, ok := prices[timestamp]; ok && price > 0 {
			realGDP += gdp / price
		}
	}

	return realGDP, nil
}
