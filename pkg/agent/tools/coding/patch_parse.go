package coding

import (
	"fmt"
	"strings"
)

func parseCodexPatch(text string) ([]patchOp, error) {
	lines := splitPatchLines(text)
	start, end, err := findPatchBounds(lines)
	if err != nil {
		return nil, err
	}

	var ops []patchOp
	i := start + 1
	for i < end {
		line := strings.TrimRight(lines[i], "\r")
		switch {
		case strings.HasPrefix(line, "*** Add File: "):
			op, next, parseErr := parseAddFile(lines, i, end)
			if parseErr != nil {
				return nil, parseErr
			}
			ops = append(ops, op)
			i = next
		case strings.HasPrefix(line, "*** Delete File: "):
			path := strings.TrimSpace(strings.TrimPrefix(line, "*** Delete File: "))
			ops = append(ops, patchOp{Kind: patchDelete, Path: path})
			i++
		case strings.HasPrefix(line, "*** Update File: "):
			op, next, parseErr := parseUpdateFile(lines, i, end)
			if parseErr != nil {
				return nil, parseErr
			}
			ops = append(ops, op)
			i = next
		default:
			if strings.TrimSpace(line) == "" {
				i++
				continue
			}
			return nil, fmt.Errorf("unexpected patch line %q", line)
		}
	}
	return ops, nil
}

func findPatchBounds(lines []string) (start, end int, err error) {
	start, end = -1, -1
	for i, line := range lines {
		if strings.TrimSpace(line) == "*** Begin Patch" {
			start = i
		}
		if strings.TrimSpace(line) == "*** End Patch" {
			end = i
			break
		}
	}
	if start < 0 || end < 0 || end <= start {
		return 0, 0, fmt.Errorf("patch must contain *** Begin Patch and *** End Patch")
	}
	return start, end, nil
}

func parseAddFile(lines []string, i, end int) (patchOp, int, error) {
	path := strings.TrimSpace(strings.TrimPrefix(strings.TrimRight(lines[i], "\r"), "*** Add File: "))
	i++
	var contentLines []string
	for i < end && !strings.HasPrefix(lines[i], "*** ") {
		raw := lines[i]
		switch {
		case strings.HasPrefix(raw, "+"):
			contentLines = append(contentLines, strings.TrimPrefix(raw, "+"))
		case raw == "":
			contentLines = append(contentLines, "")
		default:
			return patchOp{}, 0, fmt.Errorf("add file %s: lines must start with +", path)
		}
		i++
	}
	return patchOp{Kind: patchAdd, Path: path, Content: strings.Join(contentLines, "\n")}, i, nil
}

func parseUpdateFile(lines []string, i, end int) (patchOp, int, error) {
	path := strings.TrimSpace(strings.TrimPrefix(strings.TrimRight(lines[i], "\r"), "*** Update File: "))
	i++
	var hunks [][]string
	var current []string
	for i < end && !strings.HasPrefix(lines[i], "*** ") {
		raw := lines[i]
		if strings.HasPrefix(raw, "@@") {
			if len(current) > 0 {
				hunks = append(hunks, current)
				current = nil
			}
			i++
			continue
		}
		if len(raw) > 0 && (raw[0] == ' ' || raw[0] == '-' || raw[0] == '+') {
			current = append(current, raw)
			i++
			continue
		}
		if strings.TrimSpace(raw) == "" {
			i++
			continue
		}
		return patchOp{}, 0, fmt.Errorf("update file %s: unexpected line %q", path, raw)
	}
	if len(current) > 0 {
		hunks = append(hunks, current)
	}
	return patchOp{Kind: patchUpdate, Path: path, Hunks: hunks}, i, nil
}
