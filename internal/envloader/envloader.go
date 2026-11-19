package envloader

import (
	"bufio"
	"fmt"
	"os"
	"regexp"
	"strconv"
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

		// strconv.Unquote correctly reverses escaping by UpdateEnv.
		if strings.HasPrefix(value, "\"") {
			decoded, err := strconv.Unquote(value)
			if err != nil { return fmt.Errorf("invalid quoted env value at line %d: %w", lineNum, err) }
			value = decoded
		} else if len(value) >= 2 && strings.HasPrefix(value, "'") && strings.HasSuffix(value, "'") {
			value = value[1:len(value)-1]
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
var envKeyPattern = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)

func UpdateEnv(key, value string) error {
	if !envKeyPattern.MatchString(key) || strings.ContainsAny(value, "\n\r") {
		return fmt.Errorf("invalid environment input")
	}
	envPath := findEnvFile()
	if envPath == "" {
		// Create .env in current directory
		cwd, err := os.Getwd()
		if err != nil {
			return err
		}
		envPath = filepath.Join(cwd, ".env")
		if err := writePrivateEnvFile(envPath, []byte(fmt.Sprintf("%s=%s\n", key, strconv.Quote(value)))); err != nil { return err }
		return os.Setenv(key, value)
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
			lines[i] = fmt.Sprintf("%s=%s", key, strconv.Quote(value))
			found = true
			break
		}
	}
	if !found {
		if len(lines) > 0 && lines[len(lines)-1] != "" {
			lines = append(lines, "")
		}
		lines = append(lines, fmt.Sprintf("%s=%s", key, strconv.Quote(value)))
	}
	
	newContent := strings.Join(lines, "\n")
	if !strings.HasSuffix(newContent, "\n") {
		newContent += "\n"
	}
	if err := writePrivateEnvFile(envPath, []byte(newContent)); err != nil { return err }
	return os.Setenv(key, value)
}

func writePrivateEnvFile(path string, content []byte) error {
	if st, err := os.Lstat(path); err == nil {
		if !st.Mode().IsRegular() { return fmt.Errorf("refusing non-regular env file") }
	} else if !os.IsNotExist(err) { return err }
	f, err := os.CreateTemp(filepath.Dir(path), ".autoar-env-*")
	if err != nil { return err }
	defer os.Remove(f.Name())
	if err := f.Chmod(0600); err != nil { f.Close(); return err }
	if _, err := f.Write(content); err != nil { f.Close(); return err }
	if err := f.Sync(); err != nil { f.Close(); return err }
	if err := f.Close(); err != nil { return err }
	return os.Rename(f.Name(), path)
}
