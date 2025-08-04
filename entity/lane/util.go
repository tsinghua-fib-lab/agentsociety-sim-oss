package lane

import (
	"sync"

	"github.com/tsinghua-fib-lab/agentsociety-sim-oss/utils/container"
)

// laneList 车道列表数据结构，用于管理车道上的车辆或行人
// 功能：提供线程安全的车辆/行人列表管理，支持缓冲式添加和删除操作
// 泛型参数：T-列表元素类型（必须实现IHasVAndLength接口），E-侧链数据类型
type laneList[T container.IHasVAndLength, E any] struct {
	list              *container.List[T, E]
	addBuffer         []*container.ListNode[T, E]
	addBufferMutex    sync.Mutex
	removeBuffer      []*container.ListNode[T, E]
	removeBufferMutex sync.Mutex
}

// newLaneList 创建新的车道列表实例
// 功能：初始化车道列表，设置基础数据结构和互斥锁
// 参数：id-列表标识符，用于调试和日志
// 返回：初始化完成的车道列表实例
func newLaneList[T container.IHasVAndLength, E any](id string) laneList[T, E] {
	return laneList[T, E]{
		list: &container.List[T, E]{
			ID: id,
		},
		addBuffer:         make([]*container.ListNode[T, E], 0),
		addBufferMutex:    sync.Mutex{},
		removeBuffer:      make([]*container.ListNode[T, E], 0),
		removeBufferMutex: sync.Mutex{},
	}
}

// prepare 准备阶段，处理缓冲区的添加和删除操作
// 功能：将缓冲区中的操作应用到主列表，清空缓冲区
// 说明：已处理为nil的情况，使用缓冲机制提高并发性能
func (l *laneList[T, E]) prepare() {
	if l == nil || l.list == nil {
		return
	}
	for _, v := range l.removeBuffer {
		l.list.Remove(v)
	}
	unsorted := l.list.PopUnsorted()
	l.list.Merge(append(l.addBuffer, unsorted...))
	l.removeBuffer = l.removeBuffer[:0]
	l.addBuffer = l.addBuffer[:0]
}

// add 添加节点到缓冲区
// 功能：将节点添加到添加缓冲区，延迟到prepare阶段实际插入列表
// 参数：node-要添加的节点
// 说明：使用互斥锁保证线程安全，如果节点已有父节点则panic
func (l *laneList[T, E]) add(node *container.ListNode[T, E]) {
	if node.Parent() != nil {
		log.Panic("add node who has parent")
	}
	l.addBufferMutex.Lock()
	l.addBuffer = append(l.addBuffer, node)
	l.addBufferMutex.Unlock()
}

// remove 从缓冲区移除节点
// 功能：将节点添加到删除缓冲区，延迟到prepare阶段实际从列表移除
// 参数：node-要移除的节点
// 说明：使用互斥锁保证线程安全，验证节点的父节点关系
func (l *laneList[T, E]) remove(node *container.ListNode[T, E]) {
	if node.Parent() != l.list {
		log.Panicf("remove node %v (parent=%v) from wrong parent %+v", node, node.Parent(), l.list)
	}
	l.removeBufferMutex.Lock()
	l.removeBuffer = append(l.removeBuffer, node)
	l.removeBufferMutex.Unlock()
}
