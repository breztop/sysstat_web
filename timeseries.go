package main

import (
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

type SeriesBlock struct {
	Labels []string              `json:"labels"`
	Series map[string][]*float64 `json:"series"`
}

type TimeSeries struct {
	Meta   map[string]interface{} `json:"meta"`
	CPU    *SeriesBlock           `json:"cpu,omitempty"`
	Memory *SeriesBlock           `json:"memory,omitempty"`
	Disk   *SeriesBlock           `json:"disk,omitempty"`
	Net    *SeriesBlock           `json:"net,omitempty"`
	Swap   *SeriesBlock           `json:"swap,omitempty"`
	Load   *SeriesBlock           `json:"load,omitempty"`
	IO     *SeriesBlock           `json:"io,omitempty"`
}

type TimeSeriesOptions struct {
	Hours         int
	Stride        int
	TZOffsetHours int
}

type statEntry struct {
	ts      time.Time
	label   string
	stat    map[string]interface{}
	hasTime bool
}

func (b *SeriesBlock) add(label string, values map[string]*float64) {
	b.Labels = append(b.Labels, label)
	if b.Series == nil {
		b.Series = map[string][]*float64{}
	}
	for key := range b.Series {
		b.Series[key] = append(b.Series[key], nil)
	}
	for key, value := range values {
		if _, ok := b.Series[key]; !ok {
			b.Series[key] = make([]*float64, len(b.Labels)-1)
			b.Series[key] = append(b.Series[key], value)
			continue
		}
		b.Series[key][len(b.Series[key])-1] = value
	}
}

func buildTimeSeries(files []string, opts TimeSeriesOptions) (*TimeSeries, error) {
	resp := &TimeSeries{
		Meta: map[string]interface{}{
			"files":        baseNames(files),
			"generated_at": time.Now().Format(time.RFC3339),
		},
	}

	var entries []statEntry
	var maxTime time.Time

	for _, file := range files {
		raw, err := runSadfJSON(file)
		if err != nil {
			return nil, err
		}

		host, err := extractHost(raw)
		if err != nil {
			return nil, err
		}

		if hostName, ok := host["nodename"].(string); ok {
			resp.Meta["host"] = hostName
		}

		stats, ok := host["statistics"].([]interface{})
		if !ok {
			continue
		}

		for _, statRaw := range stats {
			stat, ok := statRaw.(map[string]interface{})
			if !ok {
				continue
			}
			label := buildLabel(stat)
			ts, ok := buildTimestamp(stat, opts.TZOffsetHours)
			if ok && ts.After(maxTime) {
				maxTime = ts
			}
			entries = append(entries, statEntry{
				ts:      ts,
				label:   label,
				stat:    stat,
				hasTime: ok,
			})
		}
	}

	cutoff := time.Time{}
	if opts.Hours > 0 && !maxTime.IsZero() {
		cutoff = maxTime.Add(-time.Duration(opts.Hours) * time.Hour)
	}

	stride := opts.Stride
	if stride < 1 {
		stride = 1
	}

	filteredIndex := 0
	for _, entry := range entries {
		if !cutoff.IsZero() {
			if !entry.hasTime || entry.ts.Before(cutoff) {
				continue
			}
		}
		filteredIndex++
		if stride > 1 && (filteredIndex-1)%stride != 0 {
			continue
		}

		label := entry.label
		if entry.hasTime {
			label = entry.ts.Format("2006-01-02 15:04:05")
		} else if label == "" {
			label = "-"
		}
		stat := entry.stat

		if values := extractCPU(stat); values != nil {
			if resp.CPU == nil {
				resp.CPU = &SeriesBlock{}
			}
			resp.CPU.add(label, values)
		}

		if values := extractMemory(stat); values != nil {
			if resp.Memory == nil {
				resp.Memory = &SeriesBlock{}
			}
			resp.Memory.add(label, values)
		}

		if values := extractDisk(stat); values != nil {
			if resp.Disk == nil {
				resp.Disk = &SeriesBlock{}
			}
			resp.Disk.add(label, values)
		}

		if values := extractNet(stat); values != nil {
			if resp.Net == nil {
				resp.Net = &SeriesBlock{}
			}
			resp.Net.add(label, values)
		}

		if values := extractSwap(stat); values != nil {
			if resp.Swap == nil {
				resp.Swap = &SeriesBlock{}
			}
			resp.Swap.add(label, values)
		}

		if values := extractLoad(stat); values != nil {
			if resp.Load == nil {
				resp.Load = &SeriesBlock{}
			}
			resp.Load.add(label, values)
		}

		if values := extractIO(stat); values != nil {
			if resp.IO == nil {
				resp.IO = &SeriesBlock{}
			}
			resp.IO.add(label, values)
		}
	}

	return resp, nil
}

func buildLabel(stat map[string]interface{}) string {
	ts, ok := stat["timestamp"].(map[string]interface{})
	if !ok {
		return ""
	}
	date, _ := ts["date"].(string)
	timePart, _ := ts["time"].(string)
	label := strings.TrimSpace(date + " " + timePart)
	return label
}

func buildTimestamp(stat map[string]interface{}, offsetHours int) (time.Time, bool) {
	ts, ok := stat["timestamp"].(map[string]interface{})
	if !ok {
		return time.Time{}, false
	}
	date, _ := ts["date"].(string)
	timePart, _ := ts["time"].(string)
	if date == "" || timePart == "" {
		return time.Time{}, false
	}
	value := strings.TrimSpace(date + " " + timePart)
	if value == "" {
		return time.Time{}, false
	}
	if t, err := time.ParseInLocation("2006-01-02 15:04:05", value, time.Local); err == nil {
		return t.Add(time.Duration(offsetHours) * time.Hour), true
	}
	if t, err := time.ParseInLocation("2006-01-02 15:04", value, time.Local); err == nil {
		return t.Add(time.Duration(offsetHours) * time.Hour), true
	}
	return time.Time{}, false
}

func extractCPU(stat map[string]interface{}) map[string]*float64 {
	items := toList(stat["cpu-load"])
	if len(items) == 0 {
		return nil
	}
	for _, item := range items {
		m := toMap(item)
		if m == nil {
			continue
		}
		cpu, _ := m["cpu"].(string)
		if cpu != "all" {
			continue
		}
		return map[string]*float64{
			"user":   floatPtr(m["user"]),
			"system": floatPtr(m["system"]),
			"iowait": floatPtr(m["iowait"]),
			"idle":   floatPtr(m["idle"]),
			"steal":  floatPtr(m["steal"]),
		}
	}
	return nil
}

func extractMemory(stat map[string]interface{}) map[string]*float64 {
	items := toList(stat["memory"])
	if len(items) == 0 {
		return nil
	}
	m := toMap(items[0])
	if m == nil {
		return nil
	}
	return map[string]*float64{
		"used_pct": floatPtr(m["memused-percent"]),
		"free_kb":  floatPtr(m["memfree"]),
		"buffers":  floatPtr(m["buffers"]),
		"cached":   floatPtr(m["cached"]),
	}
}

func extractDisk(stat map[string]interface{}) map[string]*float64 {
	items := toList(stat["disk"])
	if len(items) == 0 {
		return nil
	}

	var tps, rdSec, wrSec float64
	var has bool
	for _, item := range items {
		m := toMap(item)
		if m == nil {
			continue
		}
		if v, ok := toFloat(m["tps"]); ok {
			tps += v
			has = true
		}
		if v, ok := toFloat(m["rd_sec"]); ok {
			rdSec += v
			has = true
		}
		if v, ok := toFloat(m["wr_sec"]); ok {
			wrSec += v
			has = true
		}
	}
	if !has {
		return nil
	}
	return map[string]*float64{
		"tps":    floatPtr(tps),
		"rd_sec": floatPtr(rdSec),
		"wr_sec": floatPtr(wrSec),
	}
}

func extractNet(stat map[string]interface{}) map[string]*float64 {
	items := toList(stat["net-dev"])
	if len(items) == 0 {
		return nil
	}
	var rx, tx float64
	var has bool
	for _, item := range items {
		m := toMap(item)
		if m == nil {
			continue
		}
		iface, _ := m["iface"].(string)
		if iface == "lo" || iface == "lo0" {
			continue
		}
		if v, ok := toFloat(m["rxkB"]); ok {
			rx += v
			has = true
		}
		if v, ok := toFloat(m["txkB"]); ok {
			tx += v
			has = true
		}
	}
	if !has {
		return nil
	}
	return map[string]*float64{
		"rx_kb": floatPtr(rx),
		"tx_kb": floatPtr(tx),
	}
}

func extractSwap(stat map[string]interface{}) map[string]*float64 {
	items := toList(stat["swap-pages"])
	if len(items) == 0 {
		return nil
	}
	m := toMap(items[0])
	if m == nil {
		return nil
	}
	return map[string]*float64{
		"pswpin":  floatPtr(m["pswpin"]),
		"pswpout": floatPtr(m["pswpout"]),
	}
}

func extractLoad(stat map[string]interface{}) map[string]*float64 {
	items := toList(stat["queue"])
	if len(items) == 0 {
		items = toList(stat["load-queue"])
	}
	if len(items) == 0 {
		return nil
	}
	m := toMap(items[0])
	if m == nil {
		return nil
	}
	return map[string]*float64{
		"ldavg1":  floatPtr(m["ldavg-1"]),
		"ldavg5":  floatPtr(m["ldavg-5"]),
		"ldavg15": floatPtr(m["ldavg-15"]),
	}
}

func extractIO(stat map[string]interface{}) map[string]*float64 {
	items := toList(stat["io"])
	if len(items) == 0 {
		return nil
	}
	m := toMap(items[0])
	if m == nil {
		return nil
	}
	return map[string]*float64{
		"tps":  floatPtr(m["tps"]),
		"rtps": floatPtr(m["rtps"]),
		"wtps": floatPtr(m["wtps"]),
	}
}

func toList(v interface{}) []interface{} {
	if v == nil {
		return nil
	}
	if list, ok := v.([]interface{}); ok {
		return list
	}
	return nil
}

func toMap(v interface{}) map[string]interface{} {
	if v == nil {
		return nil
	}
	if m, ok := v.(map[string]interface{}); ok {
		return m
	}
	return nil
}

func floatPtr(v interface{}) *float64 {
	f, ok := toFloat(v)
	if !ok {
		return nil
	}
	return &f
}

func toFloat(v interface{}) (float64, bool) {
	switch t := v.(type) {
	case float64:
		return t, true
	case int:
		return float64(t), true
	case int64:
		return float64(t), true
	case string:
		if t == "" {
			return 0, false
		}
		f, err := strconv.ParseFloat(t, 64)
		if err != nil {
			return 0, false
		}
		return f, true
	default:
		return 0, false
	}
}

func baseNames(files []string) []string {
	out := make([]string, 0, len(files))
	for _, file := range files {
		out = append(out, filepath.Base(file))
	}
	return out
}
