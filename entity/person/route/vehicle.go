package route

import (
	"fmt"
	"math"
	"sync"

	"git.fiblab.net/general/common/v2/mathutil"
	routingv2 "git.fiblab.net/sim/protos/v2/go/city/routing/v2"
	tripv2 "git.fiblab.net/sim/protos/v2/go/city/trip/v2"
	"git.fiblab.net/sim/simulet-go/entity"
	"github.com/samber/lo"
)

var CallbackWaitGroup sync.WaitGroup

const (
	viewDistanceFactor = 12   // 在一般情况下，观察距离应等于汽车在12秒内所通过的路程。如果车速为每小时60公里，则观察距离应为200米。
	minViewDistance    = 50   // 最小观察距离
	minLCDistance      = 10.0 // 最小强制变道距离
	maxLCDistance      = 30.0 // 最大强制变道距离
	lcFactor           = 3    // 强制变道时间比例参数
)

type JunctionCandidate struct {
	// Lanes和PreLanes一一对应，即PreLanes[i]是Lanes[i]的前驱
	// PreLanes按从左到右排列
	Junction        entity.IJunction // 路口
	Lanes           []entity.ILane   // 路口内的车道
	PreLanes        []entity.ILane   // 进入路口的车道
	hasTrafficLight bool             // 是否有信控
}

func (j JunctionCandidate) String() string {
	return fmt.Sprintf("JunctionCandidate{id: %v}", j.Junction.ID())
}

// 路径规划结果指针化，主要处理车辆
type VehicleRoute struct {
	ctx entity.ITaskContext

	p entity.IPerson // 对应的人

	Start, End entity.RoutePosition // 导航起点终点
	waitCh     chan struct{}        // 路径规划请求等待通道
	ok         bool                 // 导航请求是否成功

	// 路径的组成：start -> roads[0] -> juncLaneGroups[0] -> roads[1] -> ... -> roads[n-1] -> end
	// Vehicle对Route的使用方式：
	// 1. 如果是AtLast（len(roads) == 1 && len(juncLaneGroups) == 0），则需要变道到当前road最右边车道（或者不处理）
	// 2. 如果当前在road上，主动变道可以倾向于往juncLaneGroups[0]中标记的进入路口的车道集合中变道，
	// 强制变道则必须进入最近的进入路口的车道集合
	// 3. 如果在junction上，无
	// 4. 完成一条lane后，如果当前在road上，且lane不在juncLaneGroups[0]的进入路口的车道集合中，则传送到最近的对应车道
	// 5. 完成一条lane后，如果当前在junction上，则进入其后继

	AtRoad                 bool                // 当前导航在roads上
	Roads                  []entity.IRoad      // 路径中的所有road
	JuncLaneGroups         []JunctionCandidate // 路径中的所有路口（长度总是等于roads或者比roads少1）
	Eta                    float64             // 预计到达用时
	EtaFreeFlow            float64             // 预计到达用时（道路最高限速+路口不计算）
	EstimatedTotalDistance float64             // 估计的总行驶距离（米）
}

func NewVehicleRoute(ctx entity.ITaskContext, p entity.IPerson) *VehicleRoute {
	return &VehicleRoute{
		ctx:    ctx,
		p:      p,
		waitCh: nil,
	}
}

func (r VehicleRoute) String() string {
	return fmt.Sprintf("VehicleRoute<person=%d>{Start: %v, End: %v, AtRoad: %v, Roads: %v, JuncLaneGroups: %v, Eta: %v, EtaFreeFlow: %v}",
		r.p.ID(), r.Start, r.End, r.AtRoad, r.Roads, r.JuncLaneGroups, r.Eta, r.EtaFreeFlow)
}

func (r *VehicleRoute) Wait() {
	if r.waitCh != nil {
		<-r.waitCh
		r.waitCh = nil
	}
}

func (r *VehicleRoute) RegisterWaitCallback(callback func()) {
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

func (r *VehicleRoute) Clear() {
	r.ok = false
}

func (r *VehicleRoute) Ok() bool {
	return r.ok
}

// 根据指示的进入路口前的车道，找到"最适合"的junction lane
// 最适合：offset差距最小（可能不为0，即不为直行可达的）
func (r *VehicleRoute) GetJunctionLaneByPreLane(preLane entity.ILane, juncIndex int) (entity.ILane, int) {
	if juncIndex >= len(r.JuncLaneGroups) {
		return nil, 0
	}
	group := r.JuncLaneGroups[juncIndex]
	preLaneOffset := preLane.OffsetInRoad()
	minDelta := math.MaxInt
	var nearestLanes []entity.ILane
	for i, pre := range group.PreLanes {
		delta := mathutil.Abs(pre.OffsetInRoad() - preLaneOffset)
		if delta < minDelta {
			minDelta = delta
			nearestLanes = []entity.ILane{group.Lanes[i]}
		} else if delta == minDelta {
			nearestLanes = append(nearestLanes, group.Lanes[i])
		}
	}
	if len(nearestLanes) == 0 {
		log.Panicf("VehicleRoute: no nearest lane for preLane %v with juncIndex %d", preLane, juncIndex)
	}
	// 如果只有1个合适的，不用再考虑了
	if len(nearestLanes) == 1 {
		return nearestLanes[0], minDelta
	}
	// 如果有超过1个合适的，计算junction lane的后继lane再下一个路口的offset情况
	var bestLane entity.ILane
	minNextDelta := math.MaxInt
	for _, juncLane := range nearestLanes {
		nextPreLane, err := juncLane.UniqueSuccessor()
		if err != nil {
			log.Panicf("VehicleRoute: juncLane %v has no successor: err=%v", juncLane, err)
		}
		_, nextDelta := r.GetJunctionLaneByPreLane(nextPreLane, juncIndex+1)
		if nextDelta < minNextDelta {
			minNextDelta = nextDelta
			bestLane = juncLane
		}
	}
	return bestLane, minNextDelta
}

// 完成curLane行驶后的下一个车道
func (r *VehicleRoute) Next(curLane entity.ILane, curS float64, curV float64) entity.ILane {
	var nextLane entity.ILane
	if r.AtRoad {
		lc := r.GetLCScan(curLane, curS, curV)
		if len(r.JuncLaneGroups) == 0 {
			// 没有下一个车道，返回nil
			return nil
		}
		group := r.JuncLaneGroups[0]
		if !lc.InCandidate {
			// 需要变道
			if lc.Side == entity.LEFT {
				// 向左变道，最近的是最右边的车道
				nextLane = group.Lanes[len(group.Lanes)-1]
			} else {
				// 向右变道，最近的是最左边的车道
				nextLane = group.Lanes[0]
			}
		} else {
			nextLane, _ = r.GetJunctionLaneByPreLane(curLane, 0)
		}
		r.Roads = r.Roads[1:]
	} else {
		var err error
		nextLane, err = curLane.UniqueSuccessor()
		if err != nil {
			log.Panicf("VehicleRoute: lane %v has bad successor: err=%v, route=%v", curLane.ID(), err, r)
		}
		r.JuncLaneGroups = r.JuncLaneGroups[1:]
	}
	r.AtRoad = !r.AtRoad

	return nextLane
}

// 用于横向控制的辅助信息
type LC struct {
	InCandidate bool // 是否在通向下一路口的候选车道组中

	// InCandidate == true

	Neighbors [2]int // 左/右侧仍在候选车道组中的车道数

	// InCandidate == false

	Side            int     // 变道方向
	Count           int     // 需要变道的次数
	DeltaLCDistance float64 // 从需要变道的lane末端起算的不可变道长度 - 需要变道的lane末端到当前lane末端的距离
}

// 检查在road上，从curLane到最近的能够正确进入路口（即在juncLaneGroups[0].PreLanes中）的车道的距离
// 返回值：负数，需要向左变道；正数，需要向右变道；0，不变道；其绝对值为变道次数
func (r *VehicleRoute) GetLCScan(curLane entity.ILane, curS float64, curV float64) LC {
	if !r.AtRoad {
		log.Panic("VehicleRoute: not at road")
	}
	road := r.Roads[0]
	if curLane.ParentRoad() != road {
		log.Panicf("VehicleRoute: curLane %d is not in road %d. route=%v", curLane.ID(), road.ID(), r)
	}
	curLaneOffset := curLane.OffsetInRoad()
	if len(r.JuncLaneGroups) == 0 {
		// 没有下一个路口了，现在在最后一个road上，需要变道到最右侧的车道
		delta := r.End.Lane.OffsetInRoad() - curLaneOffset
		if delta == 0 {
			return LC{InCandidate: true, Neighbors: [2]int{0, 0}}
		} else if delta < 0 {
			return LC{InCandidate: false, Side: entity.LEFT, Count: -delta}
		} else {
			// delta > 0
			return LC{InCandidate: false, Side: entity.RIGHT, Count: delta}
		}
	}
	// 后续还有路口 向前一直探测
	// ATTENTION:现在计算探测距离使用的是最大限速而不是车的实际速度
	//viewDistance := math.Max(curV*VIEW_DISTANCE_FACTOR, MIN_VIEW_DISTANCE)
	viewDistance := math.Max(curLane.MaxV()*viewDistanceFactor, minViewDistance)
	scanDistance := curLane.Length() - curS // 已经向前探测的距离
	juncIndex := 0                          // 探测到的JuncLaneGroups下标
	scanJuncs := []JunctionCandidate{}
	scanLane := curLane // 当前观察到的路
	for scanDistance < viewDistance && juncIndex < len(r.JuncLaneGroups) {
		juncLane, _ := r.GetJunctionLaneByPreLane(scanLane, juncIndex)
		scanLane, _ = juncLane.UniqueSuccessor()
		scanDistance += juncLane.Length()
		scanDistance += scanLane.Length()
		scanJuncs = append(scanJuncs, r.JuncLaneGroups[juncIndex])
		juncIndex++
	}
	scanLane = curLane // 当前观察到的路
	lcLength := 0.0    // 从当前lane结尾到目标lane的开始的总距离
	for scanIndex, juncLaneGroup := range scanJuncs {
		// 没有信控的路口不检查
		if !juncLaneGroup.hasTrafficLight {
			juncLane, _ := r.GetJunctionLaneByPreLane(scanLane, scanIndex)
			scanLane, _ = juncLane.UniqueSuccessor()
			lcLength += juncLane.Length()
			lcLength += scanLane.Length()
			continue
		}
		scanLaneOffset := scanLane.OffsetInRoad()
		preLanes := juncLaneGroup.PreLanes
		leftPreLane := preLanes[0]
		rightPreLane := preLanes[len(preLanes)-1]
		leftPreLaneOffset := leftPreLane.OffsetInRoad()
		rightPreLaneOffset := rightPreLane.OffsetInRoad()
		deltaL := leftPreLaneOffset - scanLaneOffset
		deltaR := rightPreLaneOffset - scanLaneOffset
		if deltaL > 0 || deltaR < 0 {
			// 当前扫描到的scanLane不在PreLane范围内，需要变道
			// 变道目标选择需要退回curLane所在road
			preLanes := scanJuncs[scanIndex].PreLanes // 储存和变道目标连通并且和curLane同road的lanes
			for i := scanIndex - 1; i >= 0; i-- {
				preLaneMap := make(map[entity.ILane]struct{}) // 辅助map 帮助判断lane是否在PreLanes里
				for _, lane := range preLanes {
					preLaneMap[lane] = struct{}{}
				}
				preLanes = []entity.ILane{}
				for _, juncLane := range scanJuncs[i].Lanes {
					sucLane, _ := juncLane.UniqueSuccessor()
					if _, exists := preLaneMap[sucLane]; exists { // 说明这条juncLane连接到需要变道的目标
						preLane, _ := juncLane.UniquePredecessor()
						// 将相同road下的lane加入preLanes
						roadLanes := lo.Values(preLane.ParentRoad().Lanes())
						preLanes = append(preLanes, roadLanes...)
					}
				}
			}
			{
				minDelta := mathutil.INF // 最小变道距离
				lcSide := entity.LEFT
				lcCount := 0
				for _, preLane := range preLanes { // 遍历寻找代价最小的变道目标
					if math.Abs(float64(preLane.OffsetInRoad()-curLaneOffset)) < math.Abs(minDelta) {
						minDelta = float64(preLane.OffsetInRoad() - curLaneOffset)
						lcCount = int(math.Abs(minDelta))
					}

				}
				if lcCount > 0 { // 说明找到了合适的变道目标 由于preLanes可能是空的 这里不一定会变道
					if minDelta > 0 {
						lcSide = entity.RIGHT
					}
					forceLCDistance := lo.Clamp(curLane.MaxV()*lcFactor, minLCDistance, maxLCDistance)
					return LC{InCandidate: false, Side: lcSide, Count: lcCount, DeltaLCDistance: forceLCDistance - lcLength}
				}
			}
		}
		juncLane, _ := r.GetJunctionLaneByPreLane(scanLane, scanIndex)
		scanLane, _ = juncLane.UniqueSuccessor() // 不扫描路口内的lane
		lcLength += juncLane.Length()
		lcLength += scanLane.Length()
	}
	// 以下为不需要变道时返回
	curPreLanes := r.JuncLaneGroups[0].PreLanes
	leftPreLane := curPreLanes[0]
	rightPreLane := curPreLanes[len(curPreLanes)-1]
	leftPreLaneOffset := leftPreLane.OffsetInRoad()
	rightPreLaneOffset := rightPreLane.OffsetInRoad()
	return LC{InCandidate: true, Neighbors: [2]int{
		curLaneOffset - leftPreLaneOffset,
		rightPreLaneOffset - curLaneOffset,
	}}
}

// func (r *VehicleRoute) ProduceRouting(trip *tripv2.Trip, startPosition entity.RoutePosition, canUsePreroute bool) {
// 	target := trip.End
// 	r.Start = startPosition
// 	// 记录路径规划终点
// 	r.End = newRoutePosition(r.ctx, target)
// 	r.ok = false
// 	// 如果有预计算的路径规划结果，直接使用
// 	if canUsePreroute && len(trip.Routes) != 0 {
// 		r.ProcessRouting(&routingv2.GetRouteResponse{
// 			Journeys: trip.Routes,
// 		})
// 		r.waitCh = nil
// 		return
// 	}
// 	// 没有预计算的路径规划结果，发送请求
// 	req := &routingv2.GetRouteRequest{
// 		Start: newPbPosition(r.Start),
// 		End:   target,
// 		Type:  routingv2.RouteType_ROUTE_TYPE_DRIVING,
// 		Time:  r.ctx.Clock().T,
// 	}
// 	// 发送请求
// 	r.waitCh = r.ctx.Router().GetRoute(req, r.ProcessRouting)
// }

func (r *VehicleRoute) ProduceRoutingWithoutProcess(
	trip *tripv2.Trip,
	startPosition entity.RoutePosition,
	canUsePreroute bool,
) *routingv2.GetRouteResponse {
	target := trip.End
	r.Start = startPosition
	// 记录路径规划终点
	r.End = newRoutePosition(r.ctx, target)
	r.ok = false
	// 如果有预计算的路径规划结果，直接使用
	if canUsePreroute && len(trip.Routes) != 0 {
		return &routingv2.GetRouteResponse{
			Journeys: trip.Routes,
		}
	}
	// 没有预计算的路径规划结果，发送请求
	req := &routingv2.GetRouteRequest{
		Start: newPbPosition(r.Start),
		End:   target,
		Type:  routingv2.RouteType_ROUTE_TYPE_DRIVING,
		Time:  r.ctx.Clock().T,
	}
	// 发送请求
	return r.ctx.Router().GetRouteSync(req)
}

// 处理路径规划的共同逻辑
func (r *VehicleRoute) processJourneyCommon(roadIDs []int32, eta float64) {
	// 根据导航结果推断补全起点和终点的内容
	if r.Start.Lane == nil {
		roadID := roadIDs[0]
		r.Start.Lane = r.ctx.RoadManager().Get(roadID).RightestDrivingLane()
		laneID := r.Start.Lane.ID()
		r.Start.S = r.Start.Aoi.DrivingS(laneID)
	}
	if r.End.Lane == nil {
		roadID := roadIDs[len(roadIDs)-1]
		r.End.Lane = r.ctx.RoadManager().Get(roadID).RightestDrivingLane()
		laneID := r.End.Lane.ID()
		r.End.S = r.End.Aoi.DrivingS(laneID)
	}

	// roadIDs -> roads
	r.Roads = make([]entity.IRoad, len(roadIDs))
	for i, roadID := range roadIDs {
		r.Roads[i] = r.ctx.RoadManager().Get(roadID)
	}

	// -> junction lane group
	r.JuncLaneGroups = make([]JunctionCandidate, len(roadIDs)-1)
	for i := 0; i < len(roadIDs)-1; i++ {
		inRoad := r.Roads[i]
		outRoad := r.Roads[i+1]
		junc := inRoad.DrivingSuccessor()
		if junc == nil {
			log.Panicf("VehicleRoute: road %v has no successor", inRoad.ID())
		}
		lanes, _, _, ok := junc.DrivingLaneGroup(inRoad, outRoad)
		if !ok {
			log.Panicf("VehicleRoute: road %v and %v are not connected, please patch the map first", inRoad.ID(), outRoad.ID())
		}
		hasTrafficLight := true
		candidate := JunctionCandidate{
			Junction: junc,
			Lanes:    lanes,
			PreLanes: lo.Map(lanes, func(l entity.ILane, _ int) entity.ILane {
				pre, err := l.UniquePredecessor()
				if err != nil {
					log.Panicf("VehicleRoute: lane %v has no predecessor: err=%v", l.ID(), err)
				}
				if pre.ParentRoad() != inRoad {
					log.Panicf("VehicleRoute: road %v and %v are not the same", inRoad.ID(), pre.ParentRoad().ID())
				}
				return pre
			}),
			hasTrafficLight: hasTrafficLight,
		}
		r.JuncLaneGroups[i] = candidate
	}
	r.AtRoad = true
	r.ok = true
	r.Eta = eta
	// 预计到达用时（道路最高限速+路口不计算）
	r.EtaFreeFlow = 0
	r.EstimatedTotalDistance = 0
	// 1. 计算起点到第一个路口的时间
	road := r.Roads[0]
	d := road.GetAvgDrivingL() - r.Start.S
	r.EstimatedTotalDistance += d
	r.EtaFreeFlow += d / road.MaxV()
	// 2. 计算第一个路口到最后一个路口的时间
	for i := 0; i < len(r.JuncLaneGroups); i++ {
		road := r.Roads[i+1]
		d := road.GetAvgDrivingL()
		r.EstimatedTotalDistance += d
		r.EtaFreeFlow += d / road.MaxV()
	}
	// 3. 计算最后一个路口到终点的时间
	road = r.Roads[len(r.Roads)-1]
	d = r.End.S
	r.EstimatedTotalDistance += d
	r.EtaFreeFlow += d / road.MaxV()
}

// TODO: 存在两个重复的ProcessRouting相关函数
func (r *VehicleRoute) ProcessRouting(res *routingv2.GetRouteResponse) {
	if len(res.Journeys) == 0 {
		r.ok = false
		return
	}
	roadIDs := res.Journeys[0].Driving.RoadIds

	// res check
	if !(len(res.Journeys) == 1 &&
		res.Journeys[0].Type == *routingv2.JourneyType_JOURNEY_TYPE_DRIVING.Enum() &&
		res.Journeys[0].Driving != nil &&
		len(res.Journeys[0].Driving.RoadIds) > 0) {
		log.Panic("VehicleRoute: wrong res")
	}

	// 处理共同的路径规划逻辑
	r.processJourneyCommon(roadIDs, res.Journeys[0].Driving.Eta)

	// 如果最后一条road与r.End.Lane不匹配，报错
	if lastRoad := r.Roads[len(r.Roads)-1]; lastRoad != r.End.Lane.ParentRoad() {
		log.Panicf("VehicleRoute: last road %v in route result %v does not match end %v", lastRoad, res, r.End)
	}
}

// 将VehicleRoute的当前剩余路由转为Protobuf格式
func (r *VehicleRoute) ToPb() *routingv2.Journey {
	pb := &routingv2.Journey{
		Type: routingv2.JourneyType_JOURNEY_TYPE_DRIVING,
		Driving: &routingv2.DrivingJourneyBody{
			RoadIds: lo.Map(r.Roads, func(road entity.IRoad, _ int) int32 {
				return road.ID()
			}),
			Eta: r.Eta,
		},
	}
	return pb
}

// 处理输入的单个journey
func (r *VehicleRoute) ProcessInputJourney(pb *routingv2.Journey, start, end entity.RoutePosition) {
	if pb.Type != routingv2.JourneyType_JOURNEY_TYPE_DRIVING {
		log.Panic("VehicleRoute: unsupported journeyType")
	}
	r.waitCh = nil
	r.Start = start
	r.End = end
	roadIDs := pb.Driving.RoadIds

	// 处理共同的路径规划逻辑
	r.processJourneyCommon(roadIDs, pb.Driving.Eta)
}

// 得到当前route的起始位置
func (r *VehicleRoute) GetCurrentStartPosition() entity.RoutePosition {
	return r.Start
}

// 得到当前route的结束位置
func (r *VehicleRoute) GetCurrentEndPosition() entity.RoutePosition {
	return r.End
}
