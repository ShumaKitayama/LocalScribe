package application

import (
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/ShumaKitayama/localscribe/internal/domain"
	"github.com/ShumaKitayama/localscribe/internal/platform/macos"
)

func testCatalog(t *testing.T, url string, minBytes int64, sha string) []CatalogEntry {
	fileName, ok := domain.ModelFileName(domain.ModelLargeV3)
	if !ok {
		t.Fatal("file name")
	}
	return []CatalogEntry{{
		ID:       domain.ModelLargeV3,
		FileName: fileName,
		URL:      url,
		MinBytes: minBytes,
		SHA256:   sha,
	}}
}

func newTestModelService(t *testing.T, home string, catalog []CatalogEntry, client *http.Client) *ModelService {
	svc, err := NewModelService(ModelServiceConfig{
		Home:    home,
		Catalog: catalog,
		HTTP:    client,
		FreeBytes: func(string) (int64, error) {
			return 1 << 40, nil
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	return svc
}

func TestInstallSuccessWritesManifestAndListAvailable(t *testing.T) {
	payload := []byte("model-payload-for-test-install")
	sum := sha256.Sum256(payload)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Write(payload)
	}))
	defer server.Close()

	home := t.TempDir()
	catalog := testCatalog(t, server.URL, int64(len(payload)), "")
	svc := newTestModelService(t, home, catalog, server.Client())

	if err := svc.Install(t.Context(), domain.ModelLargeV3); err != nil {
		t.Fatalf("Install: %v", err)
	}

	finalPath := filepath.Join(macos.ModelsDir(home), catalog[0].FileName)
	if _, err := os.Stat(finalPath + ".partial"); !os.IsNotExist(err) {
		t.Fatal("partial should be gone")
	}
	if _, err := os.Stat(finalPath); err != nil {
		t.Fatalf("final missing: %v", err)
	}

	list, err := svc.List()
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 1 || list[0].Status != domain.ModelAvailable {
		t.Fatalf("list: %+v", list)
	}
	if list[0].SHA256 != hex.EncodeToString(sum[:]) {
		t.Fatalf("sha %q", list[0].SHA256)
	}
}

func TestInterruptedDownloadLeavesPartialAndNotAvailable(t *testing.T) {
	payload := []byte("partial-chunk")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Length", "9999")
		_, _ = w.Write(payload)
	}))
	defer server.Close()

	home := t.TempDir()
	catalog := testCatalog(t, server.URL, 100, "")
	svc := newTestModelService(t, home, catalog, server.Client())

	err := svc.Install(t.Context(), domain.ModelLargeV3)
	if err == nil {
		t.Fatal("expected error")
	}

	finalPath := filepath.Join(macos.ModelsDir(home), catalog[0].FileName)
	if _, err := os.Stat(finalPath + ".partial"); err != nil {
		t.Fatalf("partial should remain: %v", err)
	}
	if _, err := os.Stat(finalPath); !os.IsNotExist(err) {
		t.Fatal("final must not exist")
	}
	list, err := svc.List()
	if err != nil {
		t.Fatal(err)
	}
	if list[0].Status == domain.ModelAvailable {
		t.Fatalf("status %s", list[0].Status)
	}
}

func TestUndersizeDownloadRemovesPartial(t *testing.T) {
	payload := []byte("tiny")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Write(payload)
	}))
	defer server.Close()

	home := t.TempDir()
	catalog := testCatalog(t, server.URL, 100, "")
	svc := newTestModelService(t, home, catalog, server.Client())

	if err := svc.Install(t.Context(), domain.ModelLargeV3); err == nil {
		t.Fatal("expected undersize error")
	}
	finalPath := filepath.Join(macos.ModelsDir(home), catalog[0].FileName)
	if _, err := os.Stat(finalPath + ".partial"); !os.IsNotExist(err) {
		t.Fatal("partial should be removed after undersize")
	}
}

func TestInstallDoesNotOverwriteExistingFinal(t *testing.T) {
	payload := []byte("already-there")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Write([]byte("new-data"))
	}))
	defer server.Close()

	home := t.TempDir()
	catalog := testCatalog(t, server.URL, int64(len(payload)), "")
	modelDir := macos.ModelsDir(home)
	if err := os.MkdirAll(modelDir, 0o755); err != nil {
		t.Fatal(err)
	}
	finalPath := filepath.Join(modelDir, catalog[0].FileName)
	if err := os.WriteFile(finalPath, payload, 0o644); err != nil {
		t.Fatal(err)
	}

	svc := newTestModelService(t, home, catalog, server.Client())
	if err := svc.Install(t.Context(), domain.ModelLargeV3); err == nil {
		t.Fatal("expected already installed error")
	}
	got, err := os.ReadFile(finalPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != string(payload) {
		t.Fatalf("file overwritten: %q", got)
	}
}

func TestSHA256MismatchRejected(t *testing.T) {
	payload := []byte("consistent-model-bytes")
	sum := sha256.Sum256(payload)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Write([]byte("different-bytes-content"))
	}))
	defer server.Close()

	home := t.TempDir()
	catalog := testCatalog(t, server.URL, int64(len(payload)), hex.EncodeToString(sum[:]))
	svc := newTestModelService(t, home, catalog, server.Client())

	if err := svc.Install(t.Context(), domain.ModelLargeV3); err == nil {
		t.Fatal("expected hash mismatch")
	}
	finalPath := filepath.Join(macos.ModelsDir(home), catalog[0].FileName)
	if _, err := os.Stat(finalPath); !os.IsNotExist(err) {
		t.Fatal("final must not exist")
	}
	if _, err := os.Stat(finalPath + ".partial"); !os.IsNotExist(err) {
		t.Fatal("partial should be removed after hash mismatch")
	}
}

func TestReinstallChecksManifestSHA(t *testing.T) {
	first := []byte("model-payload-for-test-install")
	var calls int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		calls++
		if calls == 1 {
			_, _ = w.Write(first)
			return
		}
		_, _ = w.Write([]byte("model-payload-for-test-changed"))
	}))
	defer server.Close()

	home := t.TempDir()
	catalog := testCatalog(t, server.URL, int64(len(first)), "")
	svc := newTestModelService(t, home, catalog, server.Client())
	if err := svc.Install(t.Context(), domain.ModelLargeV3); err != nil {
		t.Fatalf("first Install: %v", err)
	}
	finalPath := filepath.Join(macos.ModelsDir(home), catalog[0].FileName)
	if err := os.Remove(finalPath); err != nil {
		t.Fatal(err)
	}
	if err := svc.Install(t.Context(), domain.ModelLargeV3); err == nil {
		t.Fatal("expected manifest hash mismatch")
	}
	if _, err := os.Stat(finalPath); !os.IsNotExist(err) {
		t.Fatal("final must not exist")
	}
	if _, err := os.Stat(finalPath + ".partial"); !os.IsNotExist(err) {
		t.Fatal("partial should be removed after manifest mismatch")
	}
}

type denyHTTP struct {
	t *testing.T
}

func (d denyHTTP) Do(*http.Request) (*http.Response, error) {
	d.t.Fatal("空き容量不足なら HTTP を呼ばない")
	return nil, nil
}

func TestInstallRejectsLowDisk(t *testing.T) {
	home := t.TempDir()
	catalog := testCatalog(t, "http://127.0.0.1:1/unused", 100, "")
	svc, err := NewModelService(ModelServiceConfig{
		Home:    home,
		Catalog: catalog,
		HTTP:    denyHTTP{t: t},
		FreeBytes: func(string) (int64, error) {
			return 1, nil
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := svc.Install(t.Context(), domain.ModelLargeV3); err == nil {
		t.Fatal("expected low disk error")
	}
	finalPath := filepath.Join(macos.ModelsDir(home), catalog[0].FileName)
	if _, err := os.Stat(finalPath); !os.IsNotExist(err) {
		t.Fatal("final must not exist")
	}
	if _, err := os.Stat(finalPath + ".partial"); !os.IsNotExist(err) {
		t.Fatal("partial must not exist")
	}
}
