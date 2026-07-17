package coding

import (
	"context"
	"fmt"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/flowline-io/flowbot/pkg/agent/env"
)

func walkGrep(ctx context.Context, execEnv env.ExecutionEnv, workspaceRoot, searchRoot string, re *regexp.Regexp, globFilter string, maxMatches int) ([]string, string, error) {
	state := &grepWalkState{re: re, globFilter: globFilter, maxMatches: maxMatches, workspaceRoot: workspaceRoot, searchRoot: searchRoot}
	if err := state.walk(ctx, execEnv, searchRoot); err != nil {
		return nil, "", err
	}
	return state.hits, state.truncReason, nil
}

type grepWalkState struct {
	re            *regexp.Regexp
	globFilter    string
	maxMatches    int
	workspaceRoot string
	searchRoot    string
	hits          []string
	truncReason   string
	filesScanned  int
}

func (s *grepWalkState) walk(ctx context.Context, execEnv env.ExecutionEnv, dir string) error {
	if s.stopWalking() {
		return nil
	}
	entriesResult := execEnv.ReadDir(ctx, dir)
	if !entriesResult.IsOk() {
		return fmt.Errorf("%s", env.FormatFileError(entriesResult.ErrorValue()))
	}
	for _, entry := range entriesResult.Value() {
		if s.stopWalking() {
			return nil
		}
		abs := filepath.Join(dir, entry.Name)
		if entry.IsDir {
			if ShouldSkipDir(entry.Name) {
				continue
			}
			if err := s.walk(ctx, execEnv, abs); err != nil {
				return err
			}
			continue
		}
		if err := s.scanFile(ctx, execEnv, abs); err != nil {
			return err
		}
	}
	return nil
}

func (s *grepWalkState) stopWalking() bool {
	if len(s.hits) >= s.maxMatches {
		s.truncReason = fmt.Sprintf("(truncated to %d matches)", s.maxMatches)
		return true
	}
	if s.filesScanned >= MaxGrepFilesScanned {
		s.truncReason = fmt.Sprintf("(truncated after scanning %d files)", MaxGrepFilesScanned)
		return true
	}
	return false
}

func (s *grepWalkState) scanFile(ctx context.Context, execEnv env.ExecutionEnv, abs string) error {
	rel, err := filepath.Rel(s.workspaceRoot, abs)
	if err != nil {
		return err
	}
	rel = filepath.ToSlash(rel)
	ok, err := s.matchesGlob(abs, rel)
	if err != nil || !ok {
		return err
	}
	s.filesScanned++
	readResult := execEnv.ReadFile(ctx, abs)
	if !readResult.IsOk() {
		return nil
	}
	data := readResult.Value()
	if len(data) > MaxGrepFileBytes || strings.Contains(string(data), "\x00") {
		return nil
	}
	s.collectHits(rel, string(data))
	return nil
}

func (s *grepWalkState) matchesGlob(abs, rel string) (bool, error) {
	if s.globFilter == "" {
		return true, nil
	}
	ok, err := MatchPath(s.globFilter, rel)
	if err != nil || ok {
		return ok, err
	}
	relSearch, err := filepath.Rel(s.searchRoot, abs)
	if err != nil {
		return false, err
	}
	return MatchPath(s.globFilter, filepath.ToSlash(relSearch))
}

func (s *grepWalkState) collectHits(rel, content string) {
	lines := strings.Split(content, "\n")
	for i, line := range lines {
		if len(s.hits) >= s.maxMatches {
			s.truncReason = fmt.Sprintf("(truncated to %d matches)", s.maxMatches)
			return
		}
		if s.re.MatchString(line) {
			s.hits = append(s.hits, fmt.Sprintf("%s:%d:%s", rel, i+1, line))
		}
	}
}
