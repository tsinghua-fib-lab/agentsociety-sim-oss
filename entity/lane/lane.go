package lane

import (
	"fmt"
	"math"
	"sort"
	"sync"

	"git.fiblab.net/sim/simulet-go/entity"

	"git.fiblab.net/general/common/v2/geometry"
	"git.fiblab.net/general/common/v2/mathutil"
	geov2 "git.fiblab.net/sim/protos/v2/go/city/geo/v2"
	mapv2 "git.fiblab.net/sim/protos/v2/go/city/map/v2"
	"github.com/samber/lo"
)

const (
	winLength = 600 // 统计路况的时间窗长度(s)
)

// Lane 车道实体
// 功能：表示地图中的车道，包含几何信息、交通状态、车辆/行人管理等功能
type Lane struct {
	ctx entity.ITaskContext

	id int32

	// 初始化临时变量

	initPredecessors []*mapv2.LaneConnection
	initSuccessors   []*mapv2.LaneConnection
	initLeftLaneIDs  []int32
	initRightLaneIDs []int32
	initOverlaps     []*mapv2.LaneOverlap

	typ               mapv2.LaneType   // 车道类型
	turn              mapv2.LaneTurn   // 转向类型
	maxV              float64          // 当前道路限速
	parentJunction    entity.IJunction // 所在路口
	parentRoad        entity.IRoad     // 所在道路
	parentID          int32
	offsetInRoad      int                          // 在道路中的索引，0为最左侧车道，1为左数第二侧车道，以此类推
	predecessors      map[int32]entity.Connection  // 前驱车道映射表
	successors        map[int32]entity.Connection  // 后继车道映射表
	uniquePredecessor entity.ILane                 // 唯一前驱
	uniqueSuccessor   entity.ILane                 // 唯一后继
	sideLanes         [2][]entity.ILane            // 左/右侧车道（按距离从近到远排序）
	aois              map[int32]entity.IAoi        // AOI映射表
	addAoiMutex       sync.Mutex                   // aois读写互斥锁
	overlaps          map[float64]entity.Overlap   // 冲突点数据集合
	lineLengths       []float64                    // 中心线折线点对应的的长度列表
	length            float64                      // 以中心线的长度为车道长度
	width             float64                      // 车道宽度
	lineDirections    []geometry.PolylineDirection // 中心线折线段每一段的方向（atan2）
	line              []geometry.Point             // 转成Point的中心线折线

	maxVBuffer float64 // 限速buffer
	k          float64 // 平滑系数

	pedestrians laneList[entity.IPerson, struct{}]
	vehicles    laneList[entity.IPerson, entity.VehicleSideLink]

	lightState              mapv2.LightState // 车道信号灯状态
	lightStateTotalTime     float64          // 车道信号灯本相位总时长
	lightStateRemainingTime float64          // 车道信号灯下一次切换时间
}

// newLane 创建并初始化一个新的Lane实例
// 功能：根据基础数据创建Lane对象，初始化几何信息、类型分类、信号灯等配置
// 参数：ctx-任务上下文，base-基础Lane数据
// 返回：初始化完成的Lane实例
// 说明：根据车道类型初始化不同的车辆/行人列表，计算几何属性和统计参数
func newLane(ctx entity.ITaskContext, base *mapv2.Lane) *Lane {
	l := &Lane{
		ctx:                     ctx,
		id:                      base.Id,
		initPredecessors:        base.Predecessors,
		initSuccessors:          base.Successors,
		initLeftLaneIDs:         base.LeftLaneIds,
		initRightLaneIDs:        base.RightLaneIds,
		initOverlaps:            base.Overlaps,
		typ:                     base.Type,
		turn:                    base.Turn,
		maxV:                    base.MaxSpeed,
		k:                       math.Exp(-ctx.Clock().DT / winLength),
		predecessors:            make(map[int32]entity.Connection),
		successors:              make(map[int32]entity.Connection),
		sideLanes:               [2][]entity.ILane{},
		aois:                    make(map[int32]entity.IAoi),
		addAoiMutex:             sync.Mutex{},
		overlaps:                make(map[float64]entity.Overlap),
		lineLengths:             make([]float64, 0),
		lineDirections:          make([]geometry.PolylineDirection, 0),
		line:                    make([]geometry.Point, 0),
		width:                   base.Width,
		lightState:              mapv2.LightState_LIGHT_STATE_GREEN,
		lightStateTotalTime:     mathutil.INF,
		lightStateRemainingTime: mathutil.INF,
		maxVBuffer:              base.MaxSpeed,
	}
	l.line = lo.Map(base.CenterLine.Nodes, func(node *geov2.XYPosition, _ int) geometry.Point {
		return geometry.NewPointFromPb(node)
	})
	l.lineLengths = geometry.GetPolylineLengths2D(l.line)
	l.length = l.lineLengths[len(l.lineLengths)-1]
	l.lineDirections = geometry.GetPolylineDirections(l.line)

	switch l.typ {
	case mapv2.LaneType_LANE_TYPE_DRIVING:
		l.vehicles = newLaneList[entity.IPerson, entity.VehicleSideLink](
			fmt.Sprintf("lane %d vehicles", l.id),
		)
	case mapv2.LaneType_LANE_TYPE_WALKING:
		l.pedestrians = newLaneList[entity.IPerson, struct{}](
			fmt.Sprintf("lane %d pedestrians", l.id),
		)
	case mapv2.LaneType_LANE_TYPE_RAIL_TRANSIT:
	default:
		log.Panicf("bad type %v for lane %d", l.typ, l.id)
	}
	return l
}

// initWithManager 在管理器初始化后建立Lane的连接关系
// 功能：根据初始化数据建立前驱、后继、侧车道、冲突点等连接关系
// 参数：laneManager-车道管理器
// 说明：建立车道间的拓扑关系，为后续模拟提供基础
func (l *Lane) initWithManager(laneManager entity.ILaneManager) {
	for _, conn := range l.initPredecessors {
		lane := laneManager.Get(conn.Id)
		l.predecessors[conn.Id] = entity.Connection{Lane: lane, Type: conn.Type}
	}
	if len(l.predecessors) == 1 {
		for _, conn := range l.predecessors {
			l.uniquePredecessor = conn.Lane
			break
		}
	}
	for _, conn := range l.initSuccessors {
		lane := laneManager.Get(conn.Id)
		l.successors[conn.Id] = entity.Connection{Lane: lane, Type: conn.Type}
	}
	if len(l.successors) == 1 {
		for _, conn := range l.successors {
			l.uniqueSuccessor = conn.Lane
			break
		}
	}
	for _, id := range l.initLeftLaneIDs {
		lane := laneManager.Get(id)
		l.sideLanes[entity.LEFT] = append(l.sideLanes[entity.LEFT], lane)
	}
	for _, id := range l.initRightLaneIDs {
		lane := laneManager.Get(id)
		l.sideLanes[entity.RIGHT] = append(l.sideLanes[entity.RIGHT], lane)
	}
	for _, overlap := range l.initOverlaps {
		lane := laneManager.Get(overlap.Other.LaneId)
		l.overlaps[overlap.Self.S] = entity.Overlap{
			Other:     lane,
			OtherS:    overlap.Other.S,
			SelfFirst: overlap.SelfFirst,
		}
	}
	l.initPredecessors = nil
	l.initSuccessors = nil
	l.initLeftLaneIDs = nil
	l.initRightLaneIDs = nil
	l.initOverlaps = nil
}

// prepare 准备阶段，处理Lane的准备工作
// 功能：更新限行限速状态，处理停靠车辆缓冲区，维护车辆/行人列表，更新运行时数据
// 说明：使用缓冲区机制提高并发性能，避免在更新阶段进行写操作
func (l *Lane) prepare() {
	// 限速buffer写入
	l.maxV = l.maxVBuffer
	// 维护本车道链表
	l.pedestrians.prepare()
	l.vehicles.prepare()
}

// prepare2 第二阶段准备，处理车道间的侧链关系和行人占用计算
// 功能：为行车道构建侧链关系，为人行道计算占用区间
// 说明：等待相邻车道完成主链构建后进行，确保数据一致性
func (l *Lane) prepare2() {
	switch l.typ {
	case mapv2.LaneType_LANE_TYPE_DRIVING:
		// 等待相邻车道完成主链构建并进行支链构建
		for which := range []int{entity.LEFT, entity.RIGHT} {
			thisSideLanes := l.sideLanes[which]
			if len(thisSideLanes) > 0 {
				neighborLane := thisSideLanes[0]
				// 根据邻居车道链表构建本车道链表支链
				nList := neighborLane.Vehicles()
				var nBack *entity.VehicleNode = nil
				nFront := nList.First()
				if nFront == nil {
					// 隔壁车道没车，不需要任何处理
					continue
				}
				for node := l.vehicles.list.First(); node != nil; node = node.Next() {
					nodeRatio := node.S / l.length
					// 找到第一个位置大于node的邻居车道上的车
					// 则nFront是第一个位置大于等于node的车，nBack是第一个位置小于node的车
					// 该算法能处理nFront和nBack为nil的情况
					for nFront != nil && nFront.S/neighborLane.Length() < nodeRatio {
						nBack = nFront
						nFront = nFront.Next()
					}
					node.Extra.Links[which][entity.BEFORE] = nBack
					node.Extra.Links[which][entity.AFTER] = nFront
				}
			}
		}
	}
}

// update 更新阶段，执行Lane的模拟逻辑
// 功能：更新行车道的车辆统计、路况计算、能耗排放统计等
// 说明：只对行车道进行统计更新，使用指数平滑算法计算平均车速
func (l *Lane) update() {
}

// 数据初始化

// SetParentRoadWhenInit 设置lane所在road与偏移量
// 功能：在初始化阶段设置Lane所属的道路和偏移量
// 参数：parent-所属道路，offset-在道路中的偏移量
// 说明：设置后清除junction关联，更新parentID
func (l *Lane) SetParentRoadWhenInit(parent entity.IRoad, offset int) {
	l.parentRoad = parent
	l.offsetInRoad = offset
	l.parentJunction = nil
	l.parentID = parent.ID()
}

// SetParentJunctionWhenInit 设置lane所在junction
// 功能：在初始化阶段设置Lane所属的路口
// 参数：parent-所属路口
// 说明：设置后清除道路关联，更新parentID，为人行道初始化占用区间
func (l *Lane) SetParentJunctionWhenInit(parent entity.IJunction) {
	l.parentJunction = parent
	l.parentRoad = nil
	l.parentID = parent.ID()
}

// AddAoiWhenInit 添加lane上的aoi
// 功能：在初始化阶段添加Lane关联的AOI
// 参数：aoi-要添加的AOI
// 说明：使用互斥锁保证线程安全
func (l *Lane) AddAoiWhenInit(aoi entity.IAoi) {
	l.addAoiMutex.Lock()
	l.aois[aoi.ID()] = aoi
	l.addAoiMutex.Unlock()
}

// 静态数据

func (l *Lane) String() string {
	return fmt.Sprintf("Lane %d", l.id)
}

// 获取Lane ID
func (l *Lane) ID() int32 {
	if l == nil {
		return -1
	}
	return l.id
}

// 获取Lane长度
func (l *Lane) Length() float64 {
	return l.length
}

// 获取Lane宽度
func (l *Lane) Width() float64 {
	return l.width
}

// 获取Lane类型
func (l *Lane) Type() mapv2.LaneType {
	return l.typ
}

// 获取Lane转向类型
func (l *Lane) Turn() mapv2.LaneTurn {
	return l.turn
}

// 获取Lane的父对象(road/junction)的ID
func (l *Lane) ParentID() int32 {
	return l.parentID
}

// 获取Lane的中心线
func (l *Lane) Line() []geometry.Point {
	return l.line
}

// Road Lane在Road中的偏移量，最左侧为0，往右侧递增
func (l *Lane) OffsetInRoad() int {
	if l.parentRoad == nil {
		log.Panicf("Lane %d: Not in road", l.id)
	}
	return l.offsetInRoad
}

// 获取Lane上的Aoi列表
func (l *Lane) Aois() map[int32]entity.IAoi {
	return l.aois
}

// 获取Lane的所有后继Lane与连接关系
func (l *Lane) Successors() map[int32]entity.Connection {
	return l.successors
}

// 获取Lane的所有前驱Lane与连接关系
func (l *Lane) Predecessors() map[int32]entity.Connection {
	return l.predecessors
}

// 查询唯一前驱，仅限于车道类型为DRIVING的路口内车道
func (l *Lane) UniquePredecessor() (entity.ILane, error) {
	if l.parentJunction == nil || l.typ != mapv2.LaneType_LANE_TYPE_DRIVING {
		return nil, fmt.Errorf("Lane %d: Not in junction or not driving", l.id)
	}
	return l.uniquePredecessor, nil
}

// 查询唯一后继，仅限于车道类型为DRIVING的路口内车道
func (l *Lane) UniqueSuccessor() (entity.ILane, error) {
	if l.parentJunction == nil || l.typ != mapv2.LaneType_LANE_TYPE_DRIVING {
		return nil, fmt.Errorf("Lane %d: Not in junction or not driving", l.id)
	}
	return l.uniqueSuccessor, nil
}

// GetPressure 计算Junction Lane的压力，用于信号灯控制
// 功能：计算车道压力值，基于前驱和后继车道的车辆密度差
// 返回：压力值，正值表示拥堵，负值表示畅通
// 算法说明：
// 1. 右转车道和步行道不参与压力计算
// 2. 计算前驱车道的车辆密度（车辆数/长度）
// 3. 计算后继车道的车辆密度
// 4. 压力 = 前驱密度 - 后继密度
// 5. 对于短车道（<10米），考虑相邻车道的车辆
func (l *Lane) GetPressure() float64 {
	if l.typ == mapv2.LaneType_LANE_TYPE_UNSPECIFIED {
		log.Panicf("Lane %d: Lane type not specified", l.id)
	}
	if l.typ == mapv2.LaneType_LANE_TYPE_WALKING {
		return 0
	}
	if l.turn == mapv2.LaneTurn_LANE_TURN_RIGHT {
		// 右转也不纳入压力考虑
		return 0
	}
	if l.uniqueSuccessor == nil || l.uniquePredecessor == nil {
		log.Panicf("Lane %d: Either successor or predecessor is not unique", l.id)
	}
	pre := l.uniquePredecessor
	incoming := .0
	// 车辆数/长度
	if pre.Length() > 10 {
		incoming = float64(pre.Vehicles().Len()) / pre.Length()
	} else {
		// 如果前驱车道长度小于10米，则向前多考虑一个路口内的车道，把堵在路口的车也考虑进来
		totalLength := pre.Length()
		totalCount := pre.Vehicles().Len()
		for _, conn := range pre.Predecessors() {
			totalLength += conn.Lane.Length()
			totalCount += conn.Lane.Vehicles().Len()
		}
		incoming = float64(totalCount) / totalLength
	}
	// 按后继数均分
	incoming /= float64(len(pre.Successors()))

	suc := l.uniqueSuccessor
	// 车辆数/长度
	outgoing := .0
	if suc.Length() > 10 {
		outgoing = float64(suc.Vehicles().Len()) / suc.Length()
	} else {
		// 如果后继车道长度小于10米，则向后多考虑一个路口内的车道，把堵在路口的车也考虑进来
		totalLength := suc.Length()
		totalCount := suc.Vehicles().Len()
		for _, conn := range suc.Successors() {
			totalLength += conn.Lane.Length()
			totalCount += conn.Lane.Vehicles().Len()
		}
		outgoing = float64(totalCount) / totalLength
	}
	// 按前驱数均分
	outgoing /= float64(len(suc.Predecessors()))
	return incoming - outgoing
}

// VehicleCount 统计非影子车辆数
// 功能：统计车道上的非影子车辆数量，用于交通流分析
// 返回：非影子车辆数量
// 说明：影子车辆是用于变道模拟的虚拟车辆，不计入实际统计
func (l *Lane) VehicleCount() int32 {
	var cnt int32
	for node := l.Vehicles().First(); node != nil; node = node.Next() {
		if node.Value.ShadowLane() != l {
			cnt++
		}
	}
	return cnt
}

// IsRightTurnDrivingLane 检查是否是右转行车道
// 功能：判断车道是否为右转专用行车道
// 返回：true表示是右转行车道，false表示不是
func (l *Lane) IsRightTurnDrivingLane() bool {
	return l.typ == mapv2.LaneType_LANE_TYPE_DRIVING && l.turn == mapv2.LaneTurn_LANE_TURN_RIGHT
}

// IsClean 检查车道是否干净
// 功能：判断车道是否没有车辆，用于信号灯控制
// 返回：true表示车道干净，false表示有车辆
// 说明：步行道和右转车道始终认为是干净的
func (l *Lane) IsClean() bool {
	if l.typ == mapv2.LaneType_LANE_TYPE_WALKING {
		return true
	}
	if l.turn == mapv2.LaneTurn_LANE_TURN_RIGHT {
		return true
	}
	return l.Vehicles().Len() == 0
}

// 获取Lane的冲突点数据集合
func (l *Lane) Overlaps() map[float64]entity.Overlap {
	return l.overlaps
}

// 获取Lane所在的Road
func (l *Lane) ParentRoad() entity.IRoad {
	return l.parentRoad
}

// 获取Lane所在的Junction
func (l *Lane) ParentJunction() entity.IJunction {
	return l.parentJunction
}

// 检查Lane是否为Road Lane
func (l *Lane) InRoad() bool {
	return l.parentRoad != nil
}

// 检查Lane是否为Junction Lane
func (l *Lane) InJunction() bool {
	return l.parentJunction != nil
}

// 获取左侧的Lane
func (l *Lane) LeftLane() entity.ILane {
	if len(l.sideLanes[entity.LEFT]) == 0 {
		return nil
	} else {
		return l.sideLanes[entity.LEFT][0]
	}
}

// 获取右侧的Lane
func (l *Lane) RightLane() entity.ILane {
	if len(l.sideLanes[entity.RIGHT]) == 0 {
		return nil
	} else {
		return l.sideLanes[entity.RIGHT][0]
	}
}

// 根据side获取左(side=0)/右(side=1)侧的Lane
func (l *Lane) NeighborLane(side int) entity.ILane {
	if len(l.sideLanes[side]) == 0 {
		return nil
	} else {
		return l.sideLanes[side][0]
	}
}

// 获取Lane的中心线
func (l *Lane) CenterLine() []geometry.Point {
	return l.line
}

// 获取Lane的中心线长度
func (l *Lane) CenterLineLengths() []float64 {
	return l.lineLengths
}

// 信号灯

// 获取信号灯状态
func (l *Lane) Light() (mapv2.LightState, float64, float64) {
	return l.lightState, l.lightStateTotalTime, l.lightStateRemainingTime
}

// 设置信号灯状态
func (l *Lane) SetLight(state mapv2.LightState, totalTime float64, remainingTime float64) {
	l.lightState = state
	l.lightStateTotalTime = totalTime
	l.lightStateRemainingTime = remainingTime
}

// 检查是否是人行道
func (l *Lane) IsWalkLane() bool {
	return l.Type() == mapv2.LaneType_LANE_TYPE_WALKING
}

// 路况

// 获取车道限速
func (l *Lane) MaxV() float64 {
	return l.maxV
}

// 设置车道限速
func (l *Lane) SetMaxV(v float64) {
	l.maxVBuffer = v
}

// 人车更新相关函数

// 获取车道上的车辆
func (l *Lane) Vehicles() *entity.VehicleList {
	return l.vehicles.list
}

// 获取车道上的行人
func (l *Lane) Pedestrians() *entity.PedestrianList {
	return l.pedestrians.list
}

// 向Lane链表中添加行人（Prepare后生效）
func (l *Lane) AddPedestrian(node *entity.PedestrianNode) {
	l.pedestrians.add(node)
}

// 从Lane链表中移除行人（Prepare后生效）
func (l *Lane) RemovePedestrian(node *entity.PedestrianNode) {
	l.pedestrians.remove(node)
}

// 向Lane链表中添加车辆（Prepare后生效）
func (l *Lane) AddVehicle(node *entity.VehicleNode) {
	l.vehicles.add(node)
}

// 从Lane链表中移除车辆（Prepare后生效）
func (l *Lane) RemoveVehicle(node *entity.VehicleNode) {
	l.vehicles.remove(node)
}

// 获取第一辆车
func (l *Lane) FirstVehicle() *entity.VehicleNode {
	return l.vehicles.list.First()
}

// 获取最后一辆车
func (l *Lane) LastVehicle() *entity.VehicleNode {
	return l.vehicles.list.Last()
}

// 获取lane相对于本车道的位置，左负右正
func (l *Lane) GetRelativePosition(lane entity.ILane) int32 {
	if lane == l {
		return 0
	}
	i := int32(-1)
	for _, leftLane := range l.sideLanes[entity.LEFT] {
		if leftLane == lane {
			return i
		}
		i--
	}
	i = 1
	for _, rightLane := range l.sideLanes[entity.RIGHT] {
		if rightLane == lane {
			return i
		}
		i++
	}
	log.Panic("compared lanes should be in same road")
	return 0
}

// 从候选集中选出与本车道最近的车道，要求候选集与本车道都在同一道路中
func (l *Lane) GetClosestLane(candidates []entity.ILane) entity.ILane {
	lanePos := map[entity.ILane]int{l: 0}
	i := 0
	for _, leftLane := range l.sideLanes[entity.LEFT] {
		i++
		lanePos[leftLane] = i
	}
	i = 0
	for _, rightLane := range l.sideLanes[entity.RIGHT] {
		i++
		lanePos[rightLane] = i
	}
	i = len(lanePos)
	var minLane entity.ILane
	for _, lane := range candidates {
		if j := lanePos[lane]; j < i {
			i = j
			minLane = lane
		}
	}
	return minLane
}

// 对同一道路内的车道按比例"投影"
func (l *Lane) ProjectFromLane(other entity.ILane, otherS float64) float64 {
	if l.ParentRoad() != other.ParentRoad() {
		log.Panic("project from lane in different road")
		return 0
	} else {
		return lo.Clamp(otherS/other.Length()*l.length, 0, l.length)
	}
}

// 根据本车道s坐标计算切向角度
func (l *Lane) GetDirectionByS(s float64) (direction geometry.PolylineDirection) {
	if s < l.lineLengths[0] || s > l.lineLengths[len(l.lineLengths)-1] {
		log.Debugf("get direction with s %v out of range{%v,%v}",
			s, l.lineLengths[0], l.lineLengths[len(l.lineLengths)-1])
		s = lo.Clamp(s, l.lineLengths[0], l.lineLengths[len(l.lineLengths)-1])
	}
	if i := sort.SearchFloat64s(l.lineLengths, s); i == 0 {
		direction = l.lineDirections[0]
	} else {
		direction = l.lineDirections[i-1]
	}
	return
}

// 将当前车道s坐标转换为xy(z)坐标
func (l *Lane) GetPositionByS(s float64) (pos geometry.Point) {
	if s < l.lineLengths[0] || s > l.lineLengths[len(l.lineLengths)-1] {
		log.Debugf("get position with s %v out of range{%v,%v}",
			s, l.lineLengths[0], l.lineLengths[len(l.lineLengths)-1])
		s = lo.Clamp(s, l.lineLengths[0], l.lineLengths[len(l.lineLengths)-1])
	}
	if i := sort.SearchFloat64s(l.lineLengths, s); i == 0 {
		pos = l.line[0]
	} else {
		sHigh, sLow := l.lineLengths[i], l.lineLengths[i-1]
		k := (s - sLow) / (sHigh - sLow)
		if k < 0 || k > 1 {
			log.Panicf("lane: GetPositionByS(), bad k %v due to pos %v. sHigh=%f, sLow=%f, s=%f", k, pos, sHigh, sLow, s)
		}
		pos = geometry.Blend(l.line[i-1], l.line[i], k)
	}
	return
}

func (l *Lane) GetOffsetPositionByS(s, offset float64) (pos geometry.Point) {
	originalPos := l.GetPositionByS(s)
	direction := l.GetDirectionByS(s)
	unitNormal := geometry.Point{X: math.Cos(direction.Direction - math.Pi/2), Y: math.Sin(direction.Direction - math.Pi/2)}
	return geometry.Point{X: originalPos.X + unitNormal.X*offset, Y: originalPos.Y + unitNormal.Y*offset, Z: originalPos.Z}
}

// 将xyz坐标投影到车道折线上，计算出对应的s坐标
func (l *Lane) ProjectToLane(pos geometry.Point) float64 {
	s := geometry.GetClosestPolylineSToPoint2D(l.line, l.lineLengths, pos)
	return lo.Clamp(s, 0, l.length)
}

// 检查车道是否不能通行（不是绿灯）
func (l *Lane) IsNoEntry() bool {
	return l.InJunction() && l.lightState != mapv2.LightState_LIGHT_STATE_GREEN
}
