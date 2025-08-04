package entity

import (
	"fmt"

	"git.fiblab.net/general/common/v2/geometry"
	mapv2 "git.fiblab.net/sim/protos/v2/go/city/map/v2"
	personv2 "git.fiblab.net/sim/protos/v2/go/city/person/v2"
	tripv2 "git.fiblab.net/sim/protos/v2/go/city/trip/v2"
	"github.com/tsinghua-fib-lab/moss-agentsociety-go/utils/container"
)

// 方位常量
const (
	LEFT   = 0 // 左侧
	RIGHT  = 1 // 右侧
	BEFORE = 0 // 后方，等价于prev/behind
	AFTER  = 1 // 前方，等价于next/ahead
)

// 导航起点终点，初始化时lane+s/aoi二选一，最后需要都有
// （aoi需要转换出对应的出门进门的lane和s）
// XY为终点坐标，指示AOI室内导航的终点
type RoutePosition struct {
	Lane ILane
	S    float64
	Aoi  IAoi
	XY   *geometry.Point
}

func (r RoutePosition) String() string {
	return fmt.Sprintf("RoutePosition{Lane=%v, S=%v, Aoi=%v, XY=%v}", r.Lane, r.S, r.Aoi.ID(), r.XY)
}

// entity/person/person.go的依赖倒置
type IPerson interface {
	// 自身属性

	ID() int32                               // 获取人的ID
	Attr() *personv2.PersonAttribute         // 获取人的属性
	VehicleAttr() *personv2.VehicleAttribute // 获取人开车时的车辆属性
	BusAttr() *personv2.BusAttribute         // 获取人作为公交车司机时的公交车属性
	BikeAttr() *personv2.BikeAttribute       // 获取人骑自行车时的自行车属性

	ParentID() int32                 // 获取人的空间父对象ID
	PersonType() personv2.PersonType // Person类型
	Aoi() IAoi                       // 获取人所在的Aoi
	Lane() ILane                     // 获取人所在的Lane
	S() float64                      // 获取人在Lane上的位置S坐标
	ShadowLane() ILane               // 获取车辆影子所在的Lane
	ShadowS() float64                // 获取车辆影子在Lane上的位置S坐标
	XYZ() geometry.Point             // 获取人的位置坐标
	V() float64                      // 获取人的速度
	Length() float64                 // 获取人在当前状态下的长度（开车->车长）
	IsLC() bool                      // 判断车辆是否正在变道
	Status() personv2.Status         // 获取人的状态
	IsForward() bool                 // 判断人是否朝向车道前进方向
	SetSchedules(schedules []*tripv2.Schedule)
	DebugTripIndex() int32 // 获取调试用的trip index

	GetLabel(key string) (string, bool) // 获取指定键的标签值
	// print

	String() string

	// 输出

	ToBasePb() *personv2.Person                                // 产生人的基础Protobuf
	ToMotionPb() *personv2.PersonMotion                        // 产生人的运行时Protobuf
	ToPersonRuntimePb(returnBase bool) *personv2.PersonRuntime // 产生人的运行时Protobuf（全量）
}

// Lane连接关系
type Connection struct {
	Lane ILane                    // 连接到的Lane
	Type mapv2.LaneConnectionType // 连接类型
}

// Lane冲突点
type Overlap struct {
	Other     ILane   // 冲突Lane
	OtherS    float64 // 冲突车道的S坐标
	SelfFirst bool    // 是否本Lane优先
}

// 车辆链表支链，记录左右车道的前后车辆
type VehicleSideLink struct {
	// [LEFT/RIGHT][BACK/FRONT]
	Links [2][2]*container.ListNode[IPerson, VehicleSideLink]
}

func (l VehicleSideLink) String() string {
	s := ""
	if l.Links[0][0] != nil {
		s += fmt.Sprintf("L-B: %v, ", l.Links[0][0].Value.ID())
	} else {
		s += "L-B: nil, "
	}
	if l.Links[0][1] != nil {
		s += fmt.Sprintf("L-F: %v, ", l.Links[0][1].Value.ID())
	} else {
		s += "L-F: nil, "
	}
	if l.Links[1][0] != nil {
		s += fmt.Sprintf("R-B: %v, ", l.Links[1][0].Value.ID())
	} else {
		s += "R-B: nil, "
	}
	if l.Links[1][1] != nil {
		s += fmt.Sprintf("R-F: %v", l.Links[1][1].Value.ID())
	} else {
		s += "R-F: nil"
	}
	return s
}

// 清空链表
func (l *VehicleSideLink) Clear() {
	l.Links[0][0] = nil
	l.Links[0][1] = nil
	l.Links[1][0] = nil
	l.Links[1][1] = nil
}

// 车辆链表节点类型
type VehicleNode = container.ListNode[IPerson, VehicleSideLink]

// 车辆链表类型
type VehicleList = container.List[IPerson, VehicleSideLink]

// 行人链表节点类型
type PedestrianNode = container.ListNode[IPerson, struct{}]

// 行人链表类型
type PedestrianList = container.List[IPerson, struct{}]

// entity/lane/lane.go的依赖倒置
type ILane interface {
	ILaneTrafficLightSetter

	// 初始化

	SetParentRoadWhenInit(parent IRoad, offset int) // 设置lane所在road的指针与偏移量
	SetParentJunctionWhenInit(parent IJunction)     // 设置lane所在junction
	AddAoiWhenInit(aoi IAoi)                        // 添加lane上的aoi

	// Print

	String() string

	// getter

	ID() int32              // 获取Lane ID
	Length() float64        // 获取Lane长度
	Width() float64         // 获取Lane宽度
	Type() mapv2.LaneType   // 获取Lane类型
	Turn() mapv2.LaneTurn   // 获取Lane转向类型
	ParentID() int32        // 获取Lane的父对象(road/junction)的ID
	Line() []geometry.Point // 获取Lane的中心线
	OffsetInRoad() int      // Road Lane在Road中的偏移量，最左侧为0，往右侧递增

	ProjectFromLane(l ILane, s float64) float64 // 对同一道路内的车道按比例"投影"
	GetClosestLane(candidates []ILane) ILane    // 从候选集中选出与本车道最近的车道，要求候选集与本车道都在同一道路中
	GetRelativePosition(lane ILane) int32       // 获取lane相对于本车道的位置，左负右正

	Predecessors() map[int32]Connection // 获取Lane的所有前驱Lane与连接关系
	Successors() map[int32]Connection   // 获取Lane的所有后继Lane与连接关系
	// 查询唯一前驱，仅限于车道类型为DRIVING的路口内车道
	UniquePredecessor() (ILane, error)
	// 查询唯一后继，仅限于车道类型为DRIVING的路口内车道
	UniqueSuccessor() (ILane, error)
	Overlaps() map[float64]Overlap                         // 获取Lane上的冲突点列表
	Aois() map[int32]IAoi                                  // 获取Lane上的Aoi列表
	LeftLane() ILane                                       // 获取左侧的Lane
	RightLane() ILane                                      // 获取右侧的Lane
	NeighborLane(side int) ILane                           // 根据side获取左(side=0)/右(side=1)侧的Lane
	CenterLine() []geometry.Point                          // 获取Lane的中心线
	CenterLineLengths() []float64                          // 获取Lane的中心线长度
	GetPositionByS(s float64) geometry.Point               // 将当前车道s坐标转换为xy坐标
	GetOffsetPositionByS(s, offset float64) geometry.Point // 将当前车道s坐标，沿行进方向平移offset后的坐标转换为xy坐标
	GetDirectionByS(s float64) geometry.PolylineDirection  // 根据本车道s坐标计算切向角度
	ProjectToLane(pos geometry.Point) float64              // 将xy坐标投影到车道上，返回s坐标
	InRoad() bool                                          // 检查Lane是否为Road Lane
	InJunction() bool                                      // 检查Lane是否为Junction Lane
	IsNoEntry() bool                                       // 检查车道是否不能通行（不是绿灯）

	// 获取特定位置车辆

	FirstVehicle() *VehicleNode   // 获取第一辆车
	LastVehicle() *VehicleNode    // 获取最后一辆车
	Vehicles() *VehicleList       // 获取车道上的车辆
	VehicleCount() int32          // 统计非影子车辆数
	Pedestrians() *PedestrianList // 获取车道上的行人

	// 车道状态

	MaxV() float64                                                             // 获取车道限速
	Light() (state mapv2.LightState, totalTime float64, remainingTime float64) // 获取信号灯状态

	// 所在道路/路口

	ParentRoad() IRoad         // 获取Lane所在的Road
	ParentJunction() IJunction // 获取Lane所在的Junction

	// Lane链表操作

	AddVehicle(node *VehicleNode)          // 向Lane链表中添加车辆（Prepare后生效）
	RemoveVehicle(node *VehicleNode)       // 从Lane链表中移除车辆（Prepare后生效）
	AddPedestrian(node *PedestrianNode)    // 向Lane链表中添加行人（Prepare后生效）
	RemovePedestrian(node *PedestrianNode) // 从Lane链表中移除行人（Prepare后生效）

	// setter

	SetMaxV(v float64) // 设置车道限速
}

// 车道的信控接口
type ILaneTrafficLightSetter interface {
	GetPressure() float64                                                      // 计算Junction Lane的压力，用于信号灯控制
	SetLight(state mapv2.LightState, totalTime float64, remainingTime float64) // 设置信号灯状态
	IsWalkLane() bool                                                          // 检查是否是人行道
	IsRightTurnDrivingLane() bool                                              // 检查是否是右转行车道
}

// entity/road/road.go的依赖倒置
type IRoad interface {
	String() string

	ID() int32                     // 获取Road ID
	Name() string                  // 获取Road名称
	Lanes() map[int32]ILane        // 获取Road的所有Lane(ID -> Lane)
	RightestDrivingLane() ILane    // 获取最右侧的行车道（最靠近路边）
	DrivingPredecessor() IJunction // 获取前驱Junction
	DrivingSuccessor() IJunction   // 获取后继Junction

	ProjectToNearestDrivingLane(walkingLane ILane, s float64) (drivingLane ILane, newS float64) // 从步行道投影到最近的行车道
	ProjectToNearestWalkingLane(drivingLane ILane, s float64) (walkingLane ILane, newS float64) // 从行车道投影到最近的步行道

	MaxV() float64 // 获取道路限速
	GetAvgDrivingL() float64
}

// entity/junction/junction.go的依赖倒置
type IJunction interface {
	ID() int32              // 获取Junction ID
	Lanes() map[int32]ILane // 获取Junction内的所有车道（Lane ID -> Lane）
	HasTrafficLight() bool  // 判断是否有信号灯

	// 根据(入道路, 出道路) 获取Junction内的行车道组与角度
	DrivingLaneGroup(inRoad, outRoad IRoad) (lanes []ILane, inAngle, outAngle float64, ok bool)
}

// entity/aoi/aoi.go的依赖倒置
type IAoi interface {
	// 自身属性

	ID() int32                // 获取Aoi ID
	Centroid() geometry.Point // 获取Aoi中心点坐标

	// 道路连接关系

	DrivingLanes() map[int32]ILane // 获取Aoi连接到的行车道（Lane ID -> ILane）
	DrivingS(laneID int32) float64 // 输入行车道ID，返回对应的S坐标
	WalkingLanes() map[int32]ILane // 获取Aoi连接到的步行道（Lane ID -> ILane）
	WalkingS(laneID int32) float64 // 输入步行道ID，返回对应的S坐标
	LaneSs() map[int32]float64     // 获取Aoi连接到的所有Lane上的位置（Lane ID -> S）

	AddPerson(p IPerson)    // 添加人到Aoi
	RemovePerson(p IPerson) // 从Aoi中移除人
}
