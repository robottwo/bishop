package core

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

type Paths struct {
	HomeDir           string
	DataDir           string
	LogFile           string
	HistoryFile       string
	AnalyticsFile     string
	LatestVersionFile string
}

var defaultPaths *Paths

func ensureDefaultPaths() {
	if defaultPaths == nil {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			panic(err)
		}

		defaultPaths = &Paths{
			HomeDir:           homeDir,
			DataDir:           filepath.Join(homeDir, ".local", "share", "bish"),
			LogFile:           filepath.Join(homeDir, ".local", "share", "bish", "bish.zst"),
			HistoryFile:       filepath.Join(homeDir, ".local", "share", "bish", "history.db"),
			AnalyticsFile:     filepath.Join(homeDir, ".local", "share", "bish", "analytics.db"),
			LatestVersionFile: filepath.Join(homeDir, ".local", "share", "bish", "latest_version.txt"),
		}

		err = os.MkdirAll(defaultPaths.DataDir, 0755)
		if err != nil {
			panic(err)
		}
	}
}

func HomeDir() string {
	ensureDefaultPaths()
	return defaultPaths.HomeDir
}

func DataDir() string {
	ensureDefaultPaths()
	return defaultPaths.DataDir
}

func LogFile() string {
	ensureDefaultPaths()
	return defaultPaths.LogFile
}

func HistoryFile() string {
	ensureDefaultPaths()
	return defaultPaths.HistoryFile
}

func AnalyticsFile() string {
	ensureDefaultPaths()
	return defaultPaths.AnalyticsFile
}

func LatestVersionFile() string {
	ensureDefaultPaths()
	return defaultPaths.LatestVersionFile
}

func LogDir() string {
	ensureDefaultPaths()
	return defaultPaths.DataDir
}

func CleanLogFiles() error {
	ensureDefaultPaths()

	// Find all files matching pattern bish.*.zst
	entries, err := os.ReadDir(defaultPaths.DataDir)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		name := entry.Name()
		// Match pattern: bish.<anything>.zst
		if strings.HasPrefix(name, "bish.") && strings.HasSuffix(name, ".zst") {
			filePath := filepath.Join(defaultPaths.DataDir, name)
			if err := os.Remove(filePath); err != nil {
				return err
			}
		}
	}

	return nil
}

// RotateLogFiles automatically removes old log files to prevent unbounded growth.
// Keeps the most recent 10 log files (based on modification time).
// This is called automatically when creating a new log sink.
func RotateLogFiles() error {
	ensureDefaultPaths()

	// Find all files matching pattern bish.*.zst
	entries, err := os.ReadDir(defaultPaths.DataDir)
	if err != nil {
		return err
	}

	var logFiles []logFileInfo
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		name := entry.Name()
		// Match pattern: bish.<anything>.zst
		if strings.HasPrefix(name, "bish.") && strings.HasSuffix(name, ".zst") {
			info, err := entry.Info()
			if err != nil {
				continue
			}

			logFiles = append(logFiles, logFileInfo{
				name:    name,
				path:    filepath.Join(defaultPaths.DataDir, name),
				modTime: info.ModTime(),
			})
		}
	}

	// If we have more than 10 log files, remove the oldest ones
	const maxLogFiles = 10
	if len(logFiles) <= maxLogFiles {
		return nil
	}

	// Sort by modification time, newest first
	sort.Slice(logFiles, func(i, j int) bool {
		return logFiles[i].modTime.After(logFiles[j].modTime)
	})

	// Remove oldest files beyond the limit
	for i := maxLogFiles; i < len(logFiles); i++ {
		if err := os.Remove(logFiles[i].path); err != nil {
			return err
		}
	}

	return nil
}

type logFileInfo struct {
	name    string
	path    string
	modTime time.Time
}
