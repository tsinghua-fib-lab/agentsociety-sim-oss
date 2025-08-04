package person

import (
	"fmt"
	"sync"

	"git.fiblab.net/general/common/v2/parallel"
	mapv2 "git.fiblab.net/sim/protos/v2/go/city/map/v2"
	personv2 "git.fiblab.net/sim/protos/v2/go/city/person/v2"
	"git.fiblab.net/sim/protos/v2/go/city/person/v2/personv2connect"
	"github.com/samber/lo"
	"github.com/tsinghua-fib-lab/agentsociety-sim-oss/entity"
	"github.com/tsinghua-fib-lab/agentsociety-sim-oss/entity/person/route"
	"github.com/tsinghua-fib-lab/agentsociety-sim-oss/utils/container"
)

// GlobalRuntime 全局运行时数据结构
// 功能：管理全局运行时数据，包括完成行程数、总行驶时间、总行驶距离
type GlobalRuntime struct {
	NumCompletedTrips int32   // 已完成的行程
	TravelTime        float64 // 总行驶时间
	TravelDistance    float64 // 总行驶距离
}

// PersonManager Person管理器
// 功能：管理所有Person实体，提供创建、查找、初始化、更新等功能，支持多种控制模式
type PersonManager struct {
	personv2connect.UnimplementedPersonServiceHandler

	ctx entity.ITaskContext

	data map[int32]*Person

	// 有计算、输出需求的person
	persons *container.IncrementalArray[*Person]

	personInserted      []*Person // 新加入的人
	personInsertedMutex sync.Mutex
	nextPersonID        int32

	snapshot, runtime GlobalRuntime
	runtimeMtx        sync.Mutex
}

// NewManager 创建Person管理器实例
// 功能：初始化Person管理器，创建内部数据结构和控制通道
// 参数：ctx-任务上下文
// 返回：新创建的Person管理器实例
func NewManager(ctx entity.ITaskContext) *PersonManager {
	m := &PersonManager{
		ctx:                 ctx,
		data:                make(map[int32]*Person),
		persons:             container.NewIncrementalArray[*Person](),
		personInserted:      make([]*Person, 0),
		personInsertedMutex: sync.Mutex{},
		nextPersonID:        10000000,
	}
	return m
}

// Init 初始化所有Person
// 功能：根据protobuf数据初始化所有Person对象，建立ID映射关系
// 参数：pbs-Person的protobuf数据列表，h-地图头信息，aoiManager-AOI管理器，laneManager-车道管理器
// 说明：使用并行处理提高初始化效率，预分配各种类型的Person列表
func (m *PersonManager) Init(
	pbs []*personv2.Person,
	h *mapv2.Header,
	aoiManager entity.IAoiManager,
	laneManager entity.ILaneManager,
) {
	m.persons = container.NewIncrementalArray[*Person]()
	persons := parallel.GoMap(pbs, func(pb *personv2.Person) *Person {
		p := newPerson(m.ctx, m, pb)
		m.persons.Add(p)
		return p
	})
	m.data = lo.SliceToMap(persons, func(p *Person) (int32, *Person) {
		return p.id, p
	})
	m.nextPersonID = lo.Max(lo.Keys(m.data)) + 1
}

// Get 根据ID获取Person实例
// 功能：通过Person ID查找对应的Person对象，如果不存在则panic
// 参数：id-Person的唯一标识符
// 返回：对应的Person实例，如果不存在则panic
func (m *PersonManager) Get(id int32) entity.IPerson {
	if p, ok := m.data[id]; !ok {
		log.Panicf("no id %d in person data", id)
		return nil
	} else {
		return p
	}
}

// GetOrError 根据ID获取Person实例（带错误处理）
// 功能：通过Person ID查找对应的Person对象，如果不存在则返回错误
// 参数：id-Person的唯一标识符
// 返回：Person实例和错误信息，如果不存在则返回nil和错误
func (m *PersonManager) GetOrError(id int32) (entity.IPerson, error) {
	if p, ok := m.data[id]; !ok {
		return nil, fmt.Errorf("no id %d in person data", id)
	} else {
		return p, nil
	}
}

// add 添加新的Person到管理器
// 功能：动态添加新的Person，支持ID自动分配
// 参数：pb-Person的protobuf数据
// 返回：新创建的Person实例
// 说明：使用互斥锁保证线程安全，支持外部指定ID或自动分配ID
func (m *PersonManager) add(pb *personv2.Person) *Person {
	m.personInsertedMutex.Lock()
	defer m.personInsertedMutex.Unlock()
	if pb.Id != 0 {
		// 提供id
		if _, ok := m.data[pb.Id]; ok {
			log.Panicf("Person ID %v already exists!", pb.Id)
		}
	} else {
		// 未提供id 模拟器分发
		pb.Id = m.nextPersonID
		m.nextPersonID++
	}
	p := newPerson(m.ctx, m, pb)
	m.personInserted = append(m.personInserted, p)
	return p
}

// 准备阶段：链表节点更新
func (m *PersonManager) PrepareNode() {
	// 新人加入
	for _, newP := range m.personInserted {
		if _, ok := m.data[newP.ID()]; ok {
			log.Panic("Person: same id between new person and existed person")
		}
		m.data[newP.ID()] = newP
	}
	m.personInserted = []*Person{}

	// data prepare
	// 最好不要并行处理，因为共用index，如果一个人同时从车辆中删去又加入行人，可能有问题
	m.persons.Prepare()

	parallel.GoFor(m.persons.Data(), func(p *Person) { p.prepareNode() })
}

// 准备阶段：snapshot更新
func (m *PersonManager) Prepare() {
	parallel.GoFor(m.persons.Data(), func(p *Person) {
		p.prepare()
	})
	m.snapshot = m.runtime
	log.Debug("PersonManager: prepare done")
}

// 更新阶段
func (m *PersonManager) Update(dt float64) {
	parallel.GoFor(m.persons.Data(), func(p *Person) { p.update(dt) })
	route.CallbackWaitGroup.Wait()
}

// recordRunning 记录在路上的人车
// 功能：记录在路上的人车，更新全局运行时数据
func (m *PersonManager) recordRunning(dt float64, ds float64) {
	m.runtimeMtx.Lock()
	defer m.runtimeMtx.Unlock()
	m.runtime.TravelTime += dt
	m.runtime.TravelDistance += ds
}

// recordPedestrianTripEnd 记录行程结束
// 功能：记录行程结束，更新全局运行时数据
func (m *PersonManager) recordTripEnd(p *Person) {
	m.runtimeMtx.Lock()
	defer m.runtimeMtx.Unlock()
	m.runtime.NumCompletedTrips++
}
