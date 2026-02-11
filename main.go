package main

import (
	"log"
	"net/http"
	"os"
)

func main() {
	configPath := os.Getenv("CONFIG_FILE")
	if configPath == "" {
		configPath = "config.json"
	}

	cfg, err := loadConfig(configPath)
	if err != nil {
		log.Fatal(err)
	}

	srv := &server{
		sysstatDir: cfg.SysstatDir,
		password:   cfg.Password,
		authToken:  buildAuthToken(cfg.Password),
		tzOffset:   cfg.TZOffsetHours,
	}

	log.Printf("sysstat web running on http://localhost:%s", cfg.Port)
	log.Printf("using sysstat dir: %s", cfg.SysstatDir)
	if err := http.ListenAndServe(":"+cfg.Port, srv.routes()); err != nil {
		log.Fatal(err)
	}
}
