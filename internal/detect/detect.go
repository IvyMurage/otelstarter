// Package detect finds which languages a project uses and asks the matching
// language adapter for details.
package detect

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/IvyMurage/otelstarter/internal/adapters/node"
	"github.com/IvyMurage/otelstarter/internal/adapters/python"
	"github.com/IvyMurage/otelstarter/internal/project"
)

type language struct {
	name      string
	manifests []string
	// analyze is nil until the language has an adapter.
	analyze func(fs.FS) (project.Language, error)
}

var languages = []language{
	{"Python", python.Manifests, python.Analyze},
	{"Node.js", node.Manifests, node.Analyze},
	{"Go", []string{"go.mod"}, nil},
}

// Detect analyses the project in dir. Only files directly inside dir are
// read; subfolders are not searched.
func Detect(dir string) (project.Project, error) {
	path, err := filepath.Abs(dir)
	if err != nil {
		return project.Project{}, err
	}

	// os.Root refuses any path, including a symlink, that leads outside
	// the folder, so a hostile project cannot make us read other files.
	root, err := os.OpenRoot(path)
	if err != nil {
		return project.Project{}, fmt.Errorf("cannot open the project folder: %w. "+
			"Check the path, or run otelstarter analyze from your project folder", err)
	}
	defer root.Close()

	return DetectFS(path, root.FS())
}

// DetectFS analyses the project in fsys, which the caller has already
// opened. path is only used for display. Tests pass an in-memory file system.
func DetectFS(path string, fsys fs.FS) (project.Project, error) {
	p := project.Project{Path: path}

	for _, l := range languages {
		found, err := existingFiles(fsys, l.manifests)
		if err != nil {
			return project.Project{}, err
		}
		if len(found) == 0 {
			continue
		}

		lang := project.Language{Name: l.name}
		if l.analyze != nil {
			lang, err = l.analyze(fsys)
			if err != nil {
				return project.Project{}, err
			}
		}
		lang.Manifests = found
		p.Languages = append(p.Languages, lang)
	}
	return p, nil
}

// AllManifests lists every file Detect looks for, for error messages.
func AllManifests() []string {
	var names []string
	for _, l := range languages {
		names = append(names, l.manifests...)
	}
	return names
}

func existingFiles(fsys fs.FS, names []string) ([]string, error) {
	var found []string
	for _, name := range names {
		info, err := fs.Stat(fsys, name)
		if errors.Is(err, fs.ErrNotExist) {
			continue
		}
		if err != nil {
			return nil, fmt.Errorf("cannot read %s: %w", name, err)
		}
		if info.Mode().IsRegular() {
			found = append(found, name)
		}
	}
	return found, nil
}

// NoProjectError explains that no supported manifest was found in path.
func NoProjectError(path string) error {
	return fmt.Errorf("no supported project found in %s.\n\n"+
		"Looked for: %s\n"+
		"Run otelstarter from your project's root folder, or pass the folder path, for example: otelstarter analyze ./my-app or otelstarter init ./my-app",
		path, strings.Join(AllManifests(), ", "))
}
