package main

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

type stubJobResolver struct {
	paths map[string]string
}

func (s stubJobResolver) JobSourcePath(jobID string) (string, error) {
	path, ok := s.paths[jobID]
	if !ok {
		return "", os.ErrNotExist
	}
	return path, nil
}

func TestJobAudioHandlerRejectsNonJobPaths(t *testing.T) {
	root := t.TempDir()
	audio := filepath.Join(root, "clip.m4a")
	if err := os.WriteFile(audio, []byte("audio"), 0o644); err != nil {
		t.Fatal(err)
	}
	jobID := "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	handler := NewJobAudioHandler(stubJobResolver{paths: map[string]string{jobID: audio}})

	cases := []struct {
		path       string
		wantStatus int
	}{
		{path: "/job-audio/" + jobID, wantStatus: http.StatusOK},
		{path: "/job-audio/not-hex-id", wantStatus: http.StatusNotFound},
		{path: "/job-audio/" + jobID + "/extra", wantStatus: http.StatusNotFound},
		{path: "/job-audio/../" + jobID, wantStatus: http.StatusNotFound},
		{path: "/other/" + jobID, wantStatus: http.StatusNotFound},
		{path: "/job-audio/bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb", wantStatus: http.StatusNotFound},
	}
	for _, tc := range cases {
		req := httptest.NewRequest(http.MethodGet, tc.path, nil)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		if rec.Code != tc.wantStatus {
			t.Fatalf("%s status %d want %d", tc.path, rec.Code, tc.wantStatus)
		}
	}
}

func TestJobAudioHandlerServesRegisteredJobOnly(t *testing.T) {
	root := t.TempDir()
	owned := filepath.Join(root, "owned.m4a")
	other := filepath.Join(root, "other.m4a")
	if err := os.WriteFile(owned, []byte("owned"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(other, []byte("other"), 0o644); err != nil {
		t.Fatal(err)
	}
	jobID := "cccccccccccccccccccccccccccccccc"
	handler := NewJobAudioHandler(stubJobResolver{paths: map[string]string{jobID: owned}})

	req := httptest.NewRequest(http.MethodGet, "/job-audio/"+jobID, nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "owned") {
		t.Fatalf("body %q", rec.Body.String())
	}

	req2 := httptest.NewRequest(http.MethodGet, "/job-audio/"+jobID+"?path="+other, nil)
	rec2 := httptest.NewRecorder()
	handler.ServeHTTP(rec2, req2)
	if rec2.Code != http.StatusOK {
		t.Fatalf("query must not change source: status %d", rec2.Code)
	}
	if strings.Contains(rec2.Body.String(), "other") {
		t.Fatal("must not serve arbitrary path from query")
	}
}
