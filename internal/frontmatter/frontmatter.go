package frontmatter

import (
	"fmt"
	"os"
	"strings"
)

// Metadata represents frontmatter metadata
type Metadata struct {
	Domain   string   `yaml:"domain"`
	DocType  string   `yaml:"docType"`
	Language string   `yaml:"language"`
	Tags     []string `yaml:"tags"`
	Project  string   `yaml:"project,omitempty"`
}

// Parse extracts frontmatter from markdown content
func Parse(content string) (*Metadata, string, error) {
	lines := strings.Split(content, "\n")

	if len(lines) < 3 {
		return nil, content, nil // No frontmatter
	}

	// Check for frontmatter delimiters
	if strings.TrimSpace(lines[0]) != "---" {
		return nil, content, nil // No frontmatter
	}

	// Find closing delimiter
	endIdx := -1
	for i := 1; i < len(lines); i++ {
		if strings.TrimSpace(lines[i]) == "---" {
			endIdx = i
			break
		}
	}

	if endIdx == -1 {
		return nil, content, fmt.Errorf("frontmatter not closed")
	}

	// Extract frontmatter and body
	frontmatterLines := lines[1:endIdx]
	bodyLines := lines[endIdx+1:]

	metadata := &Metadata{}

	// Simple YAML parsing
	for _, line := range frontmatterLines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		parts := strings.SplitN(line, ":", 2)
		if len(parts) != 2 {
			continue
		}

		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])

		switch key {
		case "domain":
			metadata.Domain = value
		case "docType":
			metadata.DocType = value
		case "language":
			metadata.Language = value
		case "project":
			metadata.Project = value
		case "tags":
			// Parse array: [tag1, tag2, tag3]
			value = strings.Trim(value, "[]")
			tags := strings.Split(value, ",")
			for _, tag := range tags {
				tag = strings.TrimSpace(tag)
				if tag != "" {
					metadata.Tags = append(metadata.Tags, tag)
				}
			}
		}
	}

	body := strings.Join(bodyLines, "\n")
	return metadata, body, nil
}

// Generate creates frontmatter string from metadata
func Generate(metadata *Metadata) string {
	var builder strings.Builder

	builder.WriteString("---\n")

	if metadata.Domain != "" {
		builder.WriteString(fmt.Sprintf("domain: %s\n", metadata.Domain))
	}
	if metadata.DocType != "" {
		builder.WriteString(fmt.Sprintf("docType: %s\n", metadata.DocType))
	}
	if metadata.Language != "" {
		builder.WriteString(fmt.Sprintf("language: %s\n", metadata.Language))
	}
	if len(metadata.Tags) > 0 {
		builder.WriteString("tags: [")
		builder.WriteString(strings.Join(metadata.Tags, ", "))
		builder.WriteString("]\n")
	}
	if metadata.Project != "" {
		builder.WriteString(fmt.Sprintf("project: %s\n", metadata.Project))
	}

	builder.WriteString("---\n")

	return builder.String()
}

// AddFrontmatter adds frontmatter to a file
func AddFrontmatter(filePath string, metadata *Metadata) error {
	// Read file
	content, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to read file: %w", err)
	}

	// Check if frontmatter already exists
	existing, _, err := Parse(string(content))
	if err != nil {
		return fmt.Errorf("failed to parse frontmatter: %w", err)
	}

	if existing != nil {
		return fmt.Errorf("frontmatter already exists")
	}

	// Generate new frontmatter
	frontmatter := Generate(metadata)

	// Combine frontmatter with body
	newContent := frontmatter + "\n" + string(content)

	// Write back
	if err := os.WriteFile(filePath, []byte(newContent), 0644); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	return nil
}

// UpdateFrontmatter updates existing frontmatter
func UpdateFrontmatter(filePath string, metadata *Metadata) error {
	// Read file
	content, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to read file: %w", err)
	}

	// Parse existing frontmatter
	existing, body, err := Parse(string(content))
	if err != nil {
		return fmt.Errorf("failed to parse frontmatter: %w", err)
	}

	if existing == nil {
		// No existing frontmatter, add it
		return AddFrontmatter(filePath, metadata)
	}

	// Merge metadata (new values override existing)
	if metadata.Domain != "" {
		existing.Domain = metadata.Domain
	}
	if metadata.DocType != "" {
		existing.DocType = metadata.DocType
	}
	if metadata.Language != "" {
		existing.Language = metadata.Language
	}
	if len(metadata.Tags) > 0 {
		existing.Tags = metadata.Tags
	}
	if metadata.Project != "" {
		existing.Project = metadata.Project
	}

	// Generate new frontmatter
	frontmatter := Generate(existing)

	// Combine frontmatter with body
	newContent := frontmatter + "\n" + strings.TrimLeft(body, "\n")

	// Write back
	if err := os.WriteFile(filePath, []byte(newContent), 0644); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	return nil
}

// ReadFile reads a file and returns its metadata and content
func ReadFile(filePath string) (*Metadata, string, error) {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return nil, "", fmt.Errorf("failed to read file: %w", err)
	}

	metadata, bodyContent, err := Parse(string(content))
	if err != nil {
		return nil, "", err
	}

	return metadata, bodyContent, nil
}
