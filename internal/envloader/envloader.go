package envloader

import (
	"bufio"
	"fmt"
	"os"
	"regexp"
	"path/filepath"
	"strings"
)

// LoadEnv loads environment variables from a .env file
// It looks for .env in the current directory or project root
func LoadEnv() error {
	// Try to find .env file
	envPath := findEnvFile()
	if envPath == "" {
		// No .env file found, that's okay - use environment variables directly
		return nil
	}

	file, err := os.Open(envPath)
	if err != nil {
		return fmt.Errorf("failed to open .env file: %w", err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	lineNum := 0

	for scanner.Scan() {
		lineNum++
		line := strings.TrimSpace(scanner.Text())

		// Skip empty lines and comments
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// Parse KEY=VALUE
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue // Skip malformed lines
		}

		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])

		// Remove quotes if present
		if len(value) >= 2 {
			if (strings.HasPrefix(value, `"`) && strings.HasSuffix(value, `"`)) ||
				(strings.HasPrefix(value, `'`) && strings.HasSuffix(value, `'`)) {
				value = value[1 : len(value)-1]
			}
		}

		// Always set from .env file (override any existing environment variables)
		// This ensures .env file is the source of truth
		os.Setenv(key, value)
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("error reading .env file: %w", err)
	}

	return nil
}

// findEnvFile looks for .env file in current directory and parent directories
func findEnvFile() string {
	// Try current directory first
	cwd, err := os.Getwd()
	if err == nil {
		envPath := filepath.Join(cwd, ".env")
		if _, err := os.Stat(envPath); err == nil {
			return envPath
		}
	}

	// Try executable directory
	if exe, err := os.Executable(); err == nil {
		exeDir := filepath.Dir(exe)
		envPath := filepath.Join(exeDir, ".env")
		if _, err := os.Stat(envPath); err == nil {
			return envPath
		}
	}

	// Try parent directories (up to 3 levels)
	if cwd != "" {
		dir := cwd
		for i := 0; i < 3; i++ {
			envPath := filepath.Join(dir, ".env")
			if _, err := os.Stat(envPath); err == nil {
				return envPath
			}
			parent := filepath.Dir(dir)
			if parent == dir {
				break // Reached root
			}
			dir = parent
		}
	}

	return ""
}

// UpdateEnv updates a specific key in the .env file, or appends it if missing.
var envKeyPattern = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*package envloader

import (
	"bufio"
	"fmt"
	"os"
	"regexp"
	"path/filepath"
	"strings"
)

// LoadEnv loads environment variables from a .env file
// It looks for .env in the current directory or project root
func LoadEnv() error {
	// Try to find .env file
	envPath := findEnvFile()
	if envPath == "" {
		// No .env file found, that's okay - use environment variables directly
		return nil
	}

	file, err := os.Open(envPath)
	if err != nil {
		return fmt.Errorf("failed to open .env file: %w", err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	lineNum := 0

	for scanner.Scan() {
		lineNum++
		line := strings.TrimSpace(scanner.Text())

		// Skip empty lines and comments
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// Parse KEY=VALUE
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue // Skip malformed lines
		}

		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])

		// Remove quotes if present
		if len(value) >= 2 {
			if (strings.HasPrefix(value, `"`) && strings.HasSuffix(value, `"`)) ||
				(strings.HasPrefix(value, `'`) && strings.HasSuffix(value, `'`)) {
				value = value[1 : len(value)-1]
			}
		}

		// Always set from .env file (override any existing environment variables)
		// This ensures .env file is the source of truth
		os.Setenv(key, value)
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("error reading .env file: %w", err)
	}

	return nil
}

// findEnvFile looks for .env file in current directory and parent directories
func findEnvFile() string {
	// Try current directory first
	cwd, err := os.Getwd()
	if err == nil {
		envPath := filepath.Join(cwd, ".env")
		if _, err := os.Stat(envPath); err == nil {
			return envPath
		}
	}

	// Try executable directory
	if exe, err := os.Executable(); err == nil {
		exeDir := filepath.Dir(exe)
		envPath := filepath.Join(exeDir, ".env")
		if _, err := os.Stat(envPath); err == nil {
			return envPath
		}
	}

	// Try parent directories (up to 3 levels)
	if cwd != "" {
		dir := cwd
		for i := 0; i < 3; i++ {
			envPath := filepath.Join(dir, ".env")
			if _, err := os.Stat(envPath); err == nil {
				return envPath
			}
			parent := filepath.Dir(dir)
			if parent == dir {
				break // Reached root
			}
			dir = parent
		}
	}

	return ""
}

// UpdateEnv updates a specific key in the .env file, or appends it if missing.
)

// UpdateEnv handles credentials, so the generated file must not be world readable.
func UpdateEnv(key, value string) error {
	if !envKeyPattern.MatchString(key) || strings.ContainsAny(value, "\n\r") {
		return fmt.Errorf("invalid environment key or multi-line value")
	}
	if err := os.Setenv(key, value); err != nil {
		return err
	}
	envPath := findEnvFile()
	if envPath == "" {
		// Create .env in current directory
		cwd, err := os.Getwd()
		if err != nil {
			return err
		}
		envPath = filepath.Join(cwd, ".env")
		return writePrivateEnvFile(envPath, []byte(fmt.Sprintf("%s=\"%s\"\n", key, value)))
	}

	content, err := os.ReadFile(envPath)
	if err != nil {
		return err
	}
	lines := strings.Split(string(content), "\n")
	found := false
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}
		parts := strings.SplitN(trimmed, "=", 2)
		if len(parts) == 2 && strings.TrimSpace(parts[0]) == key {
			lines[i] = fmt.Sprintf("%s=\"%s\"", key, value)
			found = true
			break
		}
	}
	if !found {
		if len(lines) > 0 && lines[len(lines)-1] != "" {
			lines = append(lines, "")
		}
		lines = append(lines, fmt.Sprintf("%s=\"%s\"", key, value))
	}
	
	newContent := strings.Join(lines, "\n")
	if !strings.HasSuffix(newContent, "\n") {
		newContent += "\n"
	}
	return writePrivateEnvFile(envPath, []byte(newContent))
}

// Reject symlink destinations, enforce owner-only permissions, and replace atomically.
func writePrivateEnvFile(path string, content []byte) error {
	if stat, err := os.Lstat(path); err == nil {
		if !stat.Mode().IsRegular() {
			return fmt.Errorf("refusing to overwrite non-regular .env file")
		}
	} else if !os.IsNotExist(err) {
		return err
	}
	tmp, err := os.CreateTemp(filepath.Dir(path), ".autoar-env-*")
	if err != nil { return err }
	defer os.Remove(tmp.Name())
	if err := tmp.Chmod(0600); err != nil { tmp.Close(); return err }
	if _, err := tmp.Write(content); err != nil { tmp.Close(); return err }
	if err := tmp.Sync(); err != nil { tmp.Close(); return err }
	if err := tmp.Close(); err != nil { return err }
	return os.Rename(tmp.Name(), path)
}
