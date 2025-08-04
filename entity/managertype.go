package entity

import (
	mapv2 "git.fiblab.net/sim/protos/v2/go/city/map/v2"
	personv2 "git.fiblab.net/sim/protos/v2/go/city/person/v2"
	"git.fiblab.net/sim/syncer/v3"
)

// Manager依赖倒置

// entity/lane/manager.go的依赖倒置
type ILaneManager interface {
	Init(pbs []*mapv2.Lane) // 初始化

	// 输入Lane ID，查找Lane，如果不存在则panic
	Get(id int32) ILane
	// 输入Lane ID，查找Lane，如果不存在则返回error
	GetOrError(id int32) (ILane, error)

	Prepare() // 准备阶段
	Update()  // 更新阶段
}

// entity/aoi/manager.go的依赖倒置
type IAoiManager interface {
	Init(
		pbs []*mapv2.Aoi,
		laneManager ILaneManager,
	) // 初始化

	// 输入Aoi ID，查找Aoi，如果不存在则panic
	Get(id int32) IAoi
	// 输入Aoi ID，查找Aoi，如果不存在则返回error
	GetOrError(id int32) (IAoi, error)

	Prepare()          // 准备阶段
	Update(dt float64) // 更新阶段
}

// entity/road/manager.go的依赖倒置
type IRoadManager interface {
	Init(pbs []*mapv2.Road, laneManager ILaneManager)   // 初始化
	InitAfterJunction(junctionManager IJunctionManager) // 初始化所有Road的Junction关系

	// 输入Road ID，查找Road，如果不存在则panic
	Get(id int32) IRoad
	// 输入Road ID，查找Road，如果不存在则返回error
	GetOrError(id int32) (IRoad, error)
}

// entity/junction/manager.go的依赖倒置
type IJunctionManager interface {
	Init(pbs []*mapv2.Junction, laneManager ILaneManager, roadManager IRoadManager) // 初始化
	Register(sidecar *syncer.Sidecar)                                               // 注册到Sidecar

	// 输入Junction ID，查找Junction，如果不存在则panic
	Get(id int32) IJunction
	// 输入Junction ID，查找Junction，如果不存在则返回error
	GetOrError(id int32) (IJunction, error)

	Prepare()          // 准备阶段
	Update(dt float64) // 更新阶段                                         // 产生所有Junction的simple输出
}

// entity/person/manager.go的依赖倒置
type IPersonManager interface {
	// 初始化
	Init(
		pbs []*personv2.Person,
		h *mapv2.Header,
		aoiManager IAoiManager,
		laneManager ILaneManager,
	)
	// 注册到Sidecar
	Register(sidecar *syncer.Sidecar)

	// 输入Person ID，查找Person，如果不存在则panic
	Get(id int32) IPerson
	// 输入Person ID，查找Person，如果不存在则返回error
	GetOrError(id int32) (IPerson, error)

	PrepareNode()      // 准备阶段：链表节点更新
	Prepare()          // 准备阶段：snapshot更新
	Update(dt float64) // 更新阶段
}
