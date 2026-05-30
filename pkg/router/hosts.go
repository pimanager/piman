package router

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

const (
	hostsHeader = "# Added by piMan - Do not edit manually"
	hostsFooter = "# End of piMan section"
)

// GetHostsPath returns the standard system path to the hosts file
func GetHostsPath() string {
	if runtime.GOOS == "windows" {
		return filepath.Join(os.Getenv("SystemRoot"), "System32", "drivers", "etc", "hosts")
	}
	return "/etc/hosts"
}

// GenerateHostsBlock returns the formatted hosts entries block for current routes
func GenerateHostsBlock(ip string) (string, error) {
	routes, err := GetRoutes()
	if err != nil {
		return "", err
	}

	var sb strings.Builder
	sb.WriteString(hostsHeader + "\n")
	seen := make(map[string]bool)
	for _, r := range routes {
		if seen[r.Domain] {
			continue
		}
		seen[r.Domain] = true
		sb.WriteString(fmt.Sprintf("%s %s\n", ip, r.Domain))
	}
	sb.WriteString(hostsFooter + "\n")
	return sb.String(), nil
}

// UpdateHostsContent parses the input hosts content, removes existing piMan block,
// and appends the new block. It returns the updated content.
func UpdateHostsContent(inputContent string, ip string) (string, error) {
	scanner := bufio.NewScanner(strings.NewReader(inputContent))
	var lines []string
	inPiManBlock := false

	for scanner.Scan() {
		line := scanner.Text()
		trimmed := strings.TrimSpace(line)
		if trimmed == hostsHeader {
			inPiManBlock = true
			continue
		}
		if trimmed == hostsFooter {
			inPiManBlock = false
			continue
		}
		if !inPiManBlock {
			lines = append(lines, line)
		}
	}

	if err := scanner.Err(); err != nil {
		return "", err
	}

	block, err := GenerateHostsBlock(ip)
	if err != nil {
		return "", err
	}

	// Remove trailing empty lines to keep it tidy before appending
	for len(lines) > 0 && strings.TrimSpace(lines[len(lines)-1]) == "" {
		lines = lines[:len(lines)-1]
	}

	var output strings.Builder
	for _, line := range lines {
		output.WriteString(line + "\n")
	}
	output.WriteString("\n" + block)

	return output.String(), nil
}

// UpdateHostsFile updates the system /etc/hosts file with the current routes
func UpdateHostsFile(ip string) error {
	hostsPath := GetHostsPath()

	data, err := os.ReadFile(hostsPath)
	if err != nil {
		return fmt.Errorf("failed to read hosts file: %w (make sure to run with administrative/sudo privileges)", err)
	}

	newContent, err := UpdateHostsContent(string(data), ip)
	if err != nil {
		return err
	}

	err = os.WriteFile(hostsPath, []byte(newContent), 0644)
	if err != nil {
		return fmt.Errorf("failed to write hosts file: %w (make sure to run with administrative/sudo privileges)", err)
	}

	return nil
}
