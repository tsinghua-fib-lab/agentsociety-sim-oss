package route

import (
	"fmt"

	routingv2 "git.fiblab.net/sim/protos/v2/go/city/routing/v2"
	tripv2 "git.fiblab.net/sim/protos/v2/go/city/trip/v2"
	"git.fiblab.net/sim/simulet-go/entity"
	"github.com/samber/lo"
)

// 行人路径规划结果中的一段
type PedestrianSegment struct {
	Lane      entity.ILane
	Direction routingv2.MovingDirection
}

// 是否正向行驶
func (s PedestrianSegment) IsForward() bool {
	return s.Direction == routingv2.MovingDirection_MOVING_DIRECTION_FORWARD
}

// 行人路径规划
type PedestrianRoute struct {
	ctx entity.ITaskContext

	p entity.IPerson // 对应的人

	Start, End   entity.RoutePosition        // 导航起点
	base         *routingv2.GetRouteResponse // 导航请求结果
	waitCh       chan struct{}               // 路径规划请求等待通道
	ok           bool                        // 导航请求是否成功
	indexJourney int                         // 当前journey的索引
	indexRoute   int                         // 当前行驶车道对应的索引编号，即route[route_index_]==parent_lane
	route        []PedestrianSegment         // 转换为指针形式后的路径
}

// 创建行人路径规划
func NewPedestrianRoute(ctx entity.ITaskContext, p entity.IPerson) *PedestrianRoute {
	return &PedestrianRoute{
		ctx:    ctx,
		p:      p,
		waitCh: nil,
		route:  make([]PedestrianSegment, 0),
	}
}

// 输出路径规划信息以及当前位置前后的segment（总计3段）
func (r *PedestrianRoute) String() string {
	state := fmt.Sprintf("PedestrianRoute: ok=%v, indexJourney=%v, indexRoute=%v, start=(%v), end=(%v)", r.ok, r.indexJourney, r.indexRoute, r.Start, r.End)
	// 添加前一、当前、后一segment
	if r.indexRoute > 0 {
		state += fmt.Sprintf(", [-1]=%v", r.route[r.indexRoute-1])
	}
	state += fmt.Sprintf(", [0]=%v", r.route[r.indexRoute])
	if r.indexRoute+1 < len(r.route) {
		state += fmt.Sprintf(", [1]=%v", r.route[r.indexRoute+1])
	}
	return state
}

// 等待路径规划完成
func (r *PedestrianRoute) Wait() {
	if r.waitCh != nil {
		<-r.waitCh
		r.waitCh = nil
	}
}

// 清空路径规划
func (r *PedestrianRoute) Clear() {
	r.ok = false
}

// 是否有路径规划结果
func (r *PedestrianRoute) Ok() bool {
	return r.ok
}

// 是否到达终点
func (r *PedestrianRoute) AtLast() bool {
	return r.indexRoute+1 >= len(r.route)
}

// 获取当前路段
func (r *PedestrianRoute) Current() PedestrianSegment {
	return r.route[r.indexRoute]
}

// 获取下一路段
func (r *PedestrianRoute) Next() PedestrianSegment {
	return r.route[r.indexRoute+1]
}

// 获取最后一个路段
func (r *PedestrianRoute) Last() PedestrianSegment {
	return r.route[len(r.route)-1]
}

// 向前增加index，返回是否正常（true: 正常, false：越界）
func (r *PedestrianRoute) Step() bool {
	r.indexRoute++
	if r.indexRoute >= len(r.route) {
		r.indexRoute = len(r.route) - 1
		return false
	}
	return true
}

// 向导航服务请求路径规划
func (r *PedestrianRoute) ProduceRouting(trip *tripv2.Trip, startPosition entity.RoutePosition, routeType routingv2.RouteType) {
	target := trip.End
	r.Start = startPosition
	// 记录路径规划终点
	r.End = newRoutePosition(r.ctx, target)
	r.ok = false
	// 如果有预计算的路径规划结果，直接使用
	if len(trip.Routes) != 0 {
		r.ProcessRouting(&routingv2.GetRouteResponse{
			Journeys: trip.Routes,
		})
		r.waitCh = nil
		return
	}
	// 没有预计算的路径规划结果，发送请求
	req := &routingv2.GetRouteRequest{
		Type:  routeType,
		Start: newPbPosition(r.Start),
		End:   target,
		Time:  r.ctx.Clock().T,
	}
	// 发送路径规划请求
	r.waitCh = r.ctx.Router().GetRoute(req, r.ProcessRouting)
}
func (r *PedestrianRoute) RegisterWaitCallback(callback func()) {
	CallbackWaitGroup.Add(1)
	go func() {
		defer CallbackWaitGroup.Done()
		if r.waitCh != nil {
			<-r.waitCh
			r.waitCh = nil
		}
		callback()
	}()
}

// 处理路径规划结果
func (r *PedestrianRoute) ProcessRouting(res *routingv2.GetRouteResponse) {
	if len(res.Journeys) == 0 {
		r.route = make([]PedestrianSegment, 0)
		r.indexRoute = 0
		r.ok = false
		return
	}
	firstJourney := res.Journeys[0]
	if firstJourney.Type != routingv2.JourneyType_JOURNEY_TYPE_WALKING {
		log.Panicf("PedestrianRoute: unsupported journeyType %v", firstJourney.Type)
	}
	route := firstJourney.Walking.Route
	if len(route) == 0 {
		r.route = make([]PedestrianSegment, 0)
		r.indexRoute = 0
		r.ok = false
		return
	}
	// 根据导航结果推断补全起点和终点的内容
	if r.Start.Lane == nil {
		firstLane := route[0].LaneId
		laneStart := r.ctx.LaneManager().Get(firstLane)
		r.Start.Lane = laneStart
		r.Start.S = r.Start.Aoi.WalkingS(laneStart.ID())
	}

	r.base = res
	r.indexJourney = -1
	r.NextJourney(r.Start.Lane)
	r.ok = true
}

// 进入下一段行程
func (r *PedestrianRoute) NextJourney(lane entity.ILane) bool {
	if r.indexJourney+1 >= len(r.base.Journeys) {
		return false
	}
	r.indexJourney++
	r.route = make([]PedestrianSegment, 0)
	r.indexRoute = 0
	pb := r.base.Journeys[r.indexJourney]
	switch pb.Type {
	case routingv2.JourneyType_JOURNEY_TYPE_WALKING:
		pbRoute := pb.Walking.Route
		if lane.ID() != pbRoute[0].LaneId {
			log.Panic("PedestrianRoute: wrong start lane when processing")
		}
		r.route = lo.Map(pbRoute, func(pb *routingv2.WalkingRouteSegment, _ int) PedestrianSegment {
			lane := r.ctx.LaneManager().Get(pb.LaneId)
			return PedestrianSegment{lane, pb.MovingDirection}
		})
		startLane, endLane := r.route[0].Lane, r.route[len(r.route)-1].Lane
		if r.indexJourney+1 < len(r.base.Journeys) {
			log.Panic("PedestrianRoute: unsupported journeyType")
		} else {
			// ATTENTION: 只有一个journey，和开车一样，直接根据AOI+导航给出的终点Lane推断终点S
			if r.Start.Lane == nil {
				r.Start.Lane = startLane
				r.Start.S = r.Start.Aoi.WalkingS(startLane.ID())
			}
			if r.End.Lane == nil {
				r.End.Lane = endLane
				r.End.S = r.End.Aoi.WalkingS(endLane.ID())
			}
		}
	case routingv2.JourneyType_JOURNEY_TYPE_BY_BUS:
		log.Panic("PedestrianRoute: unsupported journeyType")
	default:
		log.Panic("PedestrianRoute: unsupported journeyType")
	}
	return true
}

// 处理输入的单个journey
func (r *PedestrianRoute) ProcessInputJourney(pb *routingv2.Journey, start, end entity.RoutePosition) bool {
	r.route = make([]PedestrianSegment, 0)
	r.waitCh = nil
	r.Start = start
	r.End = end
	r.indexRoute = 0
	switch pb.Type {
	case routingv2.JourneyType_JOURNEY_TYPE_WALKING:
		pbRoute := pb.Walking.Route
		r.route = lo.Map(pbRoute, func(pb *routingv2.WalkingRouteSegment, _ int) PedestrianSegment {
			lane := r.ctx.LaneManager().Get(pb.LaneId)
			return PedestrianSegment{lane, pb.MovingDirection}
		})
		startLane, endLane := r.route[0].Lane, r.route[len(r.route)-1].Lane
		if r.Start.Lane == nil {
			r.Start.Lane = startLane
			r.Start.S = r.Start.Aoi.WalkingS(startLane.ID())
		}
		if r.End.Lane == nil {
			r.End.Lane = endLane
			r.End.S = r.End.Aoi.WalkingS(endLane.ID())
		}
		r.ok = true
		r.base = &routingv2.GetRouteResponse{
			Journeys: []*routingv2.Journey{pb},
		}
	default:
		log.Panic("PedestrianRoute: unsupported journeyType")
	}
	return true
}

// 得到当前route的起始位置
func (r *PedestrianRoute) GetCurrentStartPosition() entity.RoutePosition {
	return r.Start
}

// 得到当前route的结束位置
func (r *PedestrianRoute) GetCurrentEndPosition() entity.RoutePosition {
	return r.End
}

// 将PedestrianRoute转为Protobuf格式
func (r *PedestrianRoute) ToPb() *routingv2.Journey {
	pb := &routingv2.Journey{
		Type: routingv2.JourneyType_JOURNEY_TYPE_WALKING,
		Walking: &routingv2.WalkingJourneyBody{
			Route: lo.Map(r.route, func(seg PedestrianSegment, _ int) *routingv2.WalkingRouteSegment {
				return &routingv2.WalkingRouteSegment{
					LaneId:          seg.Lane.ID(),
					MovingDirection: seg.Direction,
				}
			}),
			Eta: r.base.Journeys[r.indexJourney].Walking.Eta,
		},
	}
	return pb
}
