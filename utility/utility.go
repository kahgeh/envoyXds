package utility

import "strconv"

// ToMap maps string array to map[string] int
func ToMap(texts []string) map[string]int {
	m := make(map[string]int)

	for index, text := range texts {
		m[text] = index
	}
	return m
}

// MustAtoi returns an integer, throws an exception if there is error
func MustAtoi(s string) int {
	n, err := strconv.Atoi(s)
	if err != nil {
		panic(err)
	}
	return n
}
