package route

import (
	"sync"

	mapv2 "git.fiblab.net/sim/protos/v2/go/city/map/v2"
	routingv2 "git.fiblab.net/sim/protos/v2/go/city/routing/v2"
	"git.fiblab.net/sim/routing/v2/router"
)

// 本地导航服务
type LocalRouter struct {
	router *router.Router

	wg sync.WaitGroup
}

// 创建本地导航服务
func NewLocalRouter(
	mapData *mapv2.Map,
) *LocalRouter {
	r := &LocalRouter{
		router: router.New(mapData, nil),
	}
	return r
}

// 路径规划（回调版本）
func (l *LocalRouter) GetRoute(
	in *routingv2.GetRouteRequest,
	process func(res *routingv2.GetRouteResponse),
) chan struct{} {
	ch := make(chan struct{})
	l.wg.Add(1)
	go func() {
		defer l.wg.Done()
		// response
		res := &routingv2.GetRouteResponse{}
		// 请求处理
		start, end := in.Start, in.End
		// ATTENTION: 内联导航不再检查数据范围和格式
		switch in.GetType() {
		case routingv2.RouteType_ROUTE_TYPE_DRIVING, routingv2.RouteType_ROUTE_TYPE_TAXI:
			var journeyType routingv2.JourneyType
			if in.GetType() == routingv2.RouteType_ROUTE_TYPE_DRIVING {
				journeyType = routingv2.JourneyType_JOURNEY_TYPE_DRIVING
			} else if in.GetType() == routingv2.RouteType_ROUTE_TYPE_TAXI {
				journeyType = routingv2.JourneyType_JOURNEY_TYPE_BY_TAXI
			}
			if roadIDs, cost, err := l.router.SearchDriving(start, end, in.Time); err != nil {
				// log.Warnf("search driving failed from %v to %v at t=%f: %v", start, end, in.Time, err)
			} else {
				res.Journeys = append(res.Journeys, &routingv2.Journey{
					Type: journeyType,
					Driving: &routingv2.DrivingJourneyBody{
						RoadIds: roadIDs,
						Eta:     cost,
					},
				})
			}
		case routingv2.RouteType_ROUTE_TYPE_WALKING:
			if segments, cost, err := l.router.SearchWalking(start, end, in.Time); err != nil {
				log.Warnf("search walking failed from %v to %v at t=%f: %v", start, end, in.Time, err)
			} else {
				res.Journeys = append(res.Journeys, &routingv2.Journey{
					Type: routingv2.JourneyType_JOURNEY_TYPE_WALKING,
					Walking: &routingv2.WalkingJourneyBody{
						Route: segments,
						Eta:   cost,
					},
				})
			}
		case routingv2.RouteType_ROUTE_TYPE_BUS, routingv2.RouteType_ROUTE_TYPE_SUBWAY, routingv2.RouteType_ROUTE_TYPE_BUS_SUBWAY:
			var availableSublineTypes []mapv2.SublineType
			var ptType string
			if in.GetType() == routingv2.RouteType_ROUTE_TYPE_BUS {
				availableSublineTypes = []mapv2.SublineType{mapv2.SublineType_SUBLINE_TYPE_BUS}
				ptType = "bus"
			} else if in.GetType() == routingv2.RouteType_ROUTE_TYPE_SUBWAY {
				availableSublineTypes = []mapv2.SublineType{mapv2.SublineType_SUBLINE_TYPE_SUBWAY}
				ptType = "subway"
			} else if in.GetType() == routingv2.RouteType_ROUTE_TYPE_BUS_SUBWAY {
				availableSublineTypes = []mapv2.SublineType{mapv2.SublineType_SUBLINE_TYPE_BUS, mapv2.SublineType_SUBLINE_TYPE_SUBWAY}
				ptType = "bus, subway"
			}
			log.Debugf("Search %v route from %v to %v", ptType, start, end)
			if startWalkSegments, startWalkCost, transferSegment, transferCost, endWalkSegments, endWalkCost, err := l.router.SearchBus(start, end, in.Time, availableSublineTypes); err != nil {
				log.Warnf("search bus failed from %v to %v at t=%f: %v", start, end, in.Time, err)
			} else {
				// 步行去车站
				if startWalkSegments != nil {
					res.Journeys = append(res.Journeys, &routingv2.Journey{
						Type: routingv2.JourneyType_JOURNEY_TYPE_WALKING,
						Walking: &routingv2.WalkingJourneyBody{
							Route: startWalkSegments,
							Eta:   startWalkCost,
						},
					})
				}
				// 车站->车站
				if len(transferSegment) > 0 {
					res.Journeys = append(res.Journeys, &routingv2.Journey{
						Type: routingv2.JourneyType_JOURNEY_TYPE_BY_BUS,
						ByBus: &routingv2.BusJourneyBody{
							Transfers: transferSegment,
							Eta:       transferCost,
						},
					})
				}
				// 步行去车站
				if endWalkSegments != nil {
					res.Journeys = append(res.Journeys, &routingv2.Journey{
						Type: routingv2.JourneyType_JOURNEY_TYPE_WALKING,
						Walking: &routingv2.WalkingJourneyBody{
							Route: endWalkSegments,
							Eta:   endWalkCost,
						},
					})
				}
			}

		default:
			log.Panic("wrong routing type")
		}

		process(res)
		close(ch)
	}()
	return ch
}

// 路径规划（同步版本）
func (l *LocalRouter) GetRouteSync(in *routingv2.GetRouteRequest) *routingv2.GetRouteResponse {
	var res *routingv2.GetRouteResponse
	process := func(r *routingv2.GetRouteResponse) {
		res = r
	}
	<-l.GetRoute(in, process)
	return res
}
