package workflow

import "sort"

func unionSorted(left []string, right []string) []string {
	set := map[string]struct{}{}
	for _, item := range left {
		if item != "" {
			set[item] = struct{}{}
		}
	}
	for _, item := range right {
		if item != "" {
			set[item] = struct{}{}
		}
	}
	out := make([]string, 0, len(set))
	for item := range set {
		out = append(out, item)
	}
	sort.Strings(out)
	return out
}
