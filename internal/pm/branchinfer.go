package pm

import (
	"regexp"
	"strings"
	"sync"
)

var DefaultBranchPatterns = []string{
	`^feature/pbi-(\d+)(?:$|[-/].*)`,
	`^pbi-(\d+)(?:$|[-/].*)`,
	`^(\d+)-`,
}

var (
	defaultBranchRegexOnce sync.Once
	defaultBranchRegexes   []*regexp.Regexp
)

// InferPBIFromBranch extracts a numeric PBI ID from a branch name.
// It uses DefaultBranchPatterns when patterns is nil or empty.
func InferPBIFromBranch(branch string, patterns []string) (pbiID string, ok bool) {
	trimmedBranch := strings.TrimSpace(branch)
	if trimmedBranch == "" {
		return "", false
	}

	regexes := regexesForPatterns(patterns)
	for _, re := range regexes {
		match := re.FindStringSubmatch(trimmedBranch)
		if len(match) < 2 {
			continue
		}

		captured := strings.TrimSpace(match[1])
		if captured != "" {
			return captured, true
		}
	}

	return "", false
}

func regexesForPatterns(patterns []string) []*regexp.Regexp {
	if len(patterns) == 0 {
		defaultBranchRegexOnce.Do(func() {
			defaultBranchRegexes = compileBranchPatterns(DefaultBranchPatterns)
		})
		return defaultBranchRegexes
	}

	return compileBranchPatterns(patterns)
}

func compileBranchPatterns(patterns []string) []*regexp.Regexp {
	compiled := make([]*regexp.Regexp, 0, len(patterns))
	for _, pattern := range patterns {
		re, err := regexp.Compile(pattern)
		if err != nil {
			continue
		}
		if re.NumSubexp() != 1 {
			continue
		}
		compiled = append(compiled, re)
	}

	return compiled
}
