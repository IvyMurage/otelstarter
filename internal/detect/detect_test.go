package detect

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"testing/fstest"
)

func TestDetectLanguages(t *testing.T) {
	tests := []struct {
		name      string
		files     []string
		wantNames []string
	}{
		{"empty folder", nil, nil},
		{"python", []string{"pyproject.toml"}, []string{"Python"}},
		{"node", []string{"package.json"}, []string{"Node.js"}},
		{"go", []string{"go.mod"}, []string{"Go"}},
		{"python api with node tooling", []string{"requirements.txt", "package.json"}, []string{"Python", "Node.js"}},
		{"unrelated files only", []string{"README.md", "Makefile"}, nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fsys := fstest.MapFS{}
			for _, name := range tt.files {
				fsys[name] = &fstest.MapFile{}
			}

			p, err := DetectFS("/app", fsys)
			if err != nil {
				t.Fatalf("DetectFS() error = %v", err)
			}

			var got []string
			for _, l := range p.Languages {
				got = append(got, l.Name)
			}
			if !reflect.DeepEqual(got, tt.wantNames) {
				t.Errorf("languages = %v, want %v", got, tt.wantNames)
			}
		})
	}
}

func TestDetectMarksUnsupportedDependencyDetection(t *testing.T) {
	p, err := DetectFS("/app", fstest.MapFS{"go.mod": &fstest.MapFile{}, "setup.py": &fstest.MapFile{}})
	if err != nil {
		t.Fatalf("DetectFS() error = %v", err)
	}
	for _, l := range p.Languages {
		want := l.Name == "Python"
		if l.DependenciesDetected != want {
			t.Errorf("%s DependenciesDetected = %v, want %v", l.Name, l.DependenciesDetected, want)
		}
	}
}

func TestDetectIgnoresDirectoriesNamedLikeManifests(t *testing.T) {
	fsys := fstest.MapFS{"go.mod/readme": &fstest.MapFile{}} // makes "go.mod" a folder
	p, err := DetectFS("/app", fsys)
	if err != nil {
		t.Fatalf("DetectFS() error = %v", err)
	}
	if len(p.Languages) != 0 {
		t.Errorf("languages = %v, want none", p.Languages)
	}
}

// A project could contain a symlink named requirements.txt that points at a
// file elsewhere on disk. Detect must refuse to follow it.
func TestDetectRefusesSymlinkOutsideProject(t *testing.T) {
	outside := filepath.Join(t.TempDir(), "secret.txt")
	if err := os.WriteFile(outside, []byte("fastapi\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	if err := os.Symlink(outside, filepath.Join(dir, "requirements.txt")); err != nil {
		t.Skipf("cannot create symlink: %v", err)
	}

	_, err := Detect(dir)
	if err == nil || !strings.Contains(err.Error(), "requirements.txt") {
		t.Errorf("Detect() error = %v, want an error naming requirements.txt", err)
	}
}
