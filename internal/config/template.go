package config

import (
	"fmt"
	"strings"
)

// GenerateCommentedConfig produces a config.toml content for 'jokateko init'
// using the embedded default.toml as the single source of truth.
// Only version = "0" and [project] (name and description) are active;
// all subsequent configuration sections are commented out so the project
// inherits compiled-in defaults while remaining fully documented.
func GenerateCommentedConfig(projectName, projectDescription string) string {
	if projectName == "" {
		projectName = "My Project"
	}
	if projectDescription == "" {
		projectDescription = "Markdown-driven task management and Kanban"
	}

	var sb strings.Builder
	inProjectSection := false
	pastProjectSection := false

	for line := range strings.Lines(string(defaultTOML)) {
		line = strings.TrimSuffix(line, "\r\n")
		line = strings.TrimSuffix(line, "\n")
		trimmed := strings.TrimSpace(line)

		// Check if we are entering [project]
		if trimmed == "[project]" {
			inProjectSection = true
			sb.WriteString(line)
			sb.WriteByte('\n')
			continue
		}

		if inProjectSection {
			if strings.HasPrefix(trimmed, "name =") {
				sb.WriteString(fmt.Sprintf("name = %q\n", projectName))
				continue
			}
			if strings.HasPrefix(trimmed, "description =") {
				sb.WriteString(fmt.Sprintf("description = %q\n", projectDescription))
				continue
			}
			// When a new section header begins (e.g. [paths]), we leave the project section
			if strings.HasPrefix(trimmed, "[") {
				inProjectSection = false
				pastProjectSection = true
			}
		}

		if !pastProjectSection {
			// Lines before or in [project] (like version = "0" and banner comments) remain active
			sb.WriteString(line)
			sb.WriteByte('\n')
			continue
		}

		// In pastProjectSection, empty lines and existing comments remain as-is
		if trimmed == "" {
			sb.WriteByte('\n')
			continue
		}
		if strings.HasPrefix(trimmed, "#") {
			sb.WriteString(line)
			sb.WriteByte('\n')
			continue
		}

		// Comment out the active configuration setting
		sb.WriteString("# ")
		sb.WriteString(line)
		sb.WriteByte('\n')
	}

	return sb.String()
}
