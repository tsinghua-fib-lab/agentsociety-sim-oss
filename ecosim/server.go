package ecosim

import (
	"context"
	"fmt"
	"net/http"

	"connectrpc.com/connect"
	economyv2 "git.fiblab.net/sim/protos/v2/go/city/economy/v2"
	economyv2connect "git.fiblab.net/sim/protos/v2/go/city/economy/v2/economyv2connect"
)

// Server 实现gRPC服务器
type Server struct {
	economyv2connect.UnimplementedOrgServiceHandler
	econ *EconomySim
}

// NewServer 创建新的服务器实例
func NewServer() *Server {
	return &Server{
		econ: NewEconomySim(),
	}
}

// RunServer 启动gRPC服务器
func RunServer(address string) error {
	mux := http.NewServeMux()
	path, handler := economyv2connect.NewOrgServiceHandler(NewServer())
	mux.Handle(path, handler)

	log.Printf("Server listening at %v", address)
	return http.ListenAndServe(address, mux)
}

// AddFirm 添加企业
func (s *Server) AddFirm(ctx context.Context, req *connect.Request[economyv2.AddFirmRequest]) (*connect.Response[economyv2.AddFirmResponse], error) {
	// 处理批量添加
	var firmIDs []int32
	for _, firm := range req.Msg.Firms {
		if err := s.econ.AddFirm(firm); err != nil {
			return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to add firm: %v", err))
		}
		firmIDs = append(firmIDs, firm.Id)
	}
	return connect.NewResponse(&economyv2.AddFirmResponse{
		FirmIds: firmIDs,
	}), nil
}

// RemoveFirm 移除企业
func (s *Server) RemoveFirm(ctx context.Context, req *connect.Request[economyv2.RemoveFirmRequest]) (*connect.Response[economyv2.RemoveFirmResponse], error) {
	// 处理批量删除
	for _, firmID := range req.Msg.FirmIds {
		if err := s.econ.RemoveFirm(firmID); err != nil {
			return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to remove firm: %v", err))
		}
	}
	return connect.NewResponse(&economyv2.RemoveFirmResponse{}), nil
}

// GetFirm 获取企业信息
func (s *Server) GetFirm(ctx context.Context, req *connect.Request[economyv2.GetFirmRequest]) (*connect.Response[economyv2.GetFirmResponse], error) {
	var firms []*economyv2.Firm
	for _, firmID := range req.Msg.FirmIds {
		firm, exists := s.econ.firms[firmID]
		if !exists {
			return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("firm %d not found", firmID))
		}
		firms = append(firms, firm.GetBase())
	}
	return connect.NewResponse(&economyv2.GetFirmResponse{
		Firms: firms,
	}), nil
}

// UpdateFirm 更新企业信息
func (s *Server) UpdateFirm(ctx context.Context, req *connect.Request[economyv2.UpdateFirmRequest]) (*connect.Response[economyv2.UpdateFirmResponse], error) {
	for _, firm := range req.Msg.Firms {
		if err := s.econ.UpdateFirm(firm); err != nil {
			return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to update firm: %v", err))
		}
	}
	return connect.NewResponse(&economyv2.UpdateFirmResponse{}), nil
}

// ListFirms 列出所有企业
func (s *Server) ListFirms(ctx context.Context, req *connect.Request[economyv2.ListFirmsRequest]) (*connect.Response[economyv2.ListFirmsResponse], error) {
	var firmList []*economyv2.Firm
	for _, firm := range s.econ.firms {
		firmList = append(firmList, firm.GetBase())
	}
	return connect.NewResponse(&economyv2.ListFirmsResponse{
		Firms: firmList,
	}), nil
}

// DeltaUpdateFirm 增量更新企业
func (s *Server) DeltaUpdateFirm(ctx context.Context, req *connect.Request[economyv2.DeltaUpdateFirmRequest]) (*connect.Response[economyv2.DeltaUpdateFirmResponse], error) {
	for _, update := range req.Msg.Updates {
		if err := s.econ.DeltaUpdateFirm(
			update.FirmId,
			update.DeltaInventory,
			update.DeltaPrice,
			update.DeltaCurrency,
			update.DeltaDemand,
			update.DeltaSales,
			update.AddEmployees,
			update.RemoveEmployees,
		); err != nil {
			return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to delta update firm: %v", err))
		}
	}
	return connect.NewResponse(&economyv2.DeltaUpdateFirmResponse{}), nil
}

// AddAgent 添加新代理
func (s *Server) AddAgent(ctx context.Context, req *connect.Request[economyv2.AddAgentRequest]) (*connect.Response[economyv2.AddAgentResponse], error) {
	var agentIDs []int32
	for _, agent := range req.Msg.Agents {
		if err := s.econ.AddAgent(agent); err != nil {
			return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to add agent: %v", err))
		}
		agentIDs = append(agentIDs, agent.Id)
	}
	return connect.NewResponse(&economyv2.AddAgentResponse{
		AgentIds: agentIDs,
	}), nil
}

// RemoveAgent 移除代理
func (s *Server) RemoveAgent(ctx context.Context, req *connect.Request[economyv2.RemoveAgentRequest]) (*connect.Response[economyv2.RemoveAgentResponse], error) {
	for _, agentID := range req.Msg.AgentIds {
		if err := s.econ.RemoveAgent(agentID); err != nil {
			return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to remove agent: %v", err))
		}
	}
	return connect.NewResponse(&economyv2.RemoveAgentResponse{}), nil
}

// GetAgent 获取代理信息
func (s *Server) GetAgent(ctx context.Context, req *connect.Request[economyv2.GetAgentRequest]) (*connect.Response[economyv2.GetAgentResponse], error) {
	var agents []*economyv2.Agent
	for _, agentID := range req.Msg.AgentIds {
		agent, err := s.econ.GetAgent(agentID)
		if err != nil {
			return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to get agent: %v", err))
		}
		agents = append(agents, agent.base)
	}
	return connect.NewResponse(&economyv2.GetAgentResponse{
		Agents: agents,
	}), nil
}

// UpdateAgent 更新代理信息
func (s *Server) UpdateAgent(ctx context.Context, req *connect.Request[economyv2.UpdateAgentRequest]) (*connect.Response[economyv2.UpdateAgentResponse], error) {
	for _, agent := range req.Msg.Agents {
		if err := s.econ.UpdateAgent(agent); err != nil {
			return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to update agent: %v", err))
		}
	}
	return connect.NewResponse(&economyv2.UpdateAgentResponse{}), nil
}

// DeltaUpdateAgent 增量更新代理
func (s *Server) DeltaUpdateAgent(ctx context.Context, req *connect.Request[economyv2.DeltaUpdateAgentRequest]) (*connect.Response[economyv2.DeltaUpdateAgentResponse], error) {
	for _, update := range req.Msg.Updates {
		if err := s.econ.DeltaUpdateAgent(update); err != nil {
			return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to delta update agent: %v", err))
		}
	}
	return connect.NewResponse(&economyv2.DeltaUpdateAgentResponse{}), nil
}

// ListAgents 列出所有代理
func (s *Server) ListAgents(ctx context.Context, req *connect.Request[economyv2.ListAgentsRequest]) (*connect.Response[economyv2.ListAgentsResponse], error) {
	agents := make([]*economyv2.Agent, 0)
	for _, agent := range s.econ.agents {
		agents = append(agents, agent.base)
	}
	return connect.NewResponse(&economyv2.ListAgentsResponse{
		Agents: agents,
	}), nil
}

// CalculateTaxesDue 计算应缴税额
func (s *Server) CalculateTaxesDue(ctx context.Context, req *connect.Request[economyv2.CalculateTaxesDueRequest]) (*connect.Response[economyv2.CalculateTaxesDueResponse], error) {
	totalTax, updatedIncomes, err := s.econ.CalculateTaxesDue(
		req.Msg.GovernmentId,
		req.Msg.AgentIds,
		req.Msg.Incomes,
		req.Msg.EnableRedistribution,
	)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to calculate taxes: %v", err))
	}
	return connect.NewResponse(&economyv2.CalculateTaxesDueResponse{
		TaxesDue:       totalTax,
		UpdatedIncomes: updatedIncomes,
	}), nil
}

// CalculateConsumption 计算消费
func (s *Server) CalculateConsumption(ctx context.Context, req *connect.Request[economyv2.CalculateConsumptionRequest]) (*connect.Response[economyv2.CalculateConsumptionResponse], error) {
	accumulation := false
	if req.Msg.ConsumptionAccumulation != nil {
		accumulation = *req.Msg.ConsumptionAccumulation
	}
	actualConsumption, success, err := s.econ.CalculateConsumption(
		req.Msg.FirmIds,
		req.Msg.AgentId,
		req.Msg.Demands,
		accumulation,
	)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to calculate consumption: %v", err))
	}
	return connect.NewResponse(&economyv2.CalculateConsumptionResponse{
		ActualConsumption: actualConsumption,
		Success:           success,
	}), nil
}

// CalculateInterest 计算利息
func (s *Server) CalculateInterest(ctx context.Context, req *connect.Request[economyv2.CalculateInterestRequest]) (*connect.Response[economyv2.CalculateInterestResponse], error) {
	totalInterest, updatedCurrencies, err := s.econ.CalculateInterest(
		req.Msg.BankId,
		req.Msg.AgentIds,
	)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to calculate interest: %v", err))
	}
	return connect.NewResponse(&economyv2.CalculateInterestResponse{
		TotalInterest:     totalInterest,
		UpdatedCurrencies: updatedCurrencies,
	}), nil
}

// CalculateRealGDP 计算实际GDP
func (s *Server) CalculateRealGDP(ctx context.Context, req *connect.Request[economyv2.CalculateRealGDPRequest]) (*connect.Response[economyv2.CalculateRealGDPResponse], error) {
	realGDP, err := s.econ.CalculateRealGDP(req.Msg.NbsId)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to calculate real GDP: %v", err))
	}
	return connect.NewResponse(&economyv2.CalculateRealGDPResponse{
		RealGdp: realGDP,
	}), nil
}

// SaveEconomyEntities 保存经济实体状态
func (s *Server) SaveEconomyEntities(ctx context.Context, req *connect.Request[economyv2.SaveEconomyEntitiesRequest]) (*connect.Response[economyv2.SaveEconomyEntitiesResponse], error) {
	if err := s.econ.SaveEntities(req.Msg.FilePath); err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to save entities: %v", err))
	}
	return connect.NewResponse(&economyv2.SaveEconomyEntitiesResponse{}), nil
}

// LoadEconomyEntities 加载经济实体状态
func (s *Server) LoadEconomyEntities(ctx context.Context, req *connect.Request[economyv2.LoadEconomyEntitiesRequest]) (*connect.Response[economyv2.LoadEconomyEntitiesResponse], error) {
	if err := s.econ.LoadEntities(req.Msg.FilePath); err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to load entities: %v", err))
	}
	return connect.NewResponse(&economyv2.LoadEconomyEntitiesResponse{}), nil
}

// AddNBS 添加国家统计局
func (s *Server) AddNBS(ctx context.Context, req *connect.Request[economyv2.AddNBSRequest]) (*connect.Response[economyv2.AddNBSResponse], error) {
	if err := s.econ.AddNBS(req.Msg.Nbs); err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to add NBS: %v", err))
	}
	return connect.NewResponse(&economyv2.AddNBSResponse{}), nil
}

// RemoveNBS 移除国家统计局
func (s *Server) RemoveNBS(ctx context.Context, req *connect.Request[economyv2.RemoveNBSRequest]) (*connect.Response[economyv2.RemoveNBSResponse], error) {
	if err := s.econ.RemoveNBS(req.Msg.NbsId); err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to remove NBS: %v", err))
	}
	return connect.NewResponse(&economyv2.RemoveNBSResponse{}), nil
}

// GetNBS 获取国家统计局信息
func (s *Server) GetNBS(ctx context.Context, req *connect.Request[economyv2.GetNBSRequest]) (*connect.Response[economyv2.GetNBSResponse], error) {
	nbs, exists := s.econ.nbs[req.Msg.NbsId]
	if !exists {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("NBS %d not found", req.Msg.NbsId))
	}
	return connect.NewResponse(&economyv2.GetNBSResponse{
		Nbs: nbs.GetBase(),
	}), nil
}

// UpdateNBS 更新国家统计局信息
func (s *Server) UpdateNBS(ctx context.Context, req *connect.Request[economyv2.UpdateNBSRequest]) (*connect.Response[economyv2.UpdateNBSResponse], error) {
	if err := s.econ.UpdateNBS(req.Msg.Nbs); err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to update NBS: %v", err))
	}
	return connect.NewResponse(&economyv2.UpdateNBSResponse{}), nil
}

// ListNBS 列出所有国家统计局
func (s *Server) ListNBS(ctx context.Context, req *connect.Request[economyv2.ListNBSRequest]) (*connect.Response[economyv2.ListNBSResponse], error) {
	var nbsList []*economyv2.NBS
	for _, nbs := range s.econ.nbs {
		nbsList = append(nbsList, nbs.GetBase())
	}
	return connect.NewResponse(&economyv2.ListNBSResponse{
		NbsList: nbsList,
	}), nil
}

// DeltaUpdateNBS 增量更新国家统计局
func (s *Server) DeltaUpdateNBS(ctx context.Context, req *connect.Request[economyv2.DeltaUpdateNBSRequest]) (*connect.Response[economyv2.DeltaUpdateNBSResponse], error) {
	if err := s.econ.DeltaUpdateNBS(
		req.Msg.NbsId,
		req.Msg.DeltaNominalGdp,
		req.Msg.DeltaRealGdp,
		req.Msg.DeltaUnemployment,
		req.Msg.DeltaWages,
		req.Msg.DeltaPrices,
		req.Msg.DeltaWorkingHours,
		req.Msg.DeltaDepression,
		req.Msg.DeltaConsumptionCurrency,
		req.Msg.DeltaIncomeCurrency,
		req.Msg.DeltaLocusControl,
		req.Msg.DeltaCurrency,
		req.Msg.AddCitizenIds,
		req.Msg.RemoveCitizenIds,
	); err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to delta update NBS: %v", err))
	}
	return connect.NewResponse(&economyv2.DeltaUpdateNBSResponse{}), nil
}

// AddGovernment 添加政府
func (s *Server) AddGovernment(ctx context.Context, req *connect.Request[economyv2.AddGovernmentRequest]) (*connect.Response[economyv2.AddGovernmentResponse], error) {
	if err := s.econ.AddGovernment(req.Msg.Government); err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to add government: %v", err))
	}
	return connect.NewResponse(&economyv2.AddGovernmentResponse{}), nil
}

// RemoveGovernment 移除政府
func (s *Server) RemoveGovernment(ctx context.Context, req *connect.Request[economyv2.RemoveGovernmentRequest]) (*connect.Response[economyv2.RemoveGovernmentResponse], error) {
	if err := s.econ.RemoveGovernment(req.Msg.GovernmentId); err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to remove government: %v", err))
	}
	return connect.NewResponse(&economyv2.RemoveGovernmentResponse{}), nil
}

// GetGovernment 获取政府信息
func (s *Server) GetGovernment(ctx context.Context, req *connect.Request[economyv2.GetGovernmentRequest]) (*connect.Response[economyv2.GetGovernmentResponse], error) {
	gov, exists := s.econ.govs[req.Msg.GovernmentId]
	if !exists {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("government %d not found", req.Msg.GovernmentId))
	}
	return connect.NewResponse(&economyv2.GetGovernmentResponse{
		Government: gov.GetBase(),
	}), nil
}

// UpdateGovernment 更新政府信息
func (s *Server) UpdateGovernment(ctx context.Context, req *connect.Request[economyv2.UpdateGovernmentRequest]) (*connect.Response[economyv2.UpdateGovernmentResponse], error) {
	if err := s.econ.UpdateGovernment(req.Msg.Government); err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to update government: %v", err))
	}
	return connect.NewResponse(&economyv2.UpdateGovernmentResponse{}), nil
}

// ListGovernments 列出所有政府
func (s *Server) ListGovernments(ctx context.Context, req *connect.Request[economyv2.ListGovernmentsRequest]) (*connect.Response[economyv2.ListGovernmentsResponse], error) {
	var govList []*economyv2.Government
	for _, gov := range s.econ.govs {
		govList = append(govList, gov.GetBase())
	}
	return connect.NewResponse(&economyv2.ListGovernmentsResponse{
		Governments: govList,
	}), nil
}

// DeltaUpdateGovernment 增量更新政府
func (s *Server) DeltaUpdateGovernment(ctx context.Context, req *connect.Request[economyv2.DeltaUpdateGovernmentRequest]) (*connect.Response[economyv2.DeltaUpdateGovernmentResponse], error) {
	if err := s.econ.DeltaUpdateGovernment(
		req.Msg.GovernmentId,
		req.Msg.DeltaBracketCutoffs,
		req.Msg.DeltaBracketRates,
		req.Msg.DeltaCurrency,
		req.Msg.AddCitizenIds,
		req.Msg.RemoveCitizenIds,
	); err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to delta update government: %v", err))
	}
	return connect.NewResponse(&economyv2.DeltaUpdateGovernmentResponse{}), nil
}

// AddBank 添加银行
func (s *Server) AddBank(ctx context.Context, req *connect.Request[economyv2.AddBankRequest]) (*connect.Response[economyv2.AddBankResponse], error) {
	if err := s.econ.AddBank(req.Msg.Bank); err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to add bank: %v", err))
	}
	return connect.NewResponse(&economyv2.AddBankResponse{}), nil
}

// RemoveBank 移除银行
func (s *Server) RemoveBank(ctx context.Context, req *connect.Request[economyv2.RemoveBankRequest]) (*connect.Response[economyv2.RemoveBankResponse], error) {
	if err := s.econ.RemoveBank(req.Msg.BankId); err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to remove bank: %v", err))
	}
	return connect.NewResponse(&economyv2.RemoveBankResponse{}), nil
}

// GetBank 获取银行信息
func (s *Server) GetBank(ctx context.Context, req *connect.Request[economyv2.GetBankRequest]) (*connect.Response[economyv2.GetBankResponse], error) {
	bank, exists := s.econ.banks[req.Msg.BankId]
	if !exists {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("bank %d not found", req.Msg.BankId))
	}
	return connect.NewResponse(&economyv2.GetBankResponse{
		Bank: bank.GetBase(),
	}), nil
}

// UpdateBank 更新银行信息
func (s *Server) UpdateBank(ctx context.Context, req *connect.Request[economyv2.UpdateBankRequest]) (*connect.Response[economyv2.UpdateBankResponse], error) {
	if err := s.econ.UpdateBank(req.Msg.Bank); err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to update bank: %v", err))
	}
	return connect.NewResponse(&economyv2.UpdateBankResponse{}), nil
}

// ListBanks 列出所有银行
func (s *Server) ListBanks(ctx context.Context, req *connect.Request[economyv2.ListBanksRequest]) (*connect.Response[economyv2.ListBanksResponse], error) {
	var bankList []*economyv2.Bank
	for _, bank := range s.econ.banks {
		bankList = append(bankList, bank.GetBase())
	}
	return connect.NewResponse(&economyv2.ListBanksResponse{
		Banks: bankList,
	}), nil
}

// DeltaUpdateBank 增量更新银行
func (s *Server) DeltaUpdateBank(ctx context.Context, req *connect.Request[economyv2.DeltaUpdateBankRequest]) (*connect.Response[economyv2.DeltaUpdateBankResponse], error) {
	if err := s.econ.DeltaUpdateBank(
		req.Msg.BankId,
		req.Msg.DeltaInterestRate,
		req.Msg.DeltaCurrency,
		req.Msg.AddCitizenIds,
		req.Msg.RemoveCitizenIds,
	); err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to delta update bank: %v", err))
	}
	return connect.NewResponse(&economyv2.DeltaUpdateBankResponse{}), nil
}
