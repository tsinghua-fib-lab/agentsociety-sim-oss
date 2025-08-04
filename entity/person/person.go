package person

import (
	"fmt"
	"math"

	"git.fiblab.net/general/common/v2/geometry"
	"git.fiblab.net/general/common/v2/protoutil"
	geov2 "git.fiblab.net/sim/protos/v2/go/city/geo/v2"
	mapv2 "git.fiblab.net/sim/protos/v2/go/city/map/v2"
	personv2 "git.fiblab.net/sim/protos/v2/go/city/person/v2"
	routingv2 "git.fiblab.net/sim/protos/v2/go/city/routing/v2"
	tripv2 "git.fiblab.net/sim/protos/v2/go/city/trip/v2"
	"github.com/samber/lo"
	"github.com/tsinghua-fib-lab/moss-agentsociety-go/entity"
	"github.com/tsinghua-fib-lab/moss-agentsociety-go/entity/person/route"
	"github.com/tsinghua-fib-lab/moss-agentsociety-go/entity/person/schedule"
	"github.com/tsinghua-fib-lab/moss-agentsociety-go/utils/container"
	"github.com/tsinghua-fib-lab/moss-agentsociety-go/utils/randengine"
)

const (
	maxVehicleVNoise           = 5  // 车辆速度随机扰动最大值
	maxVehicleANoise           = .5 // 车辆加速度随机扰动最大值s
	maxPedestrianPositionNoise = 2  // 行人位置输出随机扰动最大值
)

// Person 人员实体
// 功能：表示模拟系统中的所有人员，包括行人、驾驶员、乘客等，支持多种交通方式和状态管理
type Person struct {
	container.IncrementalItemBase
	ctx entity.ITaskContext
	m   *PersonManager

	// 静态属性
	base           *personv2.Person
	id             int32
	attr           *personv2.PersonAttribute     // 人的属性
	vehicleAttr    *personv2.VehicleAttribute    // 车的属性
	pedestrianAttr *personv2.PedestrianAttribute // 行人的属性
	busAttr        *personv2.BusAttribute        // 公交车的属性
	bikeAttr       *personv2.BikeAttribute       // 自行车的属性
	home           *geov2.Position               // 人的家庭位置
	labels         map[string]string             // 人的标签

	generator *randengine.Engine // 随机数生成器，以ID为seed

	// 运行时基本数据，记录位置、速度、方向、状态
	runtime  runtime // 运行时数据
	snapshot runtime // 快照

	vehicle    *vehicle    // 车辆
	pedestrian *pedestrian // 行人

	// 时刻表
	schedule          *schedule.Schedule // 时刻表
	newSchedule       []*tripv2.Schedule // schedule修改buffer
	scheduleResetFlag bool               // 时刻表是否被修改

	// 导航
	multiModalRoute *route.MultiModalRoute // 多式联运导航

	// 重置位置（目前仅支持从Sleep重置）
	resetPos *geov2.Position
}

// newPerson 创建并初始化一个新的Person实例
// 功能：根据基础数据创建Person对象，初始化各种属性和组件
// 参数：ctx-任务上下文，m-人员管理器，base-基础Person数据
// 返回：初始化完成的Person实例
// 说明：根据人员类型初始化不同的交通组件，设置随机数生成器，验证车辆属性
func newPerson(
	ctx entity.ITaskContext,
	m *PersonManager,
	base *personv2.Person,
) *Person {
	p := &Person{
		ctx:            ctx,
		m:              m,
		base:           base,
		id:             base.Id,
		attr:           base.Attribute,
		vehicleAttr:    base.VehicleAttribute,
		pedestrianAttr: base.PedestrianAttribute,
		busAttr:        base.BusAttribute,
		bikeAttr:       base.BikeAttribute,
		home:           base.Home,
		labels:         base.Labels,
		runtime: runtime{
			Status:    personv2.Status_STATUS_SLEEP,
			IsTripEnd: true,
		},
		schedule:    schedule.NewSchedule(ctx, base.GetSchedules()),
		newSchedule: make([]*tripv2.Schedule, 0),
		generator:   randengine.New(uint64(base.Id)),
	}
	// // DEBUG
	// p.vehicleAttr.Length = 15
	p.multiModalRoute = route.NewMultiModalRoute(ctx, p)
	p.SetSchedules(base.GetSchedules())
	// 属性检查
	if p.vehicleAttr.MaxSpeed <= 0 {
		log.Fatalf("person %d (vehicle_attr=%v) vehicle max speed is less than 0, please check the data", p.ID(), p.vehicleAttr)
	}
	if p.vehicleAttr.MaxAcceleration <= 0 {
		log.Fatalf("person %d (vehicle_attr=%v) vehicle max acceleration is less than 0, please check the data", p.ID(), p.vehicleAttr)
	}
	if p.vehicleAttr.MaxBrakingAcceleration >= 0 {
		log.Fatalf("person %d (vehicle_attr=%v) vehicle max braking acceleration is greater than 0, please check the data", p.ID(), p.vehicleAttr)
	}
	if p.vehicleAttr.UsualAcceleration <= 0 {
		log.Fatalf("person %d (vehicle_attr=%v) vehicle usual acceleration is less than 0, please check the data", p.ID(), p.vehicleAttr)
	}
	if p.vehicleAttr.UsualBrakingAcceleration >= 0 {
		log.Fatalf("person %d (vehicle_attr=%v) vehicle usual braking acceleration is greater than 0, please check the data", p.ID(), p.vehicleAttr)
	}
	if p.vehicleAttr.Length <= 0 {
		log.Fatalf("person %d (vehicle_attr=%v) vehicle length is less than 0, please check the data", p.ID(), p.vehicleAttr)
	}
	if p.vehicleAttr.Width <= 0 {
		log.Fatalf("person %d (vehicle_attr=%v) vehicle width is less than 0, please check the data", p.ID(), p.vehicleAttr)
	}
	if p.vehicleAttr.MinGap < 0 {
		log.Fatalf("person %d (vehicle_attr=%v) vehicle min gap is less than 0, please check the data", p.ID(), p.vehicleAttr)
	}
	if p.vehicleAttr.Headway < 0 {
		log.Fatalf("person %d (vehicle_attr=%v) vehicle headway is less than 0, please check the data", p.ID(), p.vehicleAttr)
	}
	// 为车辆属性添加随机扰动
	// 最大速度
	p.vehicleAttr.MaxSpeed = math.Max(p.vehicleAttr.MaxSpeed+
		maxVehicleVNoise*lo.Clamp(.5*p.generator.NormFloat64(), -1, 1),
		.1)
	// 最大刹车加速度
	p.vehicleAttr.MaxBrakingAcceleration = math.Min(p.vehicleAttr.MaxBrakingAcceleration+
		maxVehicleANoise*lo.Clamp(.5*p.generator.NormFloat64(), -1, 1),
		-.1)
	p.vehicle = &vehicle{
		length: p.vehicleAttr.Length,
	}
	p.vehicle.controller = newController(p)
	walkV := defaultWalkV
	if base.PedestrianAttribute != nil {
		walkV = base.PedestrianAttribute.Speed
	}
	walkV += maxVNoise * lo.Clamp(.5*p.generator.NormFloat64(), -1, 1)
	walkV = math.Max(minWalkV, walkV)
	bikeV := defaultBikeV
	if base.BikeAttribute != nil {
		bikeV = base.BikeAttribute.Speed
	}
	bikeV += maxVNoise * lo.Clamp(.5*p.generator.NormFloat64(), -1, 1)
	bikeV = math.Max(minBikeV, bikeV)
	p.pedestrian = &pedestrian{
		walkingV:           walkV,
		bikingV:            bikeV,
		verticalOffsetRate: p.generator.Float64(),
		horizontalOffset: lo.Clamp(
			p.generator.NormFloat64(),
			-maxPedestrianPositionNoise,
			maxPedestrianPositionNoise,
		),
	}
	// 设置人的初始位置
	home := base.Home
	if home.AoiPosition != nil {
		aoiID := home.AoiPosition.AoiId
		aoi := p.ctx.AoiManager().Get(aoiID)
		p.runtime.Aoi = aoi
		p.runtime.XYZ = aoi.Centroid()
		aoi.AddPerson(p)
	} else if home.LanePosition != nil {
		laneID := home.LanePosition.LaneId
		s := home.LanePosition.S
		lane := p.ctx.LaneManager().Get(laneID)
		p.runtime.Lane = lane
		p.runtime.S = s
		p.runtime.XYZ = lane.GetPositionByS(s)
	} else {
		log.Panicf("person %d has no home position", p.ID())
	}
	return p
}

func (p *Person) prepareNode() {
	switch p.runtime.Status {
	case personv2.Status_STATUS_DRIVING:
		// 完成计算，清空支链
		p.vehicle.node.Extra.Clear()
		if p.snapshot.LC.IsLC {
			p.vehicle.shadowNode.Extra.Clear()
		}
		// key值更新
		p.vehicle.node.S = p.runtime.S
		if p.runtime.LC.IsLC {
			p.vehicle.shadowNode.S = p.runtime.LC.ShadowS
		}
	case personv2.Status_STATUS_WALKING:
		p.pedestrian.node.S = p.runtime.S
	case personv2.Status_STATUS_PASSENGER:
		// p.runtime.submodule.PrepareNode()
	}
}

// prepare 准备阶段，处理Person的准备工作
// 功能：更新快照数据，处理乘客缓冲区，更新事件序列，重置时刻表
// 说明：使用缓冲区机制提高并发性能，避免在更新阶段进行写操作
func (p *Person) prepare() {
	p.snapshot = p.runtime
	switch p.runtime.Status {
	case personv2.Status_STATUS_DRIVING:
		p.runtime.Action = Action{}
	}
	// 优先执行新的schedule
	p.ResetScheduleIfNeed()
}

// update 更新阶段，执行Person的模拟逻辑
// 功能：根据人员状态执行相应的更新逻辑，包括驾驶、步行、乘客、睡眠等状态
// 参数：dt-时间步长，各种控制通道用于不同模块间的通信
// 说明：根据人员类型和状态分发到不同的更新逻辑，支持出租车、地铁等特殊处理
func (p *Person) update(
	dt float64,
) {
	// 对resetPos的预检查
	if p.resetPos != nil {
		if p.runtime.Status != personv2.Status_STATUS_SLEEP {
			log.Errorf("person %d reset position %v not in sleep status", p.ID(), p.resetPos)
			p.resetPos = nil
		}
	}
	switch p.runtime.Status {
	case personv2.Status_STATUS_SLEEP:
		if p.resetPos != nil {
			log.Debugf("person %d reset position to %v", p.ID(), p.resetPos)
			// 由于限定是SLEEP状态，所以肯定不会isCrowd
			if p.runtime.Aoi != nil {
				p.runtime.Aoi.RemovePerson(p)
			}
			p.runtime.resetByPbPosition(p.ctx, p.resetPos)
			// 给Reset到的Aoi或Lane添加人
			if p.runtime.Aoi != nil {
				p.runtime.Aoi.AddPerson(p)
			} else {
				// 必须是Lane
				// do nothing
				var _ struct{}
			}
			p.resetPos = nil
		}
		// ATTENTION:一段trip的多个journey之间切换过程中必定满足出发时间触发
		if p.checkDeparture() {
			// 出发
			p.requestRoute()
			p.runtime.Status = personv2.Status_STATUS_WAIT_ROUTE
			return
		}
	case personv2.Status_STATUS_WAIT_ROUTE:
		if _, ok := p.routeSuccessful(); !ok {
			p.runtime.Status = personv2.Status_STATUS_SLEEP
			return
		}
		p.updateGoOut()
	case personv2.Status_STATUS_WALKING:
		isEnd := p.updatePedestrian(dt)
		p.runtime.IsTripEnd = isEnd
		if isEnd {
			end := p.multiModalRoute.GetCurrentEndPosition()
			// 行人结束路面行为（生命周期结束）的后处理
			// 步行和开车都只有单个journey
			// 本行程走完，进入sleep
			endAoi := end.Aoi
			p.schedule.NextTrip(p.ctx.Clock().T)
			if endAoi != nil {
				p.updateComeIn(endAoi, end.XY)
			} else {
				p.runtime.Status = personv2.Status_STATUS_SLEEP
			}
		}
	case personv2.Status_STATUS_DRIVING:
		isEnd := p.updateVehicle(dt)
		p.runtime.IsTripEnd = isEnd
		if isEnd {
			end := p.multiModalRoute.GetCurrentEndPosition()
			p.schedule.NextTrip(p.ctx.Clock().T)
			if end.Aoi != nil {
				p.updateComeIn(end.Aoi, end.XY)
			} else {
				p.runtime.Status = personv2.Status_STATUS_SLEEP
			}
		}
	default:
		log.Panicf("unknown person %d status %v when update", p.ID(), p.runtime.Status)
	}
}

// 从室内出来的辅助函数
func (p *Person) updateGoOut() {
	switch p.multiModalRoute.MultiModalType {
	case route.MultiModalType_DRIVE:
		// 导航成功，出发
		p.runtime.Status = personv2.Status_STATUS_DRIVING
		// 修改位置到门口
		p.runtime.Lane = p.multiModalRoute.GetCurrentStartPosition().Lane
		p.runtime.S = p.multiModalRoute.GetCurrentStartPosition().S
		p.runtime.clearLaneChange()
		if p.runtime.Aoi != nil {
			p.runtime.Aoi.RemovePerson(p)
			p.runtime.Aoi = nil
		}
		// 更新xy坐标
		p.runtime.XYZ = p.runtime.Lane.GetPositionByS(p.runtime.S)
		if p.snapshot.Lane == nil || (p.vehicle.node == nil && p.vehicle.shadowNode == nil) {
			// 当前不在路上，直接初始化
			p.vehicle.node = newVehicleNode(p.runtime.S, p)
			p.vehicle.shadowNode = newVehicleNode(p.runtime.S, p)
			p.runtime.Lane.AddVehicle(p.vehicle.node)
		} else {
			// 当前在路上，需要采用更新的方式
			if p.vehicle.node.Parent() == nil {
				p.runtime.Lane.AddVehicle(p.vehicle.node)
			} else {
				p.updateLaneVehicleNodes(true)
			}
		}

	case route.MultiModalType_WALK:
		// 导航成功，出发
		p.runtime.Status = personv2.Status_STATUS_WALKING
		// 修改位置到门口
		p.runtime.Lane = p.multiModalRoute.GetCurrentStartPosition().Lane
		p.runtime.S = p.multiModalRoute.GetCurrentStartPosition().S
		if p.runtime.Aoi != nil {
			p.runtime.Aoi.RemovePerson(p)
			p.runtime.Aoi = nil
		}
		// 更新xy坐标
		p.runtime.XYZ = p.runtime.Lane.GetPositionByS(p.runtime.S)
		p.pedestrian.node = newPedestrianNode(p.runtime.S, p)
		p.runtime.Lane.AddPedestrian(p.pedestrian.node)
	default:
		log.Panicf("Bad multiModal type: %v", p.multiModalRoute.MultiModalType)
	}
}

// 进入室内的辅助函数
func (p *Person) updateComeIn(endAoi entity.IAoi, endXyOrNil *geometry.Point) {
	p.runtime.Aoi = endAoi
	endAoi.AddPerson(p)
	p.runtime.XYZ = endAoi.Centroid()
	p.runtime.Status = personv2.Status_STATUS_SLEEP
	p.runtime.Lane = nil
	p.runtime.S = 0
}

// 获取人的ID
func (p *Person) ID() int32 {
	if p == nil {
		return -1
	}
	return p.id
}

// 获取人的属性
func (p *Person) Attr() *personv2.PersonAttribute {
	return p.attr
}

// 获取人开车时的车辆属性
func (p *Person) VehicleAttr() *personv2.VehicleAttribute {
	return p.vehicleAttr
}

// 获取人作为公交车司机时的公交车属性
func (p *Person) BusAttr() *personv2.BusAttribute {
	return p.busAttr
}

// 获取人骑自行车时的自行车属性
func (p *Person) BikeAttr() *personv2.BikeAttribute {
	return p.bikeAttr
}

// 获取人的位置坐标
func (p *Person) XYZ() geometry.Point {
	return p.snapshot.XYZ
}

// 获取人的速度
func (p *Person) V() float64 {
	return p.snapshot.V
}

// 获取人在当前状态下的长度（开车->车长）
func (p *Person) Length() float64 {
	if p.snapshot.Status == personv2.Status_STATUS_DRIVING {
		return p.vehicle.length
	} else {
		return 0
	}
}

// 获取人的空间父对象ID
func (p *Person) ParentID() int32 {
	switch p.snapshot.Status {
	case personv2.Status_STATUS_SLEEP,
		personv2.Status_STATUS_WAIT_ROUTE:
		return p.snapshot.Aoi.ID()
	case personv2.Status_STATUS_DRIVING,
		personv2.Status_STATUS_WALKING:
		return p.snapshot.Lane.ID()
	}
	log.Panicf("unknown person %d status %v", p.ID(), p.snapshot.Status)
	return -1
}

// 获取人所在的Aoi
func (p *Person) Aoi() entity.IAoi {
	return p.snapshot.Aoi
}

// 获取人所在的Lane
func (p *Person) Lane() entity.ILane {
	return p.snapshot.Lane
}

// 获取人在Lane上的位置S坐标
func (p *Person) S() float64 {
	return p.snapshot.S
}

// 获取人的状态
func (p *Person) Status() personv2.Status {
	return p.snapshot.Status
}

// 获取指定键的标签值
func (p *Person) GetLabel(key string) (string, bool) {
	value, ok := p.labels[key]
	return value, ok
}

// 设置时刻表
func (p *Person) SetSchedules(schedules []*tripv2.Schedule) {
	p.newSchedule = schedules
	p.scheduleResetFlag = true
}

func (p *Person) ResetScheduleIfNeed() {
	if p.scheduleResetFlag {
		p.schedule.Set(p.newSchedule, p.ctx.Clock().T)
		p.scheduleResetFlag = false
		// 强制转为Sleep模式，便于触发新的schedule
		p.runtime.Status = personv2.Status_STATUS_SLEEP
		// 清空导航
		p.multiModalRoute.Clear()
	}
}

// 更新时刻表，进入下一次出行，返回是否成功（是否有下一次出行）
func (p *Person) nextTrip() bool {
	return p.schedule.NextTrip(p.ctx.Clock().T)
}
func (p *Person) tripRouteEnd() entity.RoutePosition {
	trip := p.schedule.GetTrip()
	var routePos entity.RoutePosition
	if trip == nil {
		return routePos
	}
	if tripEnd := trip.End; tripEnd != nil {
		if tripEndLanePos := tripEnd.LanePosition; tripEndLanePos != nil {
			routePos.Lane = p.ctx.LaneManager().Get(tripEndLanePos.LaneId)
			routePos.S = tripEndLanePos.S
		}
		if tripEndAoiPos := tripEnd.AoiPosition; tripEndAoiPos != nil {
			routePos.Aoi = p.ctx.AoiManager().Get(tripEndAoiPos.AoiId)
		}
	}
	return routePos
}

// 检查是否到达出发时间
func (p *Person) checkDeparture() bool {
	return p.ctx.Clock().T >= p.schedule.GetDepartureTime()
}

// 发出导航请求
func (p *Person) requestRoute() {
	trip := p.schedule.GetTrip()
	// ATTENTION: 决定了出发后人/车的起始位置
	var startPosition entity.RoutePosition
	if p.runtime.Lane != nil && p.runtime.Aoi != nil {
		log.Panicf("person %d has both lane %v and aoi %v", p.ID(), p.runtime.Lane.ID(), p.runtime.Aoi.ID())
	}
	if p.runtime.Lane == nil && p.runtime.Aoi == nil {
		log.Panicf("person %d has neither lane nor aoi", p.ID())
	}
	// route还没走完 在外部切换到下一个route 不需要导航
	if p.multiModalRoute.Ok() {
		// do nothing
	} else {
		if p.runtime.Lane != nil {
			// 从上一次的位置出发
			lane := p.runtime.Lane
			s := p.runtime.S
			startPosition = entity.RoutePosition{Lane: lane, S: s}
			// 位置修正
			if schedule.IsDrivingTrip(trip) {
				if lane.Type() != mapv2.LaneType_LANE_TYPE_DRIVING {
					var drivingLane entity.ILane
					var s float64
					if lane.ParentRoad() != nil {
						drivingLane, s = lane.ParentRoad().ProjectToNearestDrivingLane(lane, s)
					} else {
						roadSuccessorWalkingLanes := lane.Successors()
						roadPredecessorWalkingLanes := lane.Predecessors()
						var roadWalkingLane entity.ILane
						for _, conn := range roadSuccessorWalkingLanes {
							if roadWalkingLane != nil && roadWalkingLane.ParentRoad() != nil {
								break
							}
							roadWalkingLane = conn.Lane
							s = 0
						}
						for _, conn := range roadPredecessorWalkingLanes {
							if roadWalkingLane != nil && roadWalkingLane.ParentRoad() != nil {
								break
							}
							roadWalkingLane = conn.Lane
							s = roadWalkingLane.Length()
						}
						drivingLane, s = roadWalkingLane.ParentRoad().ProjectToNearestDrivingLane(roadWalkingLane, s)
						if p.snapshot.Status != personv2.Status_STATUS_SLEEP && !p.runtime.IsTripEnd {
							// 维护行人节点
							lane.RemovePedestrian(p.pedestrian.node)
							p.runtime.Lane = drivingLane
							p.runtime.S = s
							p.pedestrian.node = newPedestrianNode(p.runtime.S, p)
						}
					}
					if drivingLane == nil {
						log.Panicf("person %d fail to request route due to bad walking->driving lane projection %+v", p.ID(), lane)
					}
					startPosition = entity.RoutePosition{Lane: drivingLane, S: s}
				}
			} else {
				// 如果是未知的出行模式，认为是步行
				// 修正为同一road上的walking lane，如果没有walking lane，则panic
				if lane.Type() != mapv2.LaneType_LANE_TYPE_WALKING {
					var walkingLane entity.ILane
					var s float64
					if lane.ParentRoad() != nil {
						walkingLane, s = lane.ParentRoad().ProjectToNearestWalkingLane(lane, s)
					} else {
						roadSuccessorDrivingLanes := lane.Successors()
						roadPredecessorDrivingLanes := lane.Predecessors()
						var roadDrivingLane entity.ILane
						for _, conn := range roadSuccessorDrivingLanes {
							if roadDrivingLane != nil && roadDrivingLane.ParentRoad() != nil {
								break
							}
							roadDrivingLane = conn.Lane
							s = 0
						}
						for _, conn := range roadPredecessorDrivingLanes {
							if roadDrivingLane != nil && roadDrivingLane.ParentRoad() != nil {
								break
							}
							roadDrivingLane = conn.Lane
							s = roadDrivingLane.Length()
						}
						walkingLane, s = roadDrivingLane.ParentRoad().ProjectToNearestDrivingLane(roadDrivingLane, s)
						if p.snapshot.Status != personv2.Status_STATUS_SLEEP && !p.runtime.IsTripEnd {
							// 维护车辆节点
							p.updateLaneVehicleNodes(false)
						}
					}
					if walkingLane == nil {
						log.Panicf("person %d fail to request route due to bad driving->walking lane projection %+v", p.ID(), lane)
					}
					startPosition = entity.RoutePosition{Lane: walkingLane, S: s}
				}
			}
		} else {
			startPosition = entity.RoutePosition{Aoi: p.runtime.Aoi}
		}
		p.multiModalRoute.Clear()
		// 根据trip类型发出不同类型的导航请求
		var routeType routingv2.RouteType
		if schedule.IsDrivingTrip(trip) {
			routeType = routingv2.RouteType_ROUTE_TYPE_DRIVING
		} else if schedule.IsWalkingTrip(trip) {
			routeType = routingv2.RouteType_ROUTE_TYPE_WALKING
		} else {
			log.Panicf("Invalid trip mode: %v", trip.Mode)
		}
		// taxi以外可以使用preroute
		p.multiModalRoute.ProduceRouting(trip, startPosition, routeType)
	}
}

// 导航请求是否成功,成功则返回true，否则转到下一trip并返回false
func (p *Person) routeSuccessful() (*tripv2.Trip, bool) {
	trip := p.schedule.GetTrip()
	p.multiModalRoute.Wait()
	if p.multiModalRoute.Ok() {
		return trip, true
	}
	p.schedule.NextTrip(p.ctx.Clock().T)
	return trip, false
}

// 产生人的基础Protobuf
func (p *Person) ToBasePb() *personv2.Person {
	pb := protoutil.Clone(p.base)
	pb.Schedules = lo.Map(p.schedule.Base(), func(s *tripv2.Schedule, _ int) *tripv2.Schedule {
		return protoutil.Clone(s)
	})
	return pb
}

// 产生人的运行时Protobuf
func (p *Person) ToMotionPb() *personv2.PersonMotion {
	return p.snapshot.ToPb(p.ctx, p)
}

// 产生全量人的运行时Protobuf
func (p *Person) ToPersonRuntimePb(returnBase bool) *personv2.PersonRuntime {
	pb := &personv2.PersonRuntime{
		Motion: p.ToMotionPb(),
	}
	if returnBase {
		pb.Base = p.ToBasePb()
	}
	return pb
}

func (p *Person) PersonType() personv2.PersonType {
	return p.base.Type
}

func (p *Person) String() string {
	s := fmt.Sprintf("Person %d", p.ID())
	s += fmt.Sprintf(" Snapshot: %v", &p.snapshot)
	s += fmt.Sprintf(" Runtime: %v", &p.runtime)
	return s
}

func (p *Person) DebugTripIndex() int32 {
	return p.schedule.TripIndex
}
