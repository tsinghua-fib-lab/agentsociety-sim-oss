package road

import (
	"fmt"

	"git.fiblab.net/general/common/v2/parallel"
	mapv2 "git.fiblab.net/sim/protos/v2/go/city/map/v2"
	"git.fiblab.net/sim/protos/v2/go/city/map/v2/mapv2connect"
	"github.com/tsinghua-fib-lab/agentsociety-sim-oss/entity"

	"github.com/samber/lo"
)

// RoadManager Road管理器
// 功能：管理所有Road实体，提供创建、查找、初始化、输出等功能
type RoadManager struct {
	mapv2connect.UnimplementedRoadServiceHandler

	ctx entity.ITaskContext

	data  map[int32]*Road
	roads []*Road
}

// NewManager 创建Road管理器实例
// 功能：初始化Road管理器，创建内部数据结构
// 参数：ctx-任务上下文
// 返回：新创建的Road管理器实例
func NewManager(ctx entity.ITaskContext) *RoadManager {
	return &RoadManager{
		ctx:   ctx,
		data:  make(map[int32]*Road),
		roads: make([]*Road, 0),
	}
}

// Init 初始化所有Road
// 功能：根据protobuf数据初始化所有Road对象，建立ID映射关系
// 参数：pbs-Road的protobuf数据列表，laneManager-车道管理器
// 说明：使用并行处理提高初始化效率
func (m *RoadManager) Init(pbs []*mapv2.Road, laneManager entity.ILaneManager) {
	m.roads = parallel.GoMap(pbs, func(pb *mapv2.Road) *Road {
		return newRoad(m.ctx, pb, laneManager)
	})
	m.data = lo.SliceToMap(m.roads, func(r *Road) (int32, *Road) {
		return r.id, r
	})
}

// InitAfterJunction 初始化所有Road的Junction关系
// 功能：在所有Junction初始化完成后，设置Road的前驱和后继路口连接关系
// 参数：junctionManager-Junction管理器
// 说明：使用并行处理提高初始化效率
func (m *RoadManager) InitAfterJunction(junctionManager entity.IJunctionManager) {
	parallel.GoFor(m.roads, func(r *Road) { r.initAfterJunction(junctionManager) })
}

// Get 根据ID获取Road实例
// 功能：通过Road ID查找对应的Road对象，如果不存在则panic
// 参数：id-Road的唯一标识符
// 返回：对应的Road实例，如果不存在则panic
func (m *RoadManager) Get(id int32) entity.IRoad {
	if road, ok := m.data[id]; !ok {
		log.Panicf("no id %d in road data", id)
		return nil
	} else {
		return road
	}
}

// GetOrError 根据ID获取Road实例（带错误处理）
// 功能：通过Road ID查找对应的Road对象，如果不存在则返回错误
// 参数：id-Road的唯一标识符
// 返回：Road实例和错误信息，如果不存在则返回nil和错误
func (m *RoadManager) GetOrError(id int32) (entity.IRoad, error) {
	if road, ok := m.data[id]; !ok {
		return nil, fmt.Errorf("no id %d in road data", id)
	} else {
		return road, nil
	}
}
