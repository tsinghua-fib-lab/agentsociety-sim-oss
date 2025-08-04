package utils

// 找出ID(int32)对应的数据。
// 如果ids为空则返回所有数据，
// 如果不存在则将失败ID记录到失败列表中。
func Find[T any](dataMap map[int32]T, data []T, ids []int32) (okData []T, failedIDs []int32) {
	if len(ids) == 0 {
		return data, nil
	}
	okData = make([]T, 0, len(ids))
	failedIDs = make([]int32, 0, len(ids))
	for _, id := range ids {
		if d, ok := dataMap[id]; ok {
			okData = append(okData, d)
		} else {
			failedIDs = append(failedIDs, id)
		}
	}
	return
}
