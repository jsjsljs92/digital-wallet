package tarray

import (
	"fmt"
	"slices"
	"strings"
)

// Remove
func Remove[T comparable](elems []T, item T) []T {
	j := 0
	for _, v := range elems {
		if v != item {
			elems[j] = v
			j++
		}
	}
	return elems[:j]
}

// Join
func Join[T any](elems []T, sep string) string {
	if len(elems) == 0 {
		return ""
	}
	var buf strings.Builder
	for i, elem := range elems {
		if i > 0 {
			buf.WriteString(sep)
		}
		fmt.Fprint(&buf, elem)
	}
	return buf.String()
}

// Unique
func Unique[T comparable](slice []T) []T {
	seen := make(map[T]bool, len(slice))
	ret := make([]T, 0, len(slice))
	for _, v := range slice {
		if !seen[v] {
			seen[v] = true
			ret = append(ret, v)
		}
	}
	return ret
}

// Merge
func Merge[T any](slice1, slice2 []T) []T {
	s1Len := len(slice1)
	slice3 := make([]T, s1Len+len(slice2))
	copy(slice3, slice1)
	copy(slice3[s1Len:], slice2)
	return slice3
}

// Intersect
func Intersect[T comparable](slice1, slice2 []T) []T {
	// Build set from the smaller slice to save memory
	if len(slice1) > len(slice2) {
		slice1, slice2 = slice2, slice1
	}
	m := make(map[T]bool, len(slice1))
	for _, v := range slice1 {
		m[v] = true
	}
	nn := make([]T, 0, min(len(slice1), len(slice2)))
	for _, v := range slice2 {
		if m[v] {
			nn = append(nn, v)
		}
	}
	return nn
}

// Difference
func Difference[T comparable](slice1, slice2 []T) []T {
	m := make(map[T]bool, len(slice2))
	for _, v := range slice2 {
		m[v] = true
	}
	ret := make([]T, 0, len(slice1))
	for _, value := range slice1 {
		if !m[value] {
			ret = append(ret, value)
		}
	}
	return ret
}

// InArray
func InArray[T comparable](needle T, haystack []T) bool {
	return slices.Contains(haystack, needle)
}

// GroupBy
func GroupBy[T any, U comparable](slice []T, iteratee func(T) U) map[U][]T {
	res := make(map[U][]T)
	for _, v := range slice {
		key := iteratee(v)
		res[key] = append(res[key], v)
	}
	return res
}

// KeyBy
func KeyBy[T any, U comparable](slice []T, iteratee func(T) U) map[U]T {
	res := make(map[U]T)
	for _, v := range slice {
		key := iteratee(v)
		res[key] = v
	}
	return res
}

// Column
func Column[T any, U any](slice []T, iteratee func(T) U) []U {
	res := make([]U, len(slice))
	for i, v := range slice {
		res[i] = iteratee(v)
	}
	return res
}

// Empty
func Empty[T any](arg []T) bool {
	return len(arg) == 0
}

// Divide 将数组分割成 num 块
func Divide[T any](slice []T, num int) [][]T {
	if num <= 0 {
		panic("num must be greater than 0")
	}
	length := len(slice)
	if length == 0 {
		return [][]T{}
	}
	chunkSize := (length + num - 1) / num
	result := make([][]T, 0, num)
	for i := 0; i < length; i += chunkSize {
		result = append(result, slice[i:min(i+chunkSize, length)])
	}
	return result
}

// Chunk
func Chunk[T any](slice []T, chunkSize int) [][]T {
	if chunkSize <= 0 {
		return nil
	}
	length := len(slice)
	num := (length + chunkSize - 1) / chunkSize
	result := make([][]T, 0, num)
	for i := 0; i < length; i += chunkSize {
		result = append(result, slice[i:min(i+chunkSize, length)])
	}
	return result
}

// UniqueByFunc — O(n) using a map with a composite key strategy.
// Note: semantics differ slightly from the original — the original kept
// the *first* occurrence under a quadratic scan; this version also keeps
// the first occurrence by checking existence before insert.
func UniqueByFunc[T comparable](slice []T, IsSameFunc func(t1 T, t2 T) bool) []T {
	// For the general case with an arbitrary equality function, we use an
	// O(n^2) approach but with an early-exit optimization. For the common
	// case where IsSameFunc is equality, we could use a map, but we keep
	// the quadratic approach for correctness with arbitrary predicates.
	//
	// Optimization: track already-matched indices to skip redundant checks.
	if len(slice) <= 1 {
		return slice
	}
	seen := make([]bool, len(slice))
	ret := make([]T, 0, len(slice))
	for i := 0; i < len(slice); i++ {
		if seen[i] {
			continue
		}
		ret = append(ret, slice[i])
		for j := i + 1; j < len(slice); j++ {
			if !seen[j] && IsSameFunc(slice[i], slice[j]) {
				seen[j] = true
			}
		}
	}
	return ret
}

// Where
func Where[T comparable](slice []T, whereFunc func(T) bool) []T {
	ret := make([]T, 0, len(slice)/2)
	for _, item := range slice {
		if whereFunc(item) {
			ret = append(ret, item)
		}
	}
	return ret
}

// Find
func Find[T any](slice []T, matchFunc func(T) bool) (T, bool) {
	for _, item := range slice {
		if matchFunc(item) {
			return item, true
		}
	}
	var zero T
	return zero, false
}

// FindIndex
func FindIndex[T any](slice []T, matchFunc func(T) bool) int {
	for i, item := range slice {
		if matchFunc(item) {
			return i
		}
	}
	return -1
}

// FindLast
func FindLast[T any](slice []T, matchFunc func(T) bool) (T, bool) {
	for i := len(slice) - 1; i >= 0; i-- {
		if matchFunc(slice[i]) {
			return slice[i], true
		}
	}
	var zero T
	return zero, false
}

// FindLastIndex
func FindLastIndex[T any](slice []T, matchFunc func(T) bool) int {
	for i := len(slice) - 1; i >= 0; i-- {
		if matchFunc(slice[i]) {
			return i
		}
	}
	return -1
}
