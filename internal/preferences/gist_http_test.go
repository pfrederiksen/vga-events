package preferences

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func withGistTestServer(t *testing.T, handler http.Handler) string {
	t.Helper()
	server := httptest.NewServer(handler)
	original := gistAPIURL
	gistAPIURL = server.URL + "/gists"
	t.Cleanup(func() {
		gistAPIURL = original
		server.Close()
	})
	return server.URL
}

func TestGistStorageLoadTruncatedFile(t *testing.T) {
	var serverURL string
	serverURL = withGistTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer token" {
			t.Errorf("Authorization = %q", r.Header.Get("Authorization"))
		}
		if r.Header.Get("X-GitHub-Api-Version") == "" {
			t.Error("missing GitHub API version header")
		}
		switch r.URL.Path {
		case "/gists/gist":
			w.Header().Set("ETag", `"revision-1"`)
			_, _ = w.Write([]byte(`{"files":{"preferences.json":{"truncated":true,"raw_url":"` + serverURL + `/raw"}}}`))
		case "/raw":
			_, _ = w.Write([]byte(`{"123":{"states":["NV"],"active":true}}`))
		default:
			http.NotFound(w, r)
		}
	}))

	storage, err := NewGistStorage("gist", "token")
	if err != nil {
		t.Fatal(err)
	}
	prefs, err := storage.Load()
	if err != nil {
		t.Fatal(err)
	}
	if got := prefs.GetUser("123").States; len(got) != 1 || got[0] != "NV" {
		t.Fatalf("states = %v, want [NV]", got)
	}
	if storage.etag != `"revision-1"` {
		t.Fatalf("etag = %q", storage.etag)
	}
}

func TestGistStorageSaveUsesLoadedRevision(t *testing.T) {
	withGistTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			w.Header().Set("ETag", `"revision-1"`)
			_, _ = w.Write([]byte(`{"files":{"preferences.json":{"content":"{}"}}}`))
		case http.MethodPatch:
			if r.Header.Get("If-Match") != `"revision-1"` {
				t.Errorf("If-Match = %q", r.Header.Get("If-Match"))
			}
			w.WriteHeader(http.StatusPreconditionFailed)
		}
	}))

	storage, _ := NewGistStorage("gist", "token")
	prefs, err := storage.Load()
	if err != nil {
		t.Fatal(err)
	}
	err = storage.Save(prefs)
	if !errors.Is(err, ErrGistConflict) {
		t.Fatalf("Save() error = %v, want ErrGistConflict", err)
	}
}

func TestReadLimitedRejectsOversizedPreferences(t *testing.T) {
	_, err := readLimited(strings.NewReader(strings.Repeat("x", maxGistBytes+1)))
	if err == nil {
		t.Fatal("readLimited() expected an error")
	}
}

func TestValidateRawURLRejectsUntrustedHost(t *testing.T) {
	if err := validateRawURL("https://example.com/preferences.json"); err == nil {
		t.Fatal("validateRawURL() accepted an untrusted host")
	}
}

func TestGistStorageSaveRejectsNil(t *testing.T) {
	storage, _ := NewGistStorage("gist", "token")
	if err := storage.Save(nil); err == nil {
		t.Fatal("Save(nil) expected an error")
	}
}
