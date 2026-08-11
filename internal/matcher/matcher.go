package matcher

import (
	"strings"

	"github.com/hasanm95/go-nx/internal/config"
)


func MatchPath(paths []config.PathConfig, requestedPath string) (*config.PathConfig, bool){
	
	for _, path := range paths {
		if path.Path == requestedPath {
			return &path, true
		}

		token, found := strings.CutSuffix(path.Path, "*")
		if found{
			if strings.HasPrefix(requestedPath, token){
				return  &path, true
			}
		}
	}

	return nil, false
}