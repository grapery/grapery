package models

import (
	"reflect"
	"testing"
)

func TestUniqueInt64s(t *testing.T) {
	tests := []struct {
		name     string
		input    []int64
		expected []int64
	}{
		{
			name:     "重复ID",
			input:    []int64{1, 2, 1, 3, 2},
			expected: []int64{1, 2, 3},
		},
		{
			name:     "包含零值",
			input:    []int64{1, 0, 2, 0, 3},
			expected: []int64{1, 2, 3},
		},
		{
			name:     "空数组",
			input:    []int64{},
			expected: []int64{},
		},
		{
			name:     "全部零值",
			input:    []int64{0, 0, 0},
			expected: []int64{},
		},
		{
			name:     "无重复",
			input:    []int64{1, 2, 3, 4, 5},
			expected: []int64{1, 2, 3, 4, 5},
		},
		{
			name:     "实际场景-日志中的重复ID",
			input:    []int64{1, 1, 1, 1, 1, 1, 4, 1, 4, 4},
			expected: []int64{1, 4},
		},
		{
			name:     "实际场景-全部零值",
			input:    []int64{0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
			expected: []int64{},
		},
		{
			name:     "单个元素",
			input:    []int64{1},
			expected: []int64{1},
		},
		{
			name:     "负数ID",
			input:    []int64{-1, -2, -1, 0, 1, 1},
			expected: []int64{-1, -2, 1},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := UniqueInt64s(tt.input)
			if !reflect.DeepEqual(result, tt.expected) {
				t.Errorf("UniqueInt64s() = %v, 期望 %v", result, tt.expected)
			}
		})
	}
}

func TestUniqueUint64s(t *testing.T) {
	tests := []struct {
		name     string
		input    []uint64
		expected []uint64
	}{
		{
			name:     "重复ID",
			input:    []uint64{1, 2, 1, 3, 2},
			expected: []uint64{1, 2, 3},
		},
		{
			name:     "包含零值",
			input:    []uint64{1, 0, 2, 0, 3},
			expected: []uint64{1, 2, 3},
		},
		{
			name:     "空数组",
			input:    []uint64{},
			expected: []uint64{},
		},
		{
			name:     "全部零值",
			input:    []uint64{0, 0, 0},
			expected: []uint64{},
		},
		{
			name:     "无重复",
			input:    []uint64{1, 2, 3, 4, 5},
			expected: []uint64{1, 2, 3, 4, 5},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := UniqueUint64s(tt.input)
			if !reflect.DeepEqual(result, tt.expected) {
				t.Errorf("UniqueUint64s() = %v, 期望 %v", result, tt.expected)
			}
		})
	}
}

func TestUniqueInts(t *testing.T) {
	tests := []struct {
		name     string
		input    []int
		expected []int
	}{
		{
			name:     "重复ID",
			input:    []int{1, 2, 1, 3, 2},
			expected: []int{1, 2, 3},
		},
		{
			name:     "包含零值",
			input:    []int{1, 0, 2, 0, 3},
			expected: []int{1, 2, 3},
		},
		{
			name:     "空数组",
			input:    []int{},
			expected: []int{},
		},
		{
			name:     "全部零值",
			input:    []int{0, 0, 0},
			expected: []int{},
		},
		{
			name:     "无重复",
			input:    []int{1, 2, 3, 4, 5},
			expected: []int{1, 2, 3, 4, 5},
		},
		{
			name:     "负数ID",
			input:    []int{-1, -2, -1, 0, 1, 1},
			expected: []int{-1, -2, 1},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := UniqueInts(tt.input)
			if !reflect.DeepEqual(result, tt.expected) {
				t.Errorf("UniqueInts() = %v, 期望 %v", result, tt.expected)
			}
		})
	}
}

// 性能基准测试
func BenchmarkUniqueInt64s_Small(b *testing.B) {
	ids := []int64{1, 2, 3, 1, 2, 3}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = UniqueInt64s(ids)
	}
}

func BenchmarkUniqueInt64s_Medium(b *testing.B) {
	ids := make([]int64, 100)
	for i := 0; i < 100; i++ {
		ids[i] = int64(i % 10) // 10个不同的值，每个重复10次
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = UniqueInt64s(ids)
	}
}

func BenchmarkUniqueInt64s_Large(b *testing.B) {
	ids := make([]int64, 1000)
	for i := 0; i < 1000; i++ {
		ids[i] = int64(i % 100) // 100个不同的值，每个重复10次
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = UniqueInt64s(ids)
	}
}

func BenchmarkUniqueInt64s_RealWorld(b *testing.B) {
	// 模拟实际场景：大量重复ID
	ids := []int64{1, 1, 1, 1, 1, 1, 4, 1, 4, 4}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = UniqueInt64s(ids)
	}
}
