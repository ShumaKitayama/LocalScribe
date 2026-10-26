package main

import (
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

const jobAudioPrefix = "/job-audio/"

var jobIDPattern = regexp.MustCompile(`^[0-9a-f]{32}$`)

// JobSourceResolver はジョブ ID から元音声のパスを返す。
type JobSourceResolver interface {
	JobSourcePath(jobID string) (string, error)
}

// NewJobAudioHandler はジョブ ID だけを受け付ける GET ハンドラを返す。
func NewJobAudioHandler(resolver JobSourceResolver) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		if !strings.HasPrefix(r.URL.Path, jobAudioPrefix) {
			http.NotFound(w, r)
			return
		}
		jobID := strings.TrimPrefix(r.URL.Path, jobAudioPrefix)
		if jobID == "" || strings.Contains(jobID, "/") || !jobIDPattern.MatchString(jobID) {
			http.NotFound(w, r)
			return
		}
		sourcePath, err := resolver.JobSourcePath(jobID)
		if err != nil {
			http.NotFound(w, r)
			return
		}
		clean := filepath.Clean(sourcePath)
		info, err := os.Stat(clean)
		if err != nil || info.IsDir() {
			http.NotFound(w, r)
			return
		}
		http.ServeFile(w, r, clean)
	})
}
