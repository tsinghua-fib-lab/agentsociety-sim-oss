package aoi

import (
	"fmt"

	"git.fiblab.net/general/common/v2/parallel"
	mapv2 "git.fiblab.net/sim/protos/v2/go/city/map/v2"
	"git.fiblab.net/sim/protos/v2/go/city/map/v2/mapv2connect"
	"github.com/samber/lo"
	"github.com/tsinghua-fib-lab/moss-agentsociety-go/entity"
)

// Aoi管理器
type AoiManager struct {
	mapv2connect.UnimplementedAoiServiceHandler

	ctx entity.ITaskContext

	data map[int32]*Aoi
	aois []*Aoi
}

// NewManager 创建AOI管理器实例
// 功能：初始化AOI管理器，创建内部数据结构
// 参数：ctx-任务上下文
// 返回：新创建的AOI管理器实例
func NewManager(ctx entity.ITaskContext) *AoiManager {
	m := &AoiManager{
		ctx:  ctx,
		data: make(map[int32]*Aoi),
		aois: make([]*Aoi, 0),
	}
	return m
}

// Init 初始化所有AOI与POI
// 功能：根据protobuf数据初始化所有AOI对象，建立与POI和车道的关联关系
// 参数：pbs-AOI的protobuf数据列表，poiManager-POI管理器，laneManager-车道管理器
// 说明：使用并行处理提高初始化效率
func (m *AoiManager) Init(pbs []*mapv2.Aoi, laneManager entity.ILaneManager) {
	// 初始化aoi
	m.aois = parallel.GoMap(pbs, func(pb *mapv2.Aoi) *Aoi {
		return newAoi(m.ctx, pb, m, laneManager)
	})
	m.data = lo.SliceToMap(m.aois, func(a *Aoi) (int32, *Aoi) {
		return a.id, a
	})
}

// Get 根据ID获取AOI实例
// 功能：通过AOI ID查找对应的AOI对象，如果不存在则panic
// 参数：id-AOI的唯一标识符
// 返回：对应的AOI实例，如果不存在则panic
func (m *AoiManager) Get(id int32) entity.IAoi {
	if aoi, ok := m.data[id]; !ok {
		log.Panicf("no id %d in aoi data", id)
		return nil
	} else {
		return aoi
	}
}

// GetOrError 根据ID获取AOI实例（带错误处理）
// 功能：通过AOI ID查找对应的AOI对象，如果不存在则返回错误
// 参数：id-AOI的唯一标识符
// 返回：AOI实例和错误信息，如果不存在则返回nil和错误
func (m *AoiManager) GetOrError(id int32) (entity.IAoi, error) {
	if aoi, ok := m.data[id]; !ok {
		return nil, fmt.Errorf("no id %d in aoi data", id)
	} else {
		return aoi, nil
	}
}

// Prepare 准备阶段，处理所有AOI的缓冲区数据
// 功能：对所有AOI执行准备阶段，处理人员进出和车辆停靠的缓冲区操作
// 说明：使用并行处理提高性能，为输出准备数据
func (m *AoiManager) Prepare() {
	parallel.GoFor(m.aois, func(a *Aoi) { a.prepare() })
}

// Update 更新阶段，执行所有AOI的模拟逻辑
// 功能：对所有AOI执行更新阶段，执行模拟计算逻辑
// 参数：dt-时间步长
// 说明：使用并行处理提高性能，目前大部分AOI的update为空实现
func (m *AoiManager) Update(dt float64) {
	parallel.GoFor(m.aois, func(a *Aoi) { a.update(dt) })
}
