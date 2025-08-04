## economy.go

### 基础CRUD方法
- AddOrg(ctx, req) 
- RemoveOrg(ctx, req)
- AddAgent(ctx, req)
- RemoveAgent(ctx, req)

### GDP相关方法
- GetNominalGDP(ctx, req)
- SetNominalGDP(ctx, req)
- GetRealGDP(ctx, req)
- SetRealGDP(ctx, req)

### 统计数据相关方法
- GetUnemployment(ctx, req)
- SetUnemployment(ctx, req)
- GetWages(ctx, req)
- SetWages(ctx, req)
- GetPrices(ctx, req)
- SetPrices(ctx, req)
- GetWorkingHours(ctx, req)
- SetWorkingHours(ctx, req)

### 库存和价格相关方法
- GetInventory(ctx, req)
- SetInventory(ctx, req)
- GetPrice(ctx, req)
- SetPrice(ctx, req)

### 税率相关方法
- GetBracketRates(ctx, req)
- SetBracketRates(ctx, req)
- GetBracketCutoffs(ctx, req)
- SetBracketCutoffs(ctx, req)

### 货币相关方法
- GetCurrency(ctx, req)
- SetCurrency(ctx, req)
- GetInterestRate(ctx, req)
- SetInterestRate(ctx, req)

### 心理指标
- GetDepression(ctx, req)
- SetDepression(ctx, req)
- GetLocusControl(ctx, req)
- SetLocusControl(ctx, req)

### 消费和收入相关方法
- GetConsumptionCurrency(ctx, req)
- SetConsumptionCurrency(ctx, req)
- GetConsumptionPropensity(ctx, req)
- SetConsumptionPropensity(ctx, req)
- GetIncomeCurrency(ctx, req)
- SetIncomeCurrency(ctx, req)

### 计算相关方法
- CalculateTaxesDue(ctx, req)
- CalculateConsumption(ctx, req)
- CalculateInterest(ctx, req)

### Add系列方法
- AddCurrency(ctx, req)
- AddPrice(ctx, req)
- AddInterestRate(ctx, req)
- AddInventory(ctx, req)

### 实体管理方法
- GetOrgEntityIds(ctx, req)
- SaveEconomyEntities(ctx, req)
- LoadEconomyEntities(ctx, req)


---
economy.go

// 基础CRUD方法
- AddAgent(agent *pb.Agent) error
- RemoveAgent(agentID int32) error
- AddOrg(org *pb.Org) error
- RemoveOrg(orgID int32) error

// 数据获取和设置方法
- GetNominalGDP(orgID int32) ([]float64, error)
- SetNominalGDP(orgID int32, value []float64) error
- GetRealGDP(orgID int32) ([]float64, error)
- SetRealGDP(orgID int32, value []float64) error
- GetUnemployment(orgID int32) ([]float64, error)
- SetUnemployment(orgID int32, value []float64) error
- GetWages(orgID int32) ([]float64, error)
- SetWages(orgID int32, value []float64) error
- GetPrices(orgID int32) ([]float64, error)
- SetPrices(orgID int32, value []float64) error
- GetInventory(orgID int32) (int32, error)
- SetInventory(orgID int32, value int32) error
- GetDepression(orgID int32) ([]float64, error)
- SetDepression(orgID int32, value []float64) error
- GetLocusControl(orgID int32) ([]float64, error)
- SetLocusControl(orgID int32, value []float64) error
- GetWorkingHours(orgID int32) ([]float64, error)
- SetWorkingHours(orgID int32, value []float64) error

// 计算方法
- CalculateTaxes(orgID int32, agentIDs []int32, incomes []float64, enableRedistribution bool) (float64, []float64, error)
- CalculateConsumption(orgID int32, income float64) (float64, error)
- CalculateInterest(orgID int32, principal float64) (float64, error)

// Add系列方法
- AddCurrency(orgID int32, deltaCurrency float64) (float64, error)
- AddPrice(orgID int32, deltaPrice float64) (float64, error)
- AddInterestRate(orgID int32, deltaInterestRate float64) (float64, error)
- AddInventory(orgID int32, deltaInventory int32) (int32, error)

// 实体管理方法
- GetOrgEntityIds(orgType pb.OrgType) ([]int32, error)
- SaveEntities(filePath string) error
- LoadEntities(filePath string) error