package util

// get the set(unique elements) of an array
func Set[T comparable](slice []T) (result []T) {
	m := make(map[T]bool)
	for _, element := range slice {
		m[element] = true
	}
	for k := range m {
		result = append(result, k)
	}
	return
}

// return "s1 union s2" and "s1 - s2", s1 and s2 must be set
func GetUnionAndComplement[T comparable](s1, s2 []T) (union, complement []T) {
	m := make(map[T]bool)
	for _, element := range s2 {
		m[element] = true
	}
	for _, element := range s1 {
		_, ok := m[element]
		if ok {
			union = append(union, element)
		} else {
			complement = append(complement, element)
		}
	}
	return
}
