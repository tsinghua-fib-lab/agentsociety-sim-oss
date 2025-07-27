package junction

import (
	"context"
	"errors"
	"net/http"

	"connectrpc.com/connect"
	mapv2 "git.fiblab.net/sim/protos/v2/go/city/map/v2"
	mapv2connect "git.fiblab.net/sim/protos/v2/go/city/map/v2/mapv2connect"
	"git.fiblab.net/sim/syncer/v3"
)

// Register 将Junction管理器注册到sidecar
// 功能：将Junction管理器注册为RPC服务，提供远程调用接口
// 参数：sidecar-同步器侧车实例
// 说明：注册信号灯服务处理器和路口服务处理器，支持gRPC-Connect协议
func (m *JunctionManager) Register(sidecar *syncer.Sidecar) {
	sidecar.Register(
		mapv2connect.TrafficLightServiceName,
		func(opts ...connect.HandlerOption) (pattern string, handler http.Handler) {
			return mapv2connect.NewTrafficLightServiceHandler(m, opts...)
		},
	)
	sidecar.Register(
		mapv2connect.JunctionServiceName,
		func(opts ...connect.HandlerOption) (pattern string, handler http.Handler) {
			return mapv2connect.NewJunctionServiceHandler(m, opts...)
		},
	)
}

// GetTrafficLight RPC接口：获取指定Junction的信号灯状态
// 功能：处理GetTrafficLight RPC请求，返回指定Junction的当前信号灯状态信息
// 参数：ctx-上下文，in-包含Junction ID的请求
// 返回：信号灯状态响应，包含当前程序、相位索引和剩余时间
// 说明：如果Junction不存在或没有信号灯则返回相应错误
func (m *JunctionManager) GetTrafficLight(
	ctx context.Context, in *connect.Request[mapv2.GetTrafficLightRequest],
) (*connect.Response[mapv2.GetTrafficLightResponse], error) {
	req := in.Msg
	j, ok := m.data[req.JunctionId]
	if !ok {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("junction id does not exist"))
	}
	if tl := j.trafficLight.Get(); tl == nil {
		return connect.NewResponse(&mapv2.GetTrafficLightResponse{}), nil
	} else {
		return connect.NewResponse(&mapv2.GetTrafficLightResponse{
			TrafficLight:  tl,
			PhaseIndex:    j.trafficLight.Step(),
			TimeRemaining: j.trafficLight.RemainingTime(),
		}), nil
	}
}

// SetTrafficLight RPC接口：设置指定Junction的信号灯程序
// 功能：处理SetTrafficLight RPC请求，为指定Junction设置新的信号灯程序
// 参数：ctx-上下文，in-包含信号灯程序和相位信息的请求
// 返回：设置结果响应
// 说明：支持设置完整的信号灯程序或取消信号灯程序（全绿灯）
func (m *JunctionManager) SetTrafficLight(
	ctx context.Context, in *connect.Request[mapv2.SetTrafficLightRequest],
) (*connect.Response[mapv2.SetTrafficLightResponse], error) {
	req := in.Msg
	j, ok := m.data[req.TrafficLight.JunctionId]
	if !ok {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("junction id does not exist"))
	}
	if len(req.TrafficLight.Phases) == 0 {
		if err := j.unsetTrafficLight(); err != nil {
			return nil, connect.NewError(connect.CodeInvalidArgument, err)
		}
		return connect.NewResponse(&mapv2.SetTrafficLightResponse{}), nil
	}
	if req.TimeRemaining < 0 {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("invalid remaining time"))
	}
	if err := j.SetTrafficLight(req.TrafficLight); err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	if err := j.setPhase(req.PhaseIndex, req.TimeRemaining); err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	return connect.NewResponse(&mapv2.SetTrafficLightResponse{}), nil
}

// SetTrafficLightPhase RPC接口：设置指定Junction的信号灯相位
// 功能：处理SetTrafficLightPhase RPC请求，设置指定Junction的当前相位和剩余时间
// 参数：ctx-上下文，in-包含Junction ID、相位索引和剩余时间的请求
// 返回：设置结果响应
// 说明：只修改相位状态，不改变信号灯程序
func (m *JunctionManager) SetTrafficLightPhase(
	ctx context.Context, in *connect.Request[mapv2.SetTrafficLightPhaseRequest],
) (*connect.Response[mapv2.SetTrafficLightPhaseResponse], error) {
	req := in.Msg
	j, ok := m.data[req.JunctionId]
	if !ok {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("junction id does not exist"))
	}
	if req.TimeRemaining < 0 {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("invalid remaining time"))
	}
	j.setPhase(req.PhaseIndex, req.TimeRemaining)
	return connect.NewResponse(&mapv2.SetTrafficLightPhaseResponse{}), nil
}

// SetTrafficLightStatus RPC接口：设置指定Junction的信号灯状态
// 功能：处理SetTrafficLightStatus RPC请求，设置指定Junction的信号灯开关状态
// 参数：ctx-上下文，in-包含Junction ID和状态标志的请求
// 返回：设置结果响应
// 说明：true表示正常工作，false表示失效（全绿灯）
func (m *JunctionManager) SetTrafficLightStatus(
	ctx context.Context, in *connect.Request[mapv2.SetTrafficLightStatusRequest],
) (*connect.Response[mapv2.SetTrafficLightStatusResponse], error) {
	req := in.Msg
	j, ok := m.data[req.JunctionId]
	if !ok {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("junction id does not exist"))
	}
	if err := j.setStatus(req.Ok); err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	return connect.NewResponse(&mapv2.SetTrafficLightStatusResponse{}), nil
}
