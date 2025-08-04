package aoi

import (
	"sync"

	"git.fiblab.net/general/common/v2/geometry"
	geov2 "git.fiblab.net/sim/protos/v2/go/city/geo/v2"
	mapv2 "git.fiblab.net/sim/protos/v2/go/city/map/v2"
	"github.com/samber/lo"
	"github.com/tsinghua-fib-lab/agentsociety-sim-oss/entity"
	"github.com/tsinghua-fib-lab/agentsociety-sim-oss/utils/randengine"
)

type aoiBufferItem struct {
	P entity.IPerson
}

type Aoi struct {
	ctx entity.ITaskContext

	base     *mapv2.Aoi
	id       int32
	area     *float64
	centroid geometry.Point
	boundary []geometry.Point // Aoi 边界点列表。各点顺序给出，注意第一点与最后一点相同

	laneSs       map[int32]float64      // aoi连接的车道id到对应道路上位置的映射
	drivingLanes map[int32]entity.ILane // 对应的行车路网车道指针
	walkingLanes map[int32]entity.ILane // 对应的步行路网车道指针

	generator *randengine.Engine // 随机数生成器

	persons               map[entity.IPerson]struct{} // 所有的人
	addPersonBuffer       []aoiBufferItem             // 缓存上一时刻进入AOI的人或进入室内行走的人
	addPersonBufferMtx    sync.Mutex
	removePersonBuffer    []aoiBufferItem // 缓存上一时刻离开AOI的人或离开室内行走的人
	removePersonBufferMtx sync.Mutex
}

// newAoi 创建并初始化一个新的AOI实例
// 功能：根据基础数据创建AOI对象，初始化边界、POI、车道连接、室内模拟等配置
// 参数：ctx-任务上下文，base-基础AOI数据，poiManager-POI管理器，laneManager-车道管理器
// 返回：初始化完成的AOI实例
func newAoi(ctx entity.ITaskContext, base *mapv2.Aoi, _ *AoiManager, laneManager entity.ILaneManager) *Aoi {
	a := &Aoi{
		ctx:  ctx,
		base: base,
		id:   base.Id,
		boundary: lo.Map(base.Positions, func(p *geov2.XYPosition, _ int) geometry.Point {
			return geometry.NewPointFromPb(p)
		}),
		area:         base.Area,
		laneSs:       make(map[int32]float64),
		drivingLanes: make(map[int32]entity.ILane),
		walkingLanes: make(map[int32]entity.ILane),
		persons:      make(map[entity.IPerson]struct{}),
		generator:    randengine.New(uint64(base.Id)),
	}
	a.centroid = geometry.GetPolygonCentroid2D(a.boundary)
	var sumZ float64
	for _, point := range a.boundary {
		sumZ += point.Z
	}
	a.centroid.Z = sumZ / float64(len(a.boundary))
	for _, position := range base.DrivingPositions {
		lane := laneManager.Get(position.LaneId)
		a.drivingLanes[lane.ID()] = lane
		a.laneSs[lane.ID()] = position.S
		lane.AddAoiWhenInit(a)
	}
	for _, position := range base.WalkingPositions {
		lane := laneManager.Get(position.LaneId)
		a.walkingLanes[lane.ID()] = lane
		a.laneSs[lane.ID()] = position.S
		lane.AddAoiWhenInit(a)
	}

	return a
}

// prepare 准备阶段，处理缓冲区的数据更新
// 功能：根据缓冲区数据更新AOI内的人员和车辆状态，包括添加/移除人员和停靠车辆
// 说明：处理上一时刻的缓冲区操作，更新内部数据结构，为输出准备数据列表
func (a *Aoi) prepare() {
	// 根据buffer更新人的情况
	for _, item := range a.removePersonBuffer {
		// 存在性检查
		if _, ok := a.persons[item.P]; !ok {
			log.Errorf("remove person %d not in aoi %d", item.P.ID(), a.id)
		}
		delete(a.persons, item.P)
	}
	a.removePersonBuffer = a.removePersonBuffer[:0]
	for _, item := range a.addPersonBuffer {
		// 存在性检查
		if _, ok := a.persons[item.P]; ok {
			log.Warnf("add person %d already in aoi %d", item.P.ID(), a.id)
		}
		a.persons[item.P] = struct{}{}
	}
	a.addPersonBuffer = a.addPersonBuffer[:0]
}

// update 更新阶段，执行AOI的模拟逻辑
// 功能：执行AOI的模拟更新逻辑，目前为空实现，预留扩展接口
// 参数：dt-时间步长
func (a *Aoi) update(dt float64) {
}

// ID 获取AOI的唯一标识符
// 功能：返回AOI的ID，用于标识和查找特定的AOI
// 返回：AOI的ID，如果AOI为nil则返回-1
func (a *Aoi) ID() int32 {
	if a == nil {
		return -1
	}
	return a.id
}

// Centroid 获取AOI的几何中心点坐标
// 功能：返回AOI多边形的几何中心点，用于定位和计算
// 返回：AOI中心点的坐标
func (a *Aoi) Centroid() geometry.Point {
	return a.centroid
}

// DrivingLanes 获取AOI连接的行车道映射
// 功能：返回AOI连接的所有行车道，以车道ID为键的映射表
// 返回：行车道ID到车道对象的映射
func (a *Aoi) DrivingLanes() map[int32]entity.ILane {
	return a.drivingLanes
}

// WalkingLanes 获取AOI连接的步行道映射
// 功能：返回AOI连接的所有步行道，以车道ID为键的映射表
// 返回：步行道ID到车道对象的映射
func (a *Aoi) WalkingLanes() map[int32]entity.ILane {
	return a.walkingLanes
}

// LaneSs 获取AOI连接的所有车道位置映射
// 功能：返回AOI连接的所有车道在道路上的位置信息
// 返回：车道ID到道路位置S坐标的映射
func (a *Aoi) LaneSs() map[int32]float64 {
	return a.laneSs
}

// DrivingS 获取指定行车道在AOI连接点的位置
// 功能：根据行车道ID返回该车道连接到AOI的位置坐标
// 参数：laneID-行车道ID
// 返回：车道在道路上的S坐标，如果车道不存在则panic
func (a *Aoi) DrivingS(laneID int32) float64 {
	if s, ok := a.laneSs[laneID]; !ok {
		log.Panicf("no lane %d with aoi %d", laneID, a.id)
		return 0
	} else {
		return s
	}
}

// WalkingS 获取指定步行道在AOI连接点的位置
// 功能：根据步行道ID返回该车道连接到AOI的位置坐标
// 参数：laneID-步行道ID
// 返回：车道在道路上的S坐标，如果车道不存在则panic
func (a *Aoi) WalkingS(laneID int32) float64 {
	if s, ok := a.laneSs[laneID]; !ok {
		log.Panicf("no lane %d with aoi %d", laneID, a.id)
		return 0
	} else {
		return s
	}
}

// AddPerson 添加人员到AOI缓冲区
// 功能：将人员添加到AOI的添加缓冲区，在下一时刻的prepare阶段处理
// 参数：p-要添加的人员，isCrowd-是否为室内行人
// 说明：使用缓冲区机制避免并发访问冲突
func (a *Aoi) AddPerson(p entity.IPerson) {
	item := aoiBufferItem{P: p}
	a.addPersonBufferMtx.Lock()
	a.addPersonBuffer = append(a.addPersonBuffer, item)
	a.addPersonBufferMtx.Unlock()
}

// RemovePerson 从AOI移除人员到缓冲区
// 功能：将人员添加到AOI的移除缓冲区，在下一时刻的prepare阶段处理
// 参数：p-要移除的人员，isCrowd-是否为室内行人
// 说明：使用缓冲区机制避免并发访问冲突
func (a *Aoi) RemovePerson(p entity.IPerson) {
	a.removePersonBufferMtx.Lock()
	a.removePersonBuffer = append(a.removePersonBuffer, aoiBufferItem{P: p})
	a.removePersonBufferMtx.Unlock()
}

// ToBasePb 获取AOI的基础protobuf数据
// 功能：返回AOI的原始protobuf数据，用于数据序列化和传输
// 返回：AOI的基础protobuf对象
func (a *Aoi) ToBasePb() *mapv2.Aoi {
	return a.base
}
