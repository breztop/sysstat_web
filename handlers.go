package main

import (
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"fmt"
	"math"
	"net/http"
	"strings"
	"time"
)

type server struct {
	sysstatDir string
	sampler    *sampler
	password   string
	authToken  string
	tzOffset   int
}

func (s *server) routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/latest", s.handleLatest)
	mux.HandleFunc("/api/timeseries", s.handleTimeSeries)
	mux.HandleFunc("/api/refresh", s.handleRefresh)
	mux.HandleFunc("/login", s.handleLogin)
	mux.HandleFunc("/logout", s.handleLogout)
	mux.Handle("/", http.FileServer(http.Dir("web")))
	return s.authMiddleware(mux)
}

func (s *server) handleLatest(w http.ResponseWriter, r *http.Request) {
	latest, err := latestSarFile(s.sysstatDir)
	if err != nil {
		httpError(w, http.StatusNotFound, err)
		return
	}

	resp, err := buildTimeSeries([]string{latest}, TimeSeriesOptions{Stride: 1, TZOffsetHours: s.tzOffset})
	if err != nil {
		httpError(w, http.StatusInternalServerError, err)
		return
	}

	writeJSON(w, resp)
}

func (s *server) handleTimeSeries(w http.ResponseWriter, r *http.Request) {
	hours := parseHours(r.URL.Query().Get("hours"))
	daysRaw := r.URL.Query().Get("days")
	if hours > 72 {
		hours = 72
	}
	if hours < 0 {
		hours = 0
	}
	if hours == 0 {
		if daysRaw != "" {
			days := parseDays(daysRaw)
			hours = days * 24
		} else {
			hours = 4
		}
	}

	daysForHours := int(math.Ceil(float64(hours) / 24.0))
	if daysForHours < 1 {
		daysForHours = 1
	}
	if daysForHours > 3 {
		daysForHours = 3
	}

	files, err := latestSarFiles(s.sysstatDir, daysForHours)
	if err != nil {
		httpError(w, http.StatusNotFound, err)
		return
	}

	stride := parseStride(r.URL.Query().Get("stride"))
	if stride == 0 {
		stride = defaultStride(hours)
	}

	resp, err := buildTimeSeries(files, TimeSeriesOptions{Hours: hours, Stride: stride, TZOffsetHours: s.tzOffset})
	if err != nil {
		httpError(w, http.StatusInternalServerError, err)
		return
	}

	resp.Meta["days"] = daysForHours
	resp.Meta["range_hours"] = hours
	resp.Meta["stride"] = stride
	if s.sampler != nil {
		resp.Meta["sampling"] = s.sampler.isRunning()
	}
	writeJSON(w, resp)
}

func (s *server) handleRefresh(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	if s.sampler == nil {
		s.sampler = &sampler{}
	}

	if err := s.sampler.start(s.sysstatDir); err != nil {
		httpError(w, http.StatusInternalServerError, err)
		return
	}

	writeJSON(w, map[string]interface{}{
		"status":       "started",
		"message":      "sampling started in background for 60s",
		"generated_at": time.Now().Format(time.RFC3339),
	})
}

func (s *server) handleLogin(w http.ResponseWriter, r *http.Request) {
	if !s.authEnabled() {
		http.Redirect(w, r, "/", http.StatusFound)
		return
	}

	if r.Method == http.MethodPost {
		if err := r.ParseForm(); err != nil {
			httpError(w, http.StatusBadRequest, err)
			return
		}
		password := r.FormValue("password")
		if s.passwordMatches(password) {
			http.SetCookie(w, &http.Cookie{
				Name:     "sysstat_auth",
				Value:    s.authToken,
				Path:     "/",
				HttpOnly: true,
				SameSite: http.SameSiteLaxMode,
				MaxAge:   60 * 60 * 24 * 7,
			})
			http.Redirect(w, r, "/", http.StatusFound)
			return
		}
		s.renderLogin(w, "Invalid password")
		return
	}

	s.renderLogin(w, "")
}

func (s *server) handleLogout(w http.ResponseWriter, r *http.Request) {
	if !s.authEnabled() {
		http.Redirect(w, r, "/", http.StatusFound)
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     "sysstat_auth",
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   -1,
	})
	http.Redirect(w, r, "/login", http.StatusFound)
}

func (s *server) authMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !s.authEnabled() {
			next.ServeHTTP(w, r)
			return
		}

		path := r.URL.Path
		if path == "/login" || path == "/logout" {
			next.ServeHTTP(w, r)
			return
		}

		if s.isAuthenticated(r) {
			next.ServeHTTP(w, r)
			return
		}

		if strings.HasPrefix(path, "/api/") {
			httpError(w, http.StatusUnauthorized, fmt.Errorf("unauthorized"))
			return
		}
		http.Redirect(w, r, "/login", http.StatusFound)
	})
}

func (s *server) authEnabled() bool {
	return s.password != ""
}

func (s *server) isAuthenticated(r *http.Request) bool {
	cookie, err := r.Cookie("sysstat_auth")
	if err != nil {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(cookie.Value), []byte(s.authToken)) == 1
}

func (s *server) passwordMatches(password string) bool {
	return subtle.ConstantTimeCompare([]byte(password), []byte(s.password)) == 1
}

func (s *server) renderLogin(w http.ResponseWriter, errMsg string) {
	message := ""
	if errMsg != "" {
		message = "<div style=\"color:#b64a2b; margin-top:8px;\">" + errMsg + "</div>"
	}
	page := "<!doctype html>" +
		"<html lang=\"en\">" +
		"<head><meta charset=\"utf-8\" /><meta name=\"viewport\" content=\"width=device-width, initial-scale=1\" />" +
		"<title>Login</title>" +
		"<style>body{font-family:Arial,sans-serif;background:#f8f3e8;margin:0;padding:0;display:flex;align-items:center;justify-content:center;height:100vh;}" +
		".card{background:#fff9ef;border:1px solid #e6dccf;border-radius:16px;padding:24px;box-shadow:0 10px 24px rgba(0,0,0,0.08);min-width:280px;}" +
		"button{border:1px solid #b64a2b;background:#b64a2b;color:#fff;padding:8px 14px;border-radius:999px;cursor:pointer;width:100%;}" +
		"input{width:100%;padding:8px;border:1px solid #e6dccf;border-radius:10px;margin-top:8px;}" +
		"</style></head><body>" +
		"<div class=\"card\"><h2>Sysstat Login</h2><form method=\"post\" action=\"/login\">" +
		"<label>Password</label><input type=\"password\" name=\"password\" autofocus />" +
		"<button type=\"submit\">Enter</button>" +
		message +
		"</form></div></body></html>"

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = w.Write([]byte(page))
}

func buildAuthToken(password string) string {
	sum := sha256.Sum256([]byte(password))
	return hex.EncodeToString(sum[:])
}
