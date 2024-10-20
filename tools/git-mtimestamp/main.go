package main

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/spf13/cobra"
)

var verbose bool

func main() {
	var rootCmd = &cobra.Command{
		Use:   "git-mtimestamp",
		Short: "Updates file and directory timestamps based on Git history",
		RunE:  run,
	}
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "Enable verbose logging")

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func run(cmd *cobra.Command, args []string) error {
	start := time.Now()
	if verbose {
		log.SetFlags(log.Ltime | log.Lmicroseconds)
	} else {
		log.SetOutput(io.Discard)
	}

	gitRoot, err := getGitRoot()
	if err != nil {
		return err
	}

	files, err := getTrackedFiles(gitRoot)
	if err != nil {
		return err
	}

	directories := getUniqueDirectories(files)

	fileTimestamps, err := getTimestamps(gitRoot, files, "files", "--")
	if err != nil {
		return err
	}

	dirTimestamps, err := getTimestamps(gitRoot, directories, "directories", "--full-history", "--")
	if err != nil {
		return err
	}

	processedCount, updatedCount := processFilesAndDirs(gitRoot, files, directories, fileTimestamps, dirTimestamps)

	fmt.Printf("Processed %d items, updated %d items in %v\n",
		processedCount, updatedCount, time.Since(start))
	return nil
}

func getTimestamps(gitRoot string, items []string, itemType string, extraArgs ...string) (map[string]int64, error) {
	timestamps := make(map[string]int64)
	args := append([]string{"log", "--name-only", "--no-renames", "--format=%ct", "-z"}, extraArgs...)
	// nolint:gosec
	cmd := exec.Command("git", args...)
	cmd.Dir = gitRoot
	cmd.Stdin = strings.NewReader(strings.Join(items, "\x00"))
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to get Git log for %s: %w", itemType, err)
	}

	var currentTime int64
	var timestampCount int
	elements := bytes.Split(output, []byte{0})
	for i := 0; i < len(elements); i++ {
		item := string(bytes.TrimSpace(elements[i]))
		if item == "" {
			continue
		}
		if t, err := strconv.ParseInt(item, 10, 64); err == nil {
			currentTime = t
			timestampCount++
		} else {
			if _, exists := timestamps[item]; !exists {
				timestamps[item] = currentTime
				if verbose {
					log.Printf("%s: %s, Timestamp: %s", itemType, item, time.Unix(currentTime, 0))
				}
			}
		}
	}

	log.Printf("Got timestamps for %d %s", len(timestamps), itemType)
	log.Printf("Found %d unique timestamps for %s", timestampCount, itemType)

	return timestamps, nil
}

func processFilesAndDirs(gitRoot string, files, directories []string, fileTimestamps, dirTimestamps map[string]int64) (int, int) {
	var processedCount, updatedCount int32
	var wg sync.WaitGroup
	semaphore := make(chan struct{}, 100) // Limit concurrent goroutines

	processItems := func(items []string, timestamps map[string]int64, itemType string) {
		for _, item := range items {
			wg.Add(1)
			semaphore <- struct{}{}
			go func(item string) {
				defer wg.Done()
				defer func() { <-semaphore }()
				if timestamp, ok := timestamps[item]; ok {
					if updateTime(gitRoot, item, timestamp) {
						atomic.AddInt32(&updatedCount, 1)
					}
					atomic.AddInt32(&processedCount, 1)
				} else if verbose {
					log.Printf("No timestamp found for %s: %s", itemType, item)
				}
			}(item)
		}
	}

	processItems(files, fileTimestamps, "file")
	processItems(directories, dirTimestamps, "directory")

	wg.Wait()
	return int(processedCount), int(updatedCount)
}

func updateTime(gitRoot, item string, timestamp int64) bool {
	fullPath := filepath.Join(gitRoot, item)
	fileInfo, err := os.Stat(fullPath)
	if err != nil {
		log.Printf("Failed to stat %s: %v", item, err)
		return false
	}

	currentModTime := fileInfo.ModTime().Unix()
	if currentModTime == timestamp {
		return false
	}

	newTime := time.Unix(timestamp, 0)
	err = os.Chtimes(fullPath, newTime, newTime)
	if err != nil {
		log.Printf("Failed to update time for %s: %v", item, err)
		return false
	}

	if verbose {
		log.Printf("Updated time for %s from %s to %s", item,
			time.Unix(currentModTime, 0), newTime)
	}
	return true
}

func getGitRoot() (string, error) {
	cmd := exec.Command("git", "rev-parse", "--show-toplevel")
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to get Git root: %w", err)
	}
	return strings.TrimSpace(string(output)), nil
}

func getTrackedFiles(gitRoot string) ([]string, error) {
	cmd := exec.Command("git", "ls-files", "-z")
	cmd.Dir = gitRoot
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to get tracked files: %w", err)
	}

	files := bytes.Split(bytes.TrimSuffix(output, []byte{0}), []byte{0})
	result := make([]string, 0, len(files))
	for _, file := range files {
		if len(file) > 0 {
			result = append(result, string(file))
		}
	}
	sort.Strings(result)
	return result, nil
}
func getUniqueDirectories(files []string) []string {
	dirSet := make(map[string]bool)
	for _, file := range files {
		dir := filepath.Dir(file)
		for dir != "." && !dirSet[dir] {
			dirSet[dir] = true
			dir = filepath.Dir(dir)
		}
	}

	dirs := make([]string, 0, len(dirSet))
	for dir := range dirSet {
		dirs = append(dirs, dir)
	}
	sort.Strings(dirs)
	return dirs
}
