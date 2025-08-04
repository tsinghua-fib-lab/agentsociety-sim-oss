package container_test

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/tsinghua-fib-lab/agentsociety-sim-oss/utils/container"
)

type testData struct {
}

func (t testData) V() float64 {
	return 0
}

func (t testData) Length() float64 {
	return 0
}

func TestListInit(t *testing.T) {
	l := &container.List[testData, struct{}]{}
	assert.Nil(t, l.First())
	assert.Nil(t, l.Last())
	assert.Equal(t, 0, l.Len())
}

func TestListOperation(t *testing.T) {
	l := &container.List[testData, struct{}]{}

	// test: insert

	// ^, 1, ^
	n1 := &container.ListNode[testData, struct{}]{
		S:     1,
		Value: testData{},
	}
	l.PushBack(n1)
	// ^, 2, 1, ^
	n2 := &container.ListNode[testData, struct{}]{
		S:     2,
		Value: testData{},
	}
	l.PushFront(n2)
	// ^, 3, 2, 1, ^
	n3 := &container.ListNode[testData, struct{}]{
		S:     3,
		Value: testData{},
	}
	n2.InsertBefore(n3)
	// ^, 3, 2, 1, 4, ^
	n4 := &container.ListNode[testData, struct{}]{
		S:     4,
		Value: testData{},
	}
	n1.InsertAfter(n4)
	assert.Equal(t, 4, l.Len())

	// test: first last next prev

	n := l.First()
	assert.Equal(t, n3, n)
	n = n.Next()
	assert.Equal(t, n2, n)
	n = n.Next()
	assert.Equal(t, n1, n)
	assert.Equal(t, n, n.Next().Prev())
	assert.Equal(t, n, n.Prev().Next())
	n = n.Next()
	assert.Equal(t, n4, n)

	assert.Equal(t, n4, l.Last())

	// test: pop merge

	// before: head, 3, 2, 1, 4, tail
	n0 := &container.ListNode[testData, struct{}]{
		S:     0,
		Value: testData{},
	}
	l.PushFront(n0)
	unsorted := l.PopUnsorted()
	assert.ElementsMatch(t, []*container.ListNode[testData, struct{}]{n2, n1}, unsorted)
	assert.Equal(t, 5-2, l.Len())

	// head, 0, 1, 2, 3, 4, tail
	l.Merge(unsorted)
	fmt.Print(l.Keys())
	node := l.First()
	assert.Equal(t, n0, node)
	node = node.Next()
	assert.Equal(t, n1, node)
	node = node.Next()
	assert.Equal(t, n2, node)
	node = node.Next()
	assert.Equal(t, n3, node)
	node = node.Next()
	assert.Equal(t, n4, node)
	node = node.Next()
	assert.Nil(t, node)

	// test: remove

	// head, 0, 1, 2, 3, tail
	l.Remove(n4)
	assert.Equal(t, n3, l.Last())
	assert.Equal(t, 5-1, l.Len())
}
