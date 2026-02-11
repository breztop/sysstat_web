package main

import (
	"context"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"time"
)

type sampler struct {
	mu      sync.Mutex
	cancel  context.CancelFunc
	active  bool
	current context.Context
}

func (s *sampler) start(sysstatDir string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.cancel != nil {
		s.cancel()
	}

	ctx, cancel := context.WithCancel(context.Background())
	s.cancel = cancel
	s.current = ctx
	s.active = true

	if err := os.MkdirAll(sysstatDir, 0755); err != nil {
		return err
	}

	filePath := filepath.Join(sysstatDir, "sa"+time.Now().Format("02"))

	go func() {
		timeoutCtx, timeoutCancel := context.WithTimeout(ctx, 70*time.Second)
		defer timeoutCancel()

		defer func() {
			s.mu.Lock()
			if s.current == ctx {
				s.cancel = nil
				s.current = nil
				s.active = false
			}
			s.mu.Unlock()
		}()

		interval := "5"
		count := "12"
		cmd := exec.CommandContext(
			timeoutCtx,
			"sudo",
			"-n",
			"/home/brez/go/src/sysstat-web/scripts/sample_sar.sh",
			filePath,
		)
		cmd.Env = append(
			os.Environ(),
			"SYSSTAT_DIR="+sysstatDir,
			"SAR_INTERVAL="+interval,
			"SAR_COUNT="+count,
		)

		log.Printf("Starting sar sampling to %s", filePath)
		if out, err := cmd.CombinedOutput(); err != nil {
			if timeoutCtx.Err() == context.Canceled {
				log.Printf("Sampling cancelled for %s", filePath)
			} else {
				log.Printf("Sampling failed: %v, output: %s", err, string(out))
			}
		} else {
			log.Printf("Sampling completed for %s", filePath)
		}
	}()

	return nil
}

func (s *sampler) isRunning() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.active
}
