package models

// UniqueInt64s 对 int64 切片进行去重
// 保持原始顺序，移除重复元素和零值
func UniqueInt64s(ids []int64) []int64 {
	if len(ids) == 0 {
		return ids
	}

	seen := make(map[int64]bool)
	result := make([]int64, 0, len(ids))

	for _, id := range ids {
		// 跳过零值和已经见过的值
		if id == 0 {
			continue
		}
		if !seen[id] {
			seen[id] = true
			result = append(result, id)
		}
	}

	return result
}

// UniqueUint64s 对 uint64 切片进行去重
// 保持原始顺序，移除重复元素和零值
func UniqueUint64s(ids []uint64) []uint64 {
	if len(ids) == 0 {
		return ids
	}

	seen := make(map[uint64]bool)
	result := make([]uint64, 0, len(ids))

	for _, id := range ids {
		// 跳过零值和已经见过的值
		if id == 0 {
			continue
		}
		if !seen[id] {
			seen[id] = true
			result = append(result, id)
		}
	}

	return result
}

// UniqueInts 对 int 切片进行去重
// 保持原始顺序，移除重复元素和零值
func UniqueInts(ids []int) []int {
	if len(ids) == 0 {
		return ids
	}

	seen := make(map[int]bool)
	result := make([]int, 0, len(ids))

	for _, id := range ids {
		// 跳过零值和已经见过的值
		if id == 0 {
			continue
		}
		if !seen[id] {
			seen[id] = true
			result = append(result, id)
		}
	}

	return result
}
