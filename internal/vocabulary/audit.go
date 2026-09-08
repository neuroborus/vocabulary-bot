package vocabulary

// FormCluster is a connected component of an item's forms, where two forms are
// linked when they share a strong lookup key. A healthy item produces exactly
// one cluster; more than one means unrelated phrases were merged into a single
// entity (the weak-key collision, e.g. many "X the Y" idioms joined via "the").
type FormCluster struct {
	Forms      []string `json:"forms"`
	StrongKeys []string `json:"strongKeys"`
}

// ClusterItemForms groups an item's forms into strong-key connected components,
// preserving first-appearance order. It is the reusable primitive for auditing
// stored items and for proposing how a collapsed item should be split back.
func ClusterItemForms(item Item) []FormCluster {
	n := len(item.Forms)
	if n == 0 {
		return nil
	}

	parent := make([]int, n)
	for i := range parent {
		parent[i] = i
	}
	var find func(int) int
	find = func(x int) int {
		for parent[x] != x {
			parent[x] = parent[parent[x]]
			x = parent[x]
		}
		return x
	}
	union := func(a, b int) {
		if ra, rb := find(a), find(b); ra != rb {
			parent[ra] = rb
		}
	}

	formStrong := make([][]string, n)
	keyOwner := make(map[string]int)
	for i, form := range item.Forms {
		strong := StrongLookupKeys(form.Value)
		formStrong[i] = strong
		for _, key := range strong {
			if owner, ok := keyOwner[key]; ok {
				union(i, owner)
			} else {
				keyOwner[key] = i
			}
		}
	}

	order := make([]int, 0)
	byRoot := make(map[int]*FormCluster)
	for i, form := range item.Forms {
		root := find(i)
		cluster, ok := byRoot[root]
		if !ok {
			cluster = &FormCluster{}
			byRoot[root] = cluster
			order = append(order, root)
		}
		cluster.Forms = append(cluster.Forms, form.Value)
		for _, key := range formStrong[i] {
			cluster.StrongKeys = appendLookupKey(cluster.StrongKeys, key)
		}
	}

	clusters := make([]FormCluster, 0, len(order))
	for _, root := range order {
		clusters = append(clusters, *byRoot[root])
	}

	return clusters
}

// IsCollapsed reports whether an item merges forms that do not share any strong
// lookup key, i.e. unrelated phrases that should live in separate items.
func IsCollapsed(item Item) bool {
	return len(ClusterItemForms(item)) > 1
}
