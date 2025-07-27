package lane

import (
	"fmt"

	"git.fiblab.net/general/common/v2/parallel"
	mapv2 "git.fiblab.net/sim/protos/v2/go/city/map/v2"
	"git.fiblab.net/sim/protos/v2/go/city/map/v2/mapv2connect"
	"git.fiblab.net/sim/simulet-go/entity"
	"github.com/samber/lo"
)

// LaneManager Lane管理器
// 功能：管理所有Lane实体，提供创建、查找、初始化、输出等功能
type LaneManager struct {
	mapv2connect.UnimplementedLaneServiceHandler

	ctx entity.ITaskContext

	data  map[int32]*Lane
	lanes []*Lane
}

// NewManager 创建Lane管理器实例
// 功能：初始化Lane管理器，创建内部数据结构
// 参数：ctx-任务上下文
// 返回：新创建的Lane管理器实例
func NewManager(ctx entity.ITaskContext) *LaneManager {
	return &LaneManager{
		ctx:   ctx,
		data:  make(map[int32]*Lane),
		lanes: make([]*Lane, 0),
	}
}

// Init 初始化所有Lane
// 功能：根据protobuf数据初始化所有Lane对象，建立ID映射关系和连接关系
// 参数：pbs-Lane的protobuf数据列表
// 说明：使用并行处理提高初始化效率，分两阶段：创建对象和建立连接关系
func (m *LaneManager) Init(pbs []*mapv2.Lane) {
	m.lanes = parallel.GoMap(pbs, func(pb *mapv2.Lane) *Lane {
		return newLane(m.ctx, pb)
	})
	m.data = lo.SliceToMap(m.lanes, func(l *Lane) (int32, *Lane) {
		return l.id, l
	})
	parallel.GoFor(m.lanes, func(l *Lane) { l.initWithManager(m) })
}

// Get 根据ID获取Lane实例
// 功能：通过Lane ID查找对应的Lane对象，如果不存在则panic
// 参数：id-Lane的唯一标识符
// 返回：对应的Lane实例，如果不存在则panic
func (m *LaneManager) Get(id int32) entity.ILane {
	if lane, ok := m.data[id]; !ok {
		log.Panicf("no id %d in lane data", id)
		return nil
	} else {
		return lane
	}
}

// GetOrError 根据ID获取Lane实例（带错误处理）
// 功能：通过Lane ID查找对应的Lane对象，如果不存在则返回错误
// 参数：id-Lane的唯一标识符
// 返回：Lane实例和错误信息，如果不存在则返回nil和错误
func (m *LaneManager) GetOrError(id int32) (entity.ILane, error) {
	if lane, ok := m.data[id]; !ok {
		return nil, fmt.Errorf("no id %d in lane data", id)
	} else {
		return lane, nil
	}
}

// Prepare 准备阶段，处理所有Lane的准备工作
// 功能：对所有Lane执行准备阶段，处理车辆/行人列表的缓冲区操作
// 说明：使用并行处理提高性能，分两个阶段：prepare和prepare2
func (m *LaneManager) Prepare() {
	parallel.GoFor(m.lanes, func(l *Lane) { l.prepare() })
	parallel.GoFor(m.lanes, func(l *Lane) { l.prepare2() })
}

// Update 更新阶段，执行所有Lane的模拟逻辑
// 功能：对所有Lane执行更新阶段，处理车道状态更新和统计计算
// 说明：使用并行处理提高性能
func (m *LaneManager) Update() {
	parallel.GoFor(m.lanes, func(l *Lane) { l.update() })
}
