// 随机数引擎，包装了golang.org/x/exp/rand，提供了一些常用的随机数生成方法
package randengine

import (
	"flag"
	"log"
	"sync"

	"golang.org/x/exp/rand"
)

var (
	seedOffset = flag.Uint64("rand.seed_offset", 0, "seed offset") // 种子偏移量，用于调整随机数生成
)

// Engine 随机数引擎
// 功能：提供高质量的随机数生成功能，支持多种分布和线程安全操作
// 说明：基于golang.org/x/exp/rand库，提供更丰富的随机数生成接口
type Engine struct {
	*rand.Rand            // 底层随机数生成器
	mtx        sync.Mutex // 互斥锁，用于线程安全操作
}

// New 创建随机数引擎
// 功能：初始化一个新的随机数引擎实例
// 参数：seed-随机数种子
// 返回：随机数引擎指针
// 算法说明：
// 1. 创建随机数源：使用提供的种子创建rand.NewSource
// 2. 应用种子偏移量：将种子偏移量加到基础种子上
// 3. 创建随机数生成器：使用调整后的种子创建rand.Rand
// 4. 初始化引擎：包装随机数生成器和互斥锁
// 说明：种子偏移量允许在不修改代码的情况下调整随机数序列
func New(seed uint64) *Engine {
	return &Engine{Rand: rand.New(rand.NewSource(seed + *seedOffset))}
}

// DiscreteDistribution 按给定概率分布生成随机数（非线程安全）
// 功能：根据权重数组生成离散分布的随机数
// 参数：weight-权重数组，每个元素表示对应索引的概率权重
// 返回：随机生成的索引值（0到len(weight)-1）
// 算法说明：
// 1. 计算总权重：遍历权重数组计算总和
// 2. 生成随机数：在[0, 总权重)范围内生成随机数
// 3. 累积概率：遍历权重数组，累积概率直到超过随机数
// 4. 返回索引：返回第一个累积概率超过随机数的索引
// 5. 错误处理：如果算法异常则panic
// 说明：使用累积分布函数的方法实现离散概率分布
func (e *Engine) DiscreteDistribution(weight []float64) int32 {
	random := .0
	for _, w := range weight {
		random += w
	}
	random *= e.Float64()
	sum := 0.
	for i, w := range weight {
		sum += w
		if sum > random {
			return int32(i)
		}
	}
	log.Panicf("randengine: DiscreteDistribution: sum: %f random: %f", sum, random)
	return -1
}

// PTrue 以指定概率返回true（非线程安全）
// 功能：根据给定概率返回布尔值
// 参数：p-返回true的概率（0.0到1.0之间）
// 返回：true或false
// 算法说明：
// 1. 生成随机数：在[0.0, 1.0)范围内生成随机数
// 2. 概率比较：如果随机数小于给定概率则返回true
// 说明：实现伯努利分布，用于模拟概率事件
func (e *Engine) PTrue(p float64) bool {
	return e.Float64() < p
}

// PTrueSafe 以指定概率返回true（线程安全）
// 功能：根据给定概率返回布尔值，支持多线程安全访问
// 参数：p-返回true的概率（0.0到1.0之间）
// 返回：true或false
// 算法说明：
// 1. 获取互斥锁：确保线程安全
// 2. 生成随机数：在[0.0, 1.0)范围内生成随机数
// 3. 概率比较：如果随机数小于给定概率则返回true
// 4. 释放互斥锁：确保其他线程可以访问
// 说明：线程安全版本的PTrue方法
func (e *Engine) PTrueSafe(p float64) bool {
	e.mtx.Lock()
	defer e.mtx.Unlock()
	return e.Float64() < p
}

// IntnSafe 随机生成整数（线程安全）
// 功能：在指定范围内生成随机整数，支持多线程安全访问
// 参数：n-范围上限（不包含）
// 返回：[0, n)范围内的随机整数
// 算法说明：
// 1. 获取互斥锁：确保线程安全
// 2. 生成随机整数：调用底层rand.Intn方法
// 3. 释放互斥锁：确保其他线程可以访问
// 说明：线程安全版本的Intn方法
func (e *Engine) IntnSafe(n int) int {
	e.mtx.Lock()
	defer e.mtx.Unlock()
	return e.Intn(n)
}

// Float64Safe 随机生成浮点数（线程安全）
// 功能：生成[0.0, 1.0)范围内的随机浮点数，支持多线程安全访问
// 返回：[0.0, 1.0)范围内的随机浮点数
// 算法说明：
// 1. 获取互斥锁：确保线程安全
// 2. 生成随机浮点数：调用底层rand.Float64方法
// 3. 释放互斥锁：确保其他线程可以访问
// 说明：线程安全版本的Float64方法
func (e *Engine) Float64Safe() float64 {
	e.mtx.Lock()
	defer e.mtx.Unlock()
	return e.Float64()
}

// DiscreteDistributionSafe 按给定概率分布生成随机数（线程安全）
// 功能：根据权重数组生成离散分布的随机数，支持多线程安全访问
// 参数：weight-权重数组，每个元素表示对应索引的概率权重
// 返回：随机生成的索引值（0到len(weight)）
// 算法说明：
// 1. 计算总权重：遍历权重数组计算总和
// 2. 生成随机数：使用线程安全的Float64Safe方法
// 3. 累积概率：遍历权重数组，累积概率直到超过随机数
// 4. 返回索引：返回第一个累积概率超过随机数的索引
// 5. 边界处理：如果没有找到匹配的索引，返回数组长度
// 说明：线程安全版本的DiscreteDistribution方法
func (e *Engine) DiscreteDistributionSafe(weight []float64) int32 {
	random := .0
	for _, w := range weight {
		random += w
	}
	random *= e.Float64Safe()
	sum := 0.
	for i, w := range weight {
		sum += w
		if sum > random {
			return int32(i)
		}
	}
	return int32(len(weight))
}
