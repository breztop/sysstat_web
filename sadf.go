package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"sync"
	"time"
)

type sadfCacheEntry struct {
	mtime time.Time
	data  map[string]interface{}
}

var sadfCache = struct {
	mu    sync.Mutex
	items map[string]sadfCacheEntry
}{
	items: map[string]sadfCacheEntry{},
}

func runSadfJSON(file string) (map[string]interface{}, error) {
	info, err := os.Stat(file)
	if err != nil {
		return nil, err
	}
	mtime := info.ModTime()

	sadfCache.mu.Lock()
	if entry, ok := sadfCache.items[file]; ok && entry.mtime.Equal(mtime) {
		data := entry.data
		sadfCache.mu.Unlock()
		return data, nil
	}
	sadfCache.mu.Unlock()

	// Limit to metrics we actually use to speed up sadf.
	cmd := exec.Command("sadf", "-j", file, "--", "-u", "-r", "-d", "-n", "DEV", "-S", "-q", "-b")
	out, err := cmd.Output()
	if err != nil {
		if ee, ok := err.(*exec.ExitError); ok {
			return nil, fmt.Errorf("sadf failed: %s", string(ee.Stderr))
		}
		return nil, err
	}

	var raw map[string]interface{}
	if err := json.Unmarshal(out, &raw); err != nil {
		return nil, err
	}

	sadfCache.mu.Lock()
	sadfCache.items[file] = sadfCacheEntry{mtime: mtime, data: raw}
	sadfCache.mu.Unlock()
	return raw, nil
}

func extractHost(raw map[string]interface{}) (map[string]interface{}, error) {
	sysstat, ok := raw["sysstat"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid sadf json: missing sysstat")
	}

	hosts, ok := sysstat["hosts"].([]interface{})
	if !ok || len(hosts) == 0 {
		return nil, fmt.Errorf("invalid sadf json: missing hosts")
	}

	host, ok := hosts[0].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid sadf json: invalid host")
	}

	return host, nil
}
