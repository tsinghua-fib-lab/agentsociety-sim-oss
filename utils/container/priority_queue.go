package container

import "container/heap"

// item 优先队列中单个元素
// 功能：表示优先队列中的一个元素，包含值和优先级信息
// 说明：实现了heap.Interface所需的索引管理功能
type item[T any] struct {
	Value    T       // 元素的值（任意类型）
	Priority float64 // 元素在队列中的优先级（越小越优先）
	// 索引由 update 方法使用，并由 heap.Interface 方法维护。
	index int // 项在堆中的索引。
}

// priorityQueue 优先队列实现了 heap.Interface 并保存了元素
// 功能：内部优先队列实现，基于Go标准库的heap包
// 说明：使用泛型支持任意类型的元素，优先级为float64类型
type priorityQueue[T any] []*item[T]

// Len 返回队列长度
// 功能：实现heap.Interface接口，返回队列中元素的数量
// 返回：队列长度
func (pq priorityQueue[T]) Len() int { return len(pq) }

// Less 比较两个元素的优先级
// 功能：实现heap.Interface接口，定义元素间的优先级比较规则
// 参数：i,j-要比较的两个元素索引
// 返回：true表示i的优先级高于j
// 说明：使用小于号，使得Pop方法返回最低优先级的项（最小堆）
func (pq priorityQueue[T]) Less(i, j int) bool {
	// 我们希望 Pop 方法返回最低优先级的项，因此这里使用小于号。
	return pq[i].Priority < pq[j].Priority
}

// Swap 交换两个元素的位置
// 功能：实现heap.Interface接口，交换队列中两个元素的位置
// 参数：i,j-要交换的两个元素索引
// 说明：交换元素位置后同时更新元素的索引信息
func (pq priorityQueue[T]) Swap(i, j int) {
	pq[i], pq[j] = pq[j], pq[i]
	pq[i].index = i
	pq[j].index = j
}

// Push 向队列中添加元素
// 功能：实现heap.Interface接口，向队列末尾添加新元素
// 参数：x-要添加的元素（类型为*item[T]）
// 说明：添加元素时自动设置正确的索引值
func (pq *priorityQueue[T]) Push(x any) {
	n := len(*pq)
	item := x.(*item[T])
	item.index = n
	*pq = append(*pq, item)
}

// Pop 从队列中移除并返回最后一个元素
// 功能：实现heap.Interface接口，移除并返回队列末尾的元素
// 返回：被移除的元素
// 说明：移除元素时清理索引信息，避免内存泄漏
func (pq *priorityQueue[T]) Pop() any {
	old := *pq
	n := len(old)
	item := old[n-1]
	old[n-1] = nil  // 避免内存泄漏
	item.index = -1 // 为了安全起见
	*pq = old[0 : n-1]
	return item
}

// PriorityQueue 优先队列
// 功能：提供优先队列的公共接口，封装内部堆实现
// 说明：支持任意类型的元素，基于优先级进行排序和访问
type PriorityQueue[T any] struct {
	queue priorityQueue[T] // 内部优先队列实现
}

// NewPriorityQueue 创建优先队列
// 功能：初始化一个新的优先队列实例
// 返回：新创建的优先队列指针
// 说明：初始化内部队列结构，准备进行优先队列操作
func NewPriorityQueue[T any]() *PriorityQueue[T] {
	return &PriorityQueue[T]{queue: make(priorityQueue[T], 0)}
}

// Len 获取当前队列长度
// 功能：返回队列中元素的数量
// 返回：队列长度
func (q *PriorityQueue[T]) Len() int {
	return len(q.queue)
}

// First 获取第一个元素（优先级数值最小的元素）
// 功能：返回队列中优先级最高的元素（优先级值最小）
// 返回：优先级最高的元素值
// 说明：不移除元素，仅查看队列顶部的元素
func (q *PriorityQueue[T]) First() T {
	return q.queue[0].Value
}

// Push 加入元素（简单添加）
// 功能：向队列中添加新元素，但不维护堆结构
// 参数：value-要添加的元素值，priority-元素优先级
// 说明：添加后需要调用Heapify()来重新构建堆结构
func (q *PriorityQueue[T]) Push(value T, priority float64) {
	q.queue = append(q.queue, &item[T]{
		Value:    value,
		Priority: priority,
	})
}

// Heapify 重新构建堆
// 功能：将队列重新构建为有效的堆结构
// 说明：在批量添加元素后调用，确保队列满足堆的性质
func (q *PriorityQueue[T]) Heapify() {
	heap.Init(&q.queue)
}

// HeapPush 加入元素（堆操作）
// 功能：向优先队列中添加新元素，并维护堆结构
// 参数：value-要添加的元素值，priority-元素优先级
// 说明：使用堆操作添加元素，自动维护队列的堆性质
func (q *PriorityQueue[T]) HeapPush(value T, priority float64) {
	heap.Push(&q.queue, &item[T]{
		Value:    value,
		Priority: priority,
	})
}

// HeapPop 弹出元素（堆操作）
// 功能：从优先队列中移除并返回优先级最高的元素
// 返回：value-元素值，priority-元素优先级
// 说明：使用堆操作移除元素，自动维护队列的堆性质
func (q *PriorityQueue[T]) HeapPop() (value T, priority float64) {
	item := heap.Pop(&q.queue).(*item[T])
	return item.Value, item.Priority
}
