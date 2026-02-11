package main

import (
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"strconv"
	"strings"
)

func parseDays(raw string) int {
	if raw == "" {
		return 1
	}
	days, err := strconv.Atoi(raw)
	if err != nil {
		return 1
	}
	if days < 1 {
		return 1
	}
	if days > 3 {
		return 3
	}
	return days
}

func parseHours(raw string) int {
	if raw == "" {
		return 0
	}
	trimmed := strings.TrimSpace(strings.ToLower(raw))
	if before, ok := strings.CutSuffix(trimmed, "h"); ok {
		trimmed = before
	}
	if before, ok :=strings.CutSuffix(trimmed, "d"); ok  {
		trimmed = before
		days, err := strconv.Atoi(trimmed)
		if err != nil {
			return 0
		}
		return days * 24
	}
	hours, err := strconv.Atoi(trimmed)
	if err != nil {
		return 0
	}
	return hours
}

func parseStride(raw string) int {
	if raw == "" {
		return 0
	}
	stride, err := strconv.Atoi(raw)
	if err != nil {
		return 0
	}
	if stride < 1 {
		return 1
	}
	return stride
}

func defaultStride(hours int) int {
	if hours <= 0 {
		return 1
	}
	stride := int(math.Ceil(float64(hours) / 4.0))
	if stride < 1 {
		return 1
	}
	return stride
}

func writeJSON(w http.ResponseWriter, payload interface{}) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	if err := json.NewEncoder(w).Encode(payload); err != nil {
		httpError(w, http.StatusInternalServerError, err)
		return
	}
}

func httpError(w http.ResponseWriter, status int, err error) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]string{
		"error": fmt.Sprintf("%v", err),
	})
}
