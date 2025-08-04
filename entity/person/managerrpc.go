package person

import (
	"context"
	"errors"
	"net/http"

	"connectrpc.com/connect"
	"git.fiblab.net/general/common/v2/parallel"
	"git.fiblab.net/sim/syncer/v3"

	personv2 "git.fiblab.net/sim/protos/v2/go/city/person/v2"
	"git.fiblab.net/sim/protos/v2/go/city/person/v2/personv2connect"
)

// Register 将Person管理器注册到Sidecar
// 功能：注册Person服务的RPC处理器到同步器
// 参数：sidecar-同步器实例
// 说明：使Person管理器能够通过RPC接口与外部系统通信
func (m *PersonManager) Register(sidecar *syncer.Sidecar) {
	sidecar.Register(
		personv2connect.PersonServiceName,
		func(opts ...connect.HandlerOption) (pattern string, handler http.Handler) {
			return personv2connect.NewPersonServiceHandler(m, opts...)
		},
	)
}

// personv2connect.PersonService

// GetPerson 获取person信息
// 功能：根据ID获取指定人员的信息
// 参数：ctx-上下文，in-请求参数（包含人员ID）
// 返回：人员信息响应，错误信息
// 算法说明：
// 1. 从请求中提取人员ID
// 2. 在管理器中查找对应的人员
// 3. 将人员信息转换为protobuf格式
// 4. 返回响应或错误
// 说明：提供人员信息的查询接口
func (m *PersonManager) GetPerson(ctx context.Context, in *connect.Request[personv2.GetPersonRequest]) (*connect.Response[personv2.GetPersonResponse], error) {
	req := in.Msg
	p, err := m.GetOrError(req.PersonId)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	res := &personv2.GetPersonResponse{
		Person: p.ToPersonRuntimePb(true),
	}
	return connect.NewResponse(res), nil
}

// AddPerson 新增person 传入person初始位置、目的地表、属性 返回personid
// 功能：创建新的人员并添加到仿真中
// 参数：ctx-上下文，in-请求参数（包含人员信息）
// 返回：人员ID响应，错误信息
// 算法说明：
// 1. 从请求中提取人员信息
// 2. 创建新的人员对象
// 3. 将人员添加到管理器中
// 4. 返回新人员的ID
// 说明：支持动态添加人员到仿真中
func (m *PersonManager) AddPerson(
	ctx context.Context, in *connect.Request[personv2.AddPersonRequest],
) (*connect.Response[personv2.AddPersonResponse], error) {
	req := in.Msg
	// FIXME: 添加检查
	p := m.add(req.Person)
	m.persons.Add(p)
	res := &personv2.AddPersonResponse{PersonId: p.ID()}
	return connect.NewResponse(res), nil
}

// SetSchedule 修改person的schedule 传入personid、目的地表
// 功能：修改指定人员的行程安排
// 参数：ctx-上下文，in-请求参数（包含人员ID和新的行程安排）
// 返回：操作结果响应，错误信息
// 算法说明：
// 1. 验证人员ID是否存在
// 2. 检查人员是否在路口内（路口内不支持修改行程）
// 3. 更新人员的行程安排
// 4. 返回操作结果
// 说明：支持动态修改人员的行程计划
func (m *PersonManager) SetSchedule(
	ctx context.Context, in *connect.Request[personv2.SetScheduleRequest],
) (*connect.Response[personv2.SetScheduleResponse], error) {
	req := in.Msg
	if p, ok := m.data[req.PersonId]; !ok {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("person id does not exist"))
	} else {
		if !(p.runtime.Lane != nil && p.runtime.Lane.ParentJunction() != nil) {
			// log.Infof("SetSchedule: %v, clock.T=%v", req, m.ctx.Clock().T)
			p.SetSchedules(req.Schedules)
		} else {
			return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("person in a junction dose support schedule setting"))
		}
	}
	return connect.NewResponse(&personv2.SetScheduleResponse{}), nil
}

// GetPersons 获取多个person信息
// 功能：批量获取人员信息，支持ID筛选和状态排除
// 参数：ctx-上下文，in-请求参数（包含人员ID列表和排除状态）
// 返回：人员信息列表响应，错误信息
// 算法说明：
// 1. 构建ID筛选集合和状态排除集合
// 2. 并行处理所有人员数据
// 3. 根据筛选条件过滤人员
// 4. 转换为protobuf格式并返回
// 说明：提供高效的人员信息批量查询接口
func (m *PersonManager) GetPersons(ctx context.Context, in *connect.Request[personv2.GetPersonsRequest]) (*connect.Response[personv2.GetPersonsResponse], error) {
	req := in.Msg
	personIdMap := map[int32]struct{}{}
	for _, id := range req.PersonIds {
		personIdMap[id] = struct{}{}
	}
	excludeStatusMap := map[personv2.Status]struct{}{}
	for _, status := range req.ExcludeStatuses {
		excludeStatusMap[status] = struct{}{}
	}
	res := &personv2.GetPersonsResponse{
		Persons: parallel.GoMapFilter(m.persons.Data(), func(p *Person) (*personv2.PersonRuntime, bool) {
			// 排除ID
			if len(personIdMap) > 0 {
				if _, ok := personIdMap[p.ID()]; !ok {
					return nil, false
				}
			}
			// 排除状态
			if _, ok := excludeStatusMap[p.Status()]; ok {
				return nil, false
			}
			return p.ToPersonRuntimePb(req.ReturnBase), true
		}),
	}
	return connect.NewResponse(res), nil
}

// ResetPersonPosition 重置person位置
// 功能：重置指定人员的位置信息
// 参数：ctx-上下文，in-请求参数（包含人员ID和新位置）
// 返回：操作结果响应，错误信息
// 算法说明：
// 1. 验证人员ID是否存在
// 2. 检查位置参数的有效性（不能同时存在多种位置类型）
// 3. 验证位置信息在地图中的有效性
// 4. 设置重置位置标记
// 说明：支持动态调整人员位置，仅适用于睡眠状态的人员
func (m *PersonManager) ResetPersonPosition(ctx context.Context, in *connect.Request[personv2.ResetPersonPositionRequest]) (*connect.Response[personv2.ResetPersonPositionResponse], error) {
	req := in.Msg
	p, ok := m.data[req.PersonId]
	if !ok {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("person id does not exist"))
	}
	pos := req.Position
	if pos.AoiPosition != nil && pos.LanePosition != nil {
		// 同时存在两个逻辑坐标
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("both aoi and lane position exist"))
	}
	if pos.AoiPosition == nil && pos.LanePosition == nil {
		// 不存在逻辑坐标
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("no position"))
	}
	if pos.AoiPosition != nil {
		_, err := m.ctx.AoiManager().GetOrError(pos.AoiPosition.AoiId)
		if err != nil {
			return nil, connect.NewError(connect.CodeInvalidArgument, err)
		}
	}
	if pos.LanePosition != nil {
		_, err := m.ctx.LaneManager().GetOrError(pos.LanePosition.LaneId)
		if err != nil {
			return nil, connect.NewError(connect.CodeInvalidArgument, err)
		}
	}
	if pos.LonglatPosition != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("longlat position is not supported"))
	}
	if p.Status() != personv2.Status_STATUS_SLEEP {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("person is not sleeping at aoi or lane, unsupported"))
	}
	p.resetPos = pos
	return connect.NewResponse(&personv2.ResetPersonPositionResponse{}), nil
}

// GetGlobalStatistics 获取全局统计信息
// 功能：获取全局统计信息
// 参数：ctx-上下文，in-请求参数
// 返回：全局统计信息响应，错误信息
// 算法说明：
// 1. 返回全局统计信息
// 说明：提供全局统计信息的查询接口
func (m *PersonManager) GetGlobalStatistics(ctx context.Context, in *connect.Request[personv2.GetGlobalStatisticsRequest]) (*connect.Response[personv2.GetGlobalStatisticsResponse], error) {
	res := &personv2.GetGlobalStatisticsResponse{
		NumCompletedTrips:          m.snapshot.NumCompletedTrips,
		RunningTotalTravelTime:     m.snapshot.TravelTime,
		RunningTotalTravelDistance: m.snapshot.TravelDistance,
	}
	return connect.NewResponse(res), nil
}
