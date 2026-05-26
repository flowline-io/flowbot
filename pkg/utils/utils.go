// Generic data manipulation utilities.

package utils

import (
	"path/filepath"
	"slices"
)

// stringDelta extracts the slices of added and removed strings from two slices:
//
//	added :=  newSlice - (oldSlice & newSlice) -- present in new but missing in old
//	removed := oldSlice - (oldSlice & newSlice) -- present in old but missing in new
//	intersection := oldSlice & newSlice -- present in both old and new
func stringSliceDelta(rold, rnew []string) (added, removed, intersection []string) {
	if len(rold) == 0 && len(rnew) == 0 {
		return nil, nil, nil
	}
	if len(rold) == 0 {
		return rnew, nil, nil
	}
	if len(rnew) == 0 {
		return nil, rold, nil
	}

	slices.Sort(rold)
	slices.Sort(rnew)

	// Match old slice against the new slice and separate removed strings from added.
	o, n := 0, 0
	lold, lnew := len(rold), len(rnew)
	for o < lold || n < lnew {
		if o == lold || (n < lnew && rold[o] > rnew[n]) {
			// Present in new, missing in old: added
			added = append(added, rnew[n])
			n++
		} else if n == lnew || rold[o] < rnew[n] {
			// Present in old, missing in new: removed
			removed = append(removed, rold[o])
			o++
		} else {
			// present in both
			intersection = append(intersection, rold[o])
			if o < lold {
				o++
			}
			if n < lnew {
				n++
			}
		}
	}
	return added, removed, intersection
}

// ToAbsolutePath Convert relative filepath to absolute.
func ToAbsolutePath(base, path string) string {
	if filepath.IsAbs(path) {
		return path
	}
	return filepath.Clean(filepath.Join(base, path))
}
