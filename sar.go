package main

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"
)

func latestSarFile(sysstatDir string) (string, error) {
	files, err := latestSarFiles(sysstatDir, 1)
	if err != nil {
		return "", err
	}
	return files[len(files)-1], nil
}

func latestSarFiles(sysstatDir string, days int) ([]string, error) {
	if days < 1 {
		days = 1
	}

	var files []string
	for i := days - 1; i >= 0; i-- {
		t := time.Now().AddDate(0, 0, -i)
		name := "sa" + t.Format("02")
		path := filepath.Join(sysstatDir, name)
		if _, err := os.Stat(path); err == nil {
			files = append(files, path)
		}
	}

	if len(files) == 0 {
		return nil, fmt.Errorf("no sar files found in %s", sysstatDir)
	}

	sort.Strings(files)
	return files, nil
}
