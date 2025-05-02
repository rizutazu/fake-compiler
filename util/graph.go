package util

// return componenets that node count is great that 1
func Kosaraju[T comparable](nodes []T, getInNeighbours, getOutNeighbours func(node T) []T) (result [][]T) {

	// https://en.wikipedia.org/wiki/Kosaraju%27s_algorithm

	visited := make(map[T]bool)
	assigned := make(map[T]bool)
	component := make(map[T][]T)
	for _, node := range nodes {
		visited[node] = false
		assigned[node] = false
	}

	var L []T

	var visit func(node T)
	visit = func(node T) {
		if !visited[node] {
			visited[node] = true
			neighbours := getOutNeighbours(node)
			for _, neighbour := range neighbours {
				visit(neighbour)
			}
			L = append([]T{node}, L...)
		}
	}

	for _, node := range nodes {
		visit(node)
	}

	var assign func(node T, root T)
	assign = func(node T, root T) {
		if !assigned[node] {
			assigned[node] = true
			_, ok := component[root]
			if !ok {
				component[root] = []T{}
			}
			component[root] = append(component[root], node)
			neighbours := getInNeighbours(node)
			for _, neighbour := range neighbours {
				assign(neighbour, root)
			}
		}
	}
	for _, node := range L {
		assign(node, node)
	}

	for _, c := range component {
		if len(c) > 1 {
			result = append(result, c)
		}
	}
	return
}

//func testK() {
//	// in-neighbour
//	in := map[int][]int{
//		0: {2},
//		1: {0},
//		2: {1},
//		3: {2},
//		4: {3, 6},
//		5: {4},
//		6: {5},
//		7: {4, 6},
//	}
//	// out
//	out := map[int][]int{
//		0: {1},
//		1: {2},
//		2: {0, 3},
//		3: {4},
//		4: {5, 7},
//		5: {6},
//		6: {4, 7},
//		7: {},
//	}
//	for root, r := range Kosaraju([]int{0, 1, 2, 3}, func(node int) []int {
//		return in[node]
//	}, func(node int) []int {
//		return out[node]
//	}) {
//		fmt.Println("root:", root, "components:", r)
//	}
//
//}
