package coding

import (
	"path"
	"strings"
)

// MatchPath reports whether relativePath matches pattern.
// Patterns use forward slashes; ** matches any number of path segments.
func MatchPath(pattern, relativePath string) (bool, error) {
	pattern = strings.ReplaceAll(pattern, "\\", "/")
	relativePath = strings.ReplaceAll(relativePath, "\\", "/")
	pattern = strings.TrimPrefix(pattern, "./")
	relativePath = strings.TrimPrefix(relativePath, "./")
	return matchPathSegments(strings.Split(pattern, "/"), strings.Split(relativePath, "/"))
}

func matchPathSegments(patternParts, pathParts []string) (bool, error) {
	for len(patternParts) > 0 {
		part := patternParts[0]
		if part == "**" {
			if len(patternParts) == 1 {
				return true, nil
			}
			for i := 0; i <= len(pathParts); i++ {
				ok, err := matchPathSegments(patternParts[1:], pathParts[i:])
				if err != nil {
					return false, err
				}
				if ok {
					return true, nil
				}
			}
			return false, nil
		}
		if len(pathParts) == 0 {
			return false, nil
		}
		ok, err := path.Match(part, pathParts[0])
		if err != nil {
			return false, err
		}
		if !ok {
			return false, nil
		}
		patternParts = patternParts[1:]
		pathParts = pathParts[1:]
	}
	return len(pathParts) == 0, nil
}
