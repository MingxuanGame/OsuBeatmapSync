package utils

import (
	"math"
)

func In[T comparable](array []T, element T) bool {
	for _, v := range array {
		if v == element {
			return true
		}
	}
	return false
}

func SplitSlice[T any](s []T, n int) [][]T {
	totalLength := len(s)
	partSize := int(math.Ceil(float64(totalLength) / float64(n)))

	var result [][]T
	for i := 0; i < totalLength; i += partSize {
		end := i + partSize
		if end > totalLength {
			end = totalLength
		}
		result = append(result, s[i:end])
	}

	return result
}
