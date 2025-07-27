package container

import (
	"sync"
)

// IIncrementalItem 支持增量更新的元素接口
// 功能：定义支持增量更新的元素必须实现的方法
// 说明：用于增量数组中元素的索引管理，确保元素能够正确跟踪自己在数组中的位置
type IIncrementalItem interface {
	Index() int         // 获取元素的索引
	SetIndex(index int) // 设置元素的索引
}

// IncrementalItemBase 增量元素基类
// 功能：提供增量元素的基础实现，包含索引管理功能
// 说明：可以作为其他结构体的嵌入字段，快速实现IIncrementalItem接口
type IncrementalItemBase struct {
	index int // 元素在数组中的索引
}

// Index 获取元素的索引
// 功能：返回元素在数组中的当前位置索引
// 返回：元素索引
func (b *IncrementalItemBase) Index() int {
	return b.index
}

// SetIndex 设置元素的索引
// 功能：更新元素在数组中的索引位置
// 参数：index-新的索引值
func (b *IncrementalItemBase) SetIndex(index int) {
	b.index = index
}

// IncrementalArray 增量数组，支持增量维护元素的数组
// 功能：提供高效的数组操作，支持批量添加和删除元素
// 说明：使用延迟更新机制，在Prepare时统一执行所有增量操作，提高性能
type IncrementalArray[T IIncrementalItem] struct {
	data        []T        // 主数据数组
	add         []T        // 待添加的元素列表
	remove      []T        // 待删除的元素列表
	addMutex    sync.Mutex // 添加操作的互斥锁
	removeMutex sync.Mutex // 删除操作的互斥锁
}

// NewIncrementalArray 创建增量数组
// 功能：初始化一个新的增量数组实例
// 返回：新创建的增量数组指针
// 说明：初始化所有内部数据结构，准备进行增量操作
func NewIncrementalArray[T IIncrementalItem]() *IncrementalArray[T] {
	return &IncrementalArray[T]{
		data:   make([]T, 0),
		add:    make([]T, 0),
		remove: make([]T, 0),
	}
}

// Len 获取当前数组长度
// 功能：返回主数据数组的当前长度
// 返回：数组长度
func (a *IncrementalArray[T]) Len() int {
	return len(a.data)
}

// Data 获取原始数据
// 功能：返回主数据数组的副本
// 返回：数据数组的副本
// 说明：返回的是当前已应用所有增量操作的数据
func (a *IncrementalArray[T]) Data() []T {
	return a.data
}

// Add 增加元素（等到Prepare时才会真正增加）
// 功能：将元素添加到待添加列表中
// 参数：value-要添加的元素
// 说明：使用互斥锁保护并发安全，元素不会立即添加到主数组中
func (a *IncrementalArray[T]) Add(value T) {
	a.addMutex.Lock()
	defer a.addMutex.Unlock()
	a.add = append(a.add, value)
}

// Remove 删除元素（等到Prepare时才会真正删除）
// 功能：将元素添加到待删除列表中
// 参数：value-要删除的元素
// 说明：使用互斥锁保护并发安全，元素不会立即从主数组中删除
func (a *IncrementalArray[T]) Remove(value T) {
	a.removeMutex.Lock()
	defer a.removeMutex.Unlock()
	a.remove = append(a.remove, value)
}

// Prepare 执行增量操作
// 功能：统一执行所有待处理的添加和删除操作
// 算法说明：
// 1. 比较添加和删除操作的数量
// 2. 如果添加 >= 删除：
//   - 先处理删除操作，用添加的元素替换被删除的元素
//   - 将剩余的添加元素追加到数组末尾
//   - 更新所有元素的索引
//
// 3. 如果删除 > 添加：
//   - 先处理添加操作，用添加的元素替换被删除的元素
//   - 从数组末尾移动元素填充剩余的删除位置
//   - 更新所有元素的索引
//
// 4. 清空待处理列表
// 说明：使用延迟更新机制，批量处理所有增量操作，提高性能
func (a *IncrementalArray[T]) Prepare() {
	// 增 >= 删
	if len(a.add) >= len(a.remove) {
		for i, x := range a.remove {
			ind := x.Index()
			a.data[ind] = a.add[i]
			a.data[ind].SetIndex(ind)
		}
		l1 := len(a.remove)
		l2 := len(a.add) - l1
		for i := 0; i < l2; i++ {
			a.add[l1+i].SetIndex(len(a.data) + i)
		}
		a.data = append(a.data, a.add[len(a.remove):]...)
	} else {
		// 删 > 增
		for i, x := range a.add {
			ind := a.remove[i].Index()
			a.data[ind] = x
			a.data[ind].SetIndex(ind)
		}
		l1 := len(a.add)
		l2 := len(a.remove) - l1
		l3 := len(a.data) - l2
		for i := 0; i < l2; i++ {
			// 从后面拿一项填过来
			ind := a.remove[l1+i].Index()
			a.data[ind] = a.data[l3+i]
			a.data[ind].SetIndex(ind)
		}
		a.data = a.data[:l3]
	}

	a.add = []T{}
	a.remove = []T{}
}
