package coding

import (
	"time"
)

const (
	// DefaultMaxOutput truncates tool result text beyond this byte count.
	DefaultMaxOutput = 8192
	// DefaultShellTimeout limits run_terminal and run_code execution.
	DefaultShellTimeout = 60 * time.Second
	// DefaultHTTPTimeout limits web_search and web_fetch HTTP calls.
	DefaultHTTPTimeout = 15 * time.Second

	// MaxReadFileBytes rejects read_file when the whole file exceeds this size.
	MaxReadFileBytes = 1 << 20
	// MaxWriteFileBytes rejects write_file content above this size.
	MaxWriteFileBytes = 1 << 20
	// MaxRunCodeBytes rejects run_code source above this size.
	MaxRunCodeBytes = 256 << 10
	// MaxWebSearchQueryBytes rejects web_search queries above this length.
	MaxWebSearchQueryBytes = 512
	// MaxWebSearchResults caps organic results returned by web_search.
	MaxWebSearchResults = 8

	// MaxListDirEntries caps list_dir result entries.
	MaxListDirEntries = 500
	// DefaultGlobMaxMatches is the default glob_files match cap.
	DefaultGlobMaxMatches = 200
	// HardGlobMaxMatches is the maximum allowed glob_files max_matches argument.
	HardGlobMaxMatches = 1000
	// DefaultGrepMaxMatches is the default grep_files hit cap.
	DefaultGrepMaxMatches = 100
	// HardGrepMaxMatches is the maximum allowed grep_files max_matches argument.
	HardGrepMaxMatches = 500
	// MaxGrepFileBytes skips grep_files files larger than this size.
	MaxGrepFileBytes = 1 << 20
	// MaxGrepFilesScanned caps how many files grep_files opens.
	MaxGrepFilesScanned = 2000
	// MaxPatchBytes rejects apply_patch payloads above this size.
	MaxPatchBytes = 1 << 20
	// MaxPatchFiles caps files touched by one apply_patch call.
	MaxPatchFiles = 20
	// MaxFetchBytes caps web_fetch response body bytes read.
	MaxFetchBytes = 1 << 20
)

// SkipDirNames are directory basenames skipped by recursive listing, glob, and grep.
var SkipDirNames = map[string]struct{}{
	".git":         {},
	"node_modules": {},
}

// ClampMaxMatches returns a positive match limit capped by hardMax, using defaultMax when arg is unset.
func ClampMaxMatches(arg, defaultMax, hardMax int) int {
	if arg <= 0 {
		arg = defaultMax
	}
	if hardMax > 0 && arg > hardMax {
		return hardMax
	}
	return arg
}

// ShouldSkipDir reports whether a directory basename should be skipped during walks.
func ShouldSkipDir(name string) bool {
	_, ok := SkipDirNames[name]
	return ok
}
