package container

import (
	"fmt"
	"log"
)

// IHasVAndLength 具有速度和长度属性的接口
// 功能：定义车辆和行人作为链表元素时需要的关键信息接口
// 说明：便于在链表中快速查找和访问元素的速度和长度信息
type IHasVAndLength interface {
	V() float64      // 获取速度
	Length() float64 // 获取长度
}

// ListNode 双向链表中的节点
// 功能：表示双向链表中的一个节点，包含键值对和额外信息
// 说明：支持泛型，可以存储任意类型的值和额外信息
type ListNode[T IHasVAndLength, E any] struct {
	parent     *List[T, E]     // 所属链表
	prev, next *ListNode[T, E] // 前驱和后继节点
	S          float64         // 键值（通常是位置信息）
	Value      T               // 主要值
	Extra      E               // 额外信息
}

// String 获取节点的字符串表示
// 功能：将节点信息格式化为可读的字符串
// 返回：格式化的节点信息字符串
func (n *ListNode[T, E]) String() string {
	return fmt.Sprintf("Node{Key:%v, Value:%+v, Extra:%+v}", n.S, n.Value, n.Extra)
}

// Prev 获取节点的前一个节点
// 功能：返回链表中的前驱节点
// 返回：前驱节点指针，如果是第一个节点则返回nil
func (n *ListNode[T, E]) Prev() *ListNode[T, E] {
	return n.prev
}

// Next 获取节点的下一个节点
// 功能：返回链表中的后继节点
// 返回：后继节点指针，如果是最后一个节点则返回nil
func (n *ListNode[T, E]) Next() *ListNode[T, E] {
	return n.next
}

// Parent 获取节点所在的链表
// 功能：返回节点所属的链表对象
// 返回：链表指针
func (n *ListNode[T, E]) Parent() *List[T, E] {
	return n.parent
}

// V 获取节点值的速度
// 功能：简化代码的特殊函数，直接获取Value的速度
// 返回：速度值（米/秒）
func (n *ListNode[T, E]) V() float64 {
	return n.Value.V()
}

// L 获取节点值的长度
// 功能：简化代码的特殊函数，直接获取Value的长度
// 返回：长度值（米）
func (n *ListNode[T, E]) L() float64 {
	return n.Value.Length()
}

// InsertBefore 在节点前插入新节点
// 功能：在当前节点之前插入一个新节点
// 参数：add-要插入的新节点
// 算法说明：
// 1. 检查新节点是否已经在其他链表中
// 2. 设置新节点的父链表和前后指针
// 3. 更新当前节点和前驱节点的指针
// 4. 如果新节点是第一个节点，更新链表头指针
// 5. 增加链表长度计数
func (n *ListNode[T, E]) InsertBefore(add *ListNode[T, E]) {
	if add.parent != nil {
		log.Panic("push back node who already in list")
	}
	add.parent = n.parent
	add.next = n
	add.prev = n.prev
	n.prev = add
	if add.prev != nil {
		add.prev.next = add
	} else {
		add.parent.head = add
	}
	n.parent.length++
}

// InsertAfter 在节点后插入新节点
// 功能：在当前节点之后插入一个新节点
// 参数：add-要插入的新节点
// 算法说明：
// 1. 检查新节点是否已经在其他链表中
// 2. 设置新节点的父链表和前后指针
// 3. 更新当前节点和后继节点的指针
// 4. 如果新节点是最后一个节点，更新链表尾指针
// 5. 增加链表长度计数
func (n *ListNode[T, E]) InsertAfter(add *ListNode[T, E]) {
	if add.parent != nil {
		log.Panic("push back node who already in list")
	}
	add.parent = n.parent
	add.prev = n
	add.next = n.next
	n.next = add
	if add.next != nil {
		add.next.prev = add
	} else {
		add.parent.tail = add
	}
	n.parent.length++
}

// List 双向链表
// 功能：实现一个通用的双向链表数据结构
// 说明：支持泛型，专门用于存储具有速度和长度属性的对象（如车辆、行人）
type List[T IHasVAndLength, E any] struct {
	ID         string          // 链表标识符
	head, tail *ListNode[T, E] // 头尾节点指针
	length     int             // 链表长度
}

// String 获取链表的字符串表示
// 功能：将链表信息格式化为可读的字符串
// 返回：格式化的链表信息字符串
func (l *List[T, E]) String() string {
	return fmt.Sprintf("List{ID:%v}", l.ID)
}

// Keys 获取双向链表中所有节点的键值
// 功能：返回链表中所有节点的键值数组
// 返回：键值数组
// 算法说明：
// 1. 创建与链表长度相同的数组
// 2. 从头节点开始遍历链表
// 3. 将每个节点的键值存入数组
func (l *List[T, E]) Keys() []float64 {
	keys := make([]float64, l.length)
	for i, node := 0, l.head; node != nil; node = node.next {
		keys[i] = node.S
		i++
	}
	return keys
}

// Values 获取双向链表中所有节点的值
// 功能：返回链表中所有节点的值数组
// 返回：值数组
// 算法说明：
// 1. 创建与链表长度相同的数组
// 2. 从头节点开始遍历链表
// 3. 将每个节点的值存入数组
func (l *List[T, E]) Values() []T {
	values := make([]T, l.length)
	for i, node := 0, l.head; node != nil; i, node = i+1, node.next {
		values[i] = node.Value
	}
	return values
}

// Len 获取双向链表长度
// 功能：返回链表中的节点数量
// 返回：链表长度
func (l *List[T, E]) Len() int {
	return l.length
}

// PushFront 向链表头部插入节点
// 功能：在链表头部添加一个新节点
// 参数：add-要插入的新节点
// 算法说明：
// 1. 检查新节点是否已经在其他链表中
// 2. 如果链表为空，直接设置为头尾节点
// 3. 如果链表不为空，在头节点前插入新节点
// 4. 更新头节点指针
func (l *List[T, E]) PushFront(add *ListNode[T, E]) {
	if add.parent != nil {
		log.Panic("push back node who already in list")
	}
	add.next = nil
	add.prev = nil
	if l.head == nil {
		add.parent = l
		l.head = add
		l.tail = add
		l.length++
	} else {
		// length++和add.parent在InsertBefore中处理
		l.head.InsertBefore(add)
		l.head = add
	}
}

// PushBack 向链表尾部插入节点
// 功能：在链表尾部添加一个新节点
// 参数：add-要插入的新节点
// 算法说明：
// 1. 检查新节点是否已经在其他链表中
// 2. 如果链表为空，直接设置为头尾节点
// 3. 如果链表不为空，在尾节点后插入新节点
// 4. 更新尾节点指针
func (l *List[T, E]) PushBack(add *ListNode[T, E]) {
	if add.parent != nil {
		log.Panic("push back node who already in list")
	}
	add.next = nil
	add.prev = nil
	if l.tail == nil {
		add.parent = l
		l.head = add
		l.tail = add
		l.length++
	} else {
		// length++和add.parent在InsertAfter中处理
		l.tail.InsertAfter(add)
		l.tail = add
	}
}

// Remove 从链表中移除节点
// 功能：从链表中删除指定的节点
// 参数：node-要删除的节点
// 算法说明：
// 1. 检查节点是否属于当前链表
// 2. 更新前驱节点的后继指针
// 3. 更新后继节点的前驱指针
// 4. 如果删除的是头节点，更新头指针
// 5. 如果删除的是尾节点，更新尾指针
// 6. 清空被删除节点的指针
// 7. 减少链表长度计数
func (l *List[T, E]) Remove(node *ListNode[T, E]) {
	if node.parent != l {
		log.Panic("remove node from wrong list")
	}
	if node.prev != nil {
		node.prev.next = node.next
	} else {
		l.head = node.next
	}
	if node.next != nil {
		node.next.prev = node.prev
	} else {
		l.tail = node.prev
	}
	node.prev = nil
	node.next = nil
	node.parent = nil
	l.length--
}

// First 获取链表头部节点
// 功能：返回链表的第一个节点
// 返回：头节点指针，如果链表为空则返回nil
func (l *List[T, E]) First() *ListNode[T, E] {
	return l.head
}

// Last 获取链表尾部节点
// 功能：返回链表的最后一个节点
// 返回：尾节点指针，如果链表为空则返回nil
func (l *List[T, E]) Last() *ListNode[T, E] {
	return l.tail
}

// PopUnsorted 移除逆序节点
// 功能：移除链表中键值逆序的节点（前驱节点的键值大于当前节点）
// 返回：被移除的逆序节点数组
// 算法说明：
// 1. 从头节点开始遍历链表
// 2. 检查每个节点与其前驱节点的键值关系
// 3. 如果前驱节点的键值大于当前节点，则移除当前节点
// 4. 将移除的节点添加到结果数组中
// 说明：用于维护链表的顺序性，确保键值按升序排列
func (l *List[T, E]) PopUnsorted() (unsorted []*ListNode[T, E]) {
	for node := l.head; node != nil; {
		next := node.next
		if node.prev != nil && node.prev.S > node.S {
			l.Remove(node)
			unsorted = append(unsorted, node)
		}
		node = next
	}
	return unsorted
}

// 批量插入节点
func (l *List[T, E]) Merge(adds []*ListNode[T, E]) {
	// 1. sort array (可优化)
	for i := 0; i < len(adds)-1; i++ {
		for j := i + 1; j < len(adds); j++ {
			if adds[i].S > adds[j].S {
				adds[i], adds[j] = adds[j], adds[i]
			}
		}
	}
	// 2. merge sort
	node := l.head
	for _, add := range adds {
		for node != nil && node.S < add.S {
			node = node.next
		}
		if node != nil {
			node.InsertBefore(add)
		} else {
			l.PushBack(add)
		}
	}
}
