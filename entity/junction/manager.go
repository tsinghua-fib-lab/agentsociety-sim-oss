package junction

import (
	"fmt"

	"git.fiblab.net/general/common/v2/parallel"
	mapv2 "git.fiblab.net/sim/protos/v2/go/city/map/v2"
	mapv2connect "git.fiblab.net/sim/protos/v2/go/city/map/v2/mapv2connect"
	"github.com/samber/lo"
	"github.com/tsinghua-fib-lab/agentsociety-sim-oss/entity"
)

// Junction管理器
type JunctionManager struct {
	mapv2connect.UnimplementedTrafficLightServiceHandler
	mapv2connect.UnimplementedJunctionServiceHandler

	ctx entity.ITaskContext

	data      map[int32]*Junction
	junctions []*Junction

	lanesInJunction []entity.ILane
}

// NewManager 创建Junction管理器实例
// 功能：初始化Junction管理器，创建内部数据结构
// 参数：ctx-任务上下文
// 返回：新创建的Junction管理器实例
func NewManager(ctx entity.ITaskContext) *JunctionManager {
	return &JunctionManager{
		ctx:             ctx,
		data:            make(map[int32]*Junction),
		junctions:       make([]*Junction, 0),
		lanesInJunction: make([]entity.ILane, 0),
	}
}

// Init 初始化所有Junction及其信控
// 功能：根据protobuf数据初始化所有Junction对象，建立车道映射关系
// 参数：pbs-Junction的protobuf数据列表，laneManager-车道管理器，roadManager-道路管理器
// 说明：使用并行处理提高初始化效率
func (m *JunctionManager) Init(pbs []*mapv2.Junction, laneManager entity.ILaneManager, roadManager entity.IRoadManager) {
	m.junctions = parallel.GoMap(pbs, func(pb *mapv2.Junction) *Junction {
		return newJunction(m.ctx, pb, laneManager, roadManager)
	})
	m.data = lo.SliceToMap(m.junctions, func(j *Junction) (int32, *Junction) {
		return j.id, j
	})
	m.lanesInJunction = make([]entity.ILane, 0)
	for _, j := range m.junctions {
		m.lanesInJunction = append(m.lanesInJunction, lo.Values(j.lanes)...)
	}
}

// Get 根据ID获取Junction实例
// 功能：通过Junction ID查找对应的Junction对象，如果不存在则panic
// 参数：id-Junction的唯一标识符
// 返回：对应的Junction实例，如果不存在则panic
func (m *JunctionManager) Get(id int32) entity.IJunction {
	if junction, ok := m.data[id]; !ok {
		log.Panicf("no id %d in junction data", id)
		return nil
	} else {
		return junction
	}
}

// GetOrError 根据ID获取Junction实例（带错误处理）
// 功能：通过Junction ID查找对应的Junction对象，如果不存在则返回错误
// 参数：id-Junction的唯一标识符
// 返回：Junction实例和错误信息，如果不存在则返回nil和错误
func (m *JunctionManager) GetOrError(id int32) (entity.IJunction, error) {
	if junction, ok := m.data[id]; !ok {
		return nil, fmt.Errorf("no id %d in junction data", id)
	} else {
		return junction, nil
	}
}

// Prepare 准备阶段，处理所有Junction的准备工作
// 功能：对所有Junction执行准备阶段，处理信号灯的准备工作
// 说明：使用并行处理提高性能
func (m *JunctionManager) Prepare() {
	parallel.GoFor(m.junctions, func(j *Junction) { j.prepare() })
}

// Update 更新阶段，执行所有Junction的模拟逻辑
// 功能：对所有Junction执行更新阶段，执行信号灯的更新逻辑
// 参数：dt-时间步长
// 说明：使用并行处理提高性能
func (m *JunctionManager) Update(dt float64) {
	parallel.GoFor(m.junctions, func(j *Junction) { j.update(dt) })
}
