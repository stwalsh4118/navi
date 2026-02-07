package task

// DiscoverProjects takes a list of session working directories, deduplicates them
// by project root (where .navi.yaml is found), and returns merged configs.
// Projects without .navi.yaml are silently skipped.
func DiscoverProjects(cwds []string, global *GlobalConfig) []ProjectConfig {
	seen := make(map[string]bool)
	var configs []ProjectConfig

	for _, cwd := range cwds {
		cfg, err := FindProjectConfig(cwd)
		if err != nil || cfg == nil {
			continue
		}

		if seen[cfg.ProjectDir] {
			continue
		}
		seen[cfg.ProjectDir] = true

		MergeConfig(cfg, global)
		configs = append(configs, *cfg)
	}

	return configs
}
