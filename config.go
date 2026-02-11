package main

import (
	"encoding/json"
	"fmt"
	"os"
)

type Config struct {
	SysstatDir    string `json:"sysstat_dir"`
	Port          string `json:"port"`
	Password      string `json:"password"`
	TZOffsetHours int    `json:"tz_offset_hours"`
}

func loadConfig(path string) (Config, error) {
	cfg := Config{
		SysstatDir:    "/var/log/sysstat",
		Port:          "",
		Password:      "",
		TZOffsetHours: 8,
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return cfg, fmt.Errorf("read config %s: %w", path, err)
	}

	if err := json.Unmarshal(data, &cfg); err != nil {
		return cfg, fmt.Errorf("parse config %s: %w", path, err)
	}

	if cfg.SysstatDir == "" {
		return cfg, fmt.Errorf("sysstat_dir is required in config")
	}
	if cfg.Port == "" {
		return cfg, fmt.Errorf("port is required in config")
	}
	if cfg.Password == "" {
		return cfg, fmt.Errorf("password is required in config")
	}

	return cfg, nil
}
