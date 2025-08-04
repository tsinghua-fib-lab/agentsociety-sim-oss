package route

import (
	routingv2 "git.fiblab.net/sim/protos/v2/go/city/routing/v2"
	tripv2 "git.fiblab.net/sim/protos/v2/go/city/trip/v2"
	"github.com/samber/lo"
	"github.com/tsinghua-fib-lab/agentsociety-sim-oss/entity"
)

// 当前route的类型
type MultiModalType int32

const (
	MultiModalType_WALK  MultiModalType = iota // 步行
	MultiModalType_DRIVE                       // 开车
)

type MultiModalRoute struct {
	ctx             entity.ITaskContext
	p               entity.IPerson
	Start, End      entity.RoutePosition        // 导航起点
	base            *routingv2.GetRouteResponse // 导航请求结果，可能包含多段Journey
	waitCh          chan struct{}               // 路径规划请求等待通道
	ok              bool                        // 导航有效指示位
	MultiModalType  MultiModalType              // 当前导航的类型
	VehicleRoute    *VehicleRoute               // 车辆导航
	PedestrianRoute *PedestrianRoute            // 行人导航
	indexJourney    int                         // 当前journey下标 假设步行和开车都只有一个journey
	ForceEnd        bool                        // 强制结束此段导航 person瞬移到route终点
}

// 创建多式联运路径规划
func NewMultiModalRoute(ctx entity.ITaskContext, p entity.IPerson) *MultiModalRoute {
	return &MultiModalRoute{
		ctx:             ctx,
		p:               p,
		waitCh:          nil,
		PedestrianRoute: NewPedestrianRoute(ctx, p),
		VehicleRoute:    NewVehicleRoute(ctx, p),
		ForceEnd:        false,
	}
}

// 检查是否有效起点
func (r *MultiModalRoute) isValidPreRoute(trip *tripv2.Trip, startPosition entity.RoutePosition) bool {
	if len(trip.Routes) == 0 {
		return false
	}
	journey := trip.Routes[0]
	switch journey.Type {
	case routingv2.JourneyType_JOURNEY_TYPE_WALKING:
		if len(journey.Walking.Route) == 0 {
			return false
		} else {
			firstLaneID := journey.Walking.Route[0].LaneId
			if aoi := startPosition.Aoi; aoi != nil {
				if _, ok := aoi.WalkingLanes()[firstLaneID]; !ok {
					return false
				}
			}
			if lane := startPosition.Lane; lane != nil {
				if lane.ID() != firstLaneID {
					return false
				}
			}
		}
	case routingv2.JourneyType_JOURNEY_TYPE_DRIVING:
		if len(journey.Driving.RoadIds) == 0 {
			return false
		} else {
			firstRoadID := journey.Driving.RoadIds[0]
			firstRoad := r.ctx.RoadManager().Get(firstRoadID)
			rightestDrivingLane := firstRoad.RightestDrivingLane()
			if aoi := startPosition.Aoi; aoi != nil {
				if _, ok := firstRoad.Lanes()[rightestDrivingLane.ID()]; !ok {
					return false
				}
			}
			if lane := startPosition.Lane; lane != nil {
				if _, ok := firstRoad.Lanes()[lane.ID()]; !ok {
					return false
				}
			}
		}
	default:
		log.Warning("MultiModalRoute: unsupported journeyType")
		return false
	}
	return true
}

// 向导航服务请求路径规划
func (r *MultiModalRoute) ProduceRouting(trip *tripv2.Trip, startPosition entity.RoutePosition, routeType routingv2.RouteType) {
	target := trip.End
	r.Start = startPosition
	// 记录路径规划终点
	r.End = newRoutePosition(r.ctx, target)
	r.ok = false
	// 如果有预计算的路径规划结果，直接使用
	if r.isValidPreRoute(trip, startPosition) {
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

func (r *MultiModalRoute) ProcessRouting(res *routingv2.GetRouteResponse) {
	// 预处理res，移除无效的journey
	// 无效的journey：route长度为0
	res.Journeys = lo.Filter(res.Journeys, func(journey *routingv2.Journey, _ int) bool {
		switch journey.Type {
		case routingv2.JourneyType_JOURNEY_TYPE_WALKING:
			if len(journey.Walking.Route) == 0 {
				log.Warnf("MultiModalRoute: walking journey with empty route, personID=%v, routeResponse=%v", r.p.ID(), res)
				return false
			}
			return true
		case routingv2.JourneyType_JOURNEY_TYPE_DRIVING:
			if len(journey.Driving.RoadIds) == 0 {
				log.Warnf("MultiModalRoute: driving journey with empty roadIds, personID=%v, routeResponse=%v", r.p.ID(), res)
				return false
			}
			return true
		default:
			log.Panic("MultiModalRoute: unsupported journeyType")
			return false
		}
	})

	if len(res.Journeys) == 0 {
		r.ok = false
		return
	}
	r.base = res
	r.indexJourney = 0
	r.ok = true
	r.ForceEnd = false
	firstJourney := r.base.Journeys[0]
	switch firstJourney.Type {
	case routingv2.JourneyType_JOURNEY_TYPE_WALKING:
		r.MultiModalType = MultiModalType_WALK
		routeEnd := r.End
		r.PedestrianRoute.ProcessInputJourney(firstJourney, r.Start, routeEnd)
	case routingv2.JourneyType_JOURNEY_TYPE_DRIVING:
		r.MultiModalType = MultiModalType_DRIVE
		r.VehicleRoute.ProcessInputJourney(firstJourney, r.Start, r.End)
	default:
		log.Panic("MultiModalRoute: unsupported journeyType")
	}

}

func (r *MultiModalRoute) GetCurrentStartPosition() entity.RoutePosition {
	var curPosition entity.RoutePosition
	switch r.MultiModalType {
	case MultiModalType_DRIVE:
		curPosition = r.VehicleRoute.GetCurrentStartPosition()
	case MultiModalType_WALK:
		curPosition = r.PedestrianRoute.GetCurrentStartPosition()
	default:
		log.Panic("MultiModalRoute: invalid MultiModalType")
	}
	return curPosition
}

func (r *MultiModalRoute) GetCurrentEndPosition() entity.RoutePosition {
	var curPosition entity.RoutePosition
	switch r.MultiModalType {
	case MultiModalType_DRIVE:
		curPosition = r.VehicleRoute.GetCurrentEndPosition()
	case MultiModalType_WALK:
		curPosition = r.PedestrianRoute.GetCurrentEndPosition()
	default:
		log.Panic("MultiModalRoute: invalid MultiModalType")
	}
	return curPosition
}

// 等待路径规划完成
func (r *MultiModalRoute) Wait() {
	r.VehicleRoute.Wait()
	r.PedestrianRoute.Wait()
	if r.waitCh != nil {
		<-r.waitCh
		r.waitCh = nil
	}
}

// 清空路径规划
func (r *MultiModalRoute) Clear() {
	r.VehicleRoute.Clear()
	r.PedestrianRoute.Clear()
	r.ok = false
}

// 是否有路径规划结果
func (r *MultiModalRoute) Ok() bool {
	return r.ok
}
func (r *MultiModalRoute) RegisterWaitCallback(callback func()) {
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
