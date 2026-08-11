package matcher

import (
	"fmt"
	"sort"
	"strings"

	"github.com/hasanm95/go-nx/internal/config"
)


func MatchPath(paths []config.PathConfig, requestedPath string) (*config.PathConfig, bool){
	
	routes := newRouteConfig(paths)
	fmt.Println(routes)

	for _, route := range routes {
		// Handle wildcard matching
		if strings.HasSuffix(route.Path, "*") {
			prefix, found := strings.CutSuffix(route.Path, "*")
			if found && strings.HasPrefix(requestedPath, prefix) {
				return &route, true
			}
			continue
		}

		// Handle exact matching
		if route.Path == requestedPath {
			return &route, true
		}
	}

	return nil, false
}

func newRouteConfig(paths []config.PathConfig) []config.PathConfig {
	// clone original paths
	clonedPaths := append([]config.PathConfig(nil), paths...)

	// Sort rules: exact matches first, then longer prefixes before shorter ones
	sort.SliceStable(clonedPaths, func(i, j int) bool {
		p1, p2 := paths[i].Path, paths[j].Path
		isWild1 := strings.HasSuffix(p1, "*")
		isWild2 := strings.HasSuffix(p2, "*")

		// Rule 1: Exact matches beat wildcard matches
		if isWild1 && !isWild2 {
			return true
		}
		if !isWild1 && isWild2 {
			return false
		}

		// Rule 2: If both are wildcards, the longer prefix wins (more specific)
		if isWild1 && isWild2 {
			return len(p1) > len(p2)
		}

		// If both are exact matches, tie-break by longer string
		return len(p1) > len(p2)
	})

	return clonedPaths
}