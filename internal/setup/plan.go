// Package setup works out, and then applies, the file changes that add
// OpenTelemetry to a project.
//
// BuildPlan only reads files. Apply is the only code in otelstarter that
// writes to a user's project, so a dry run can never change anything.
package setup

import (
	"bytes"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/IvyMurage/otelstarter/internal/adapters/python"
	"github.com/IvyMurage/otelstarter/internal/detect"
	"github.com/IvyMurage/otelstarter/internal/project"
)

// EnvFileName is the settings file init creates in the project folder.
const EnvFileName = "otel.env"

// Change is one file that Apply will create or update.
type Change struct {
	Path    string   // relative to the project folder
	Summary string   // one line, e.g. "add 6 OpenTelemetry packages"
	Preview []string // lines shown to the user before they confirm

	isNew    bool
	original []byte // content when the plan was built, to detect later edits
	content  []byte // content Apply will write
}

// Action is "create" or "update".
func (c Change) Action() string {
	if c.isNew {
		return "create"
	}
	return "update"
}

// Plan is everything init will do, built before anything is written.
type Plan struct {
	Path         string
	Description  string   // e.g. "Python, FastAPI, pip"
	Changes      []Change // files Apply writes
	Commands     []string // commands the user runs afterwards, e.g. to install packages
	StartCommand string
	Notes        []string
}

// NothingToDo reports whether the project is already fully set up.
func (p Plan) NothingToDo() bool {
	return len(p.Changes) == 0 && len(p.Commands) == 0
}

// BuildPlan analyses the project in dir and works out what init would change.
// It never writes.
func BuildPlan(dir string) (Plan, error) {
	path, err := filepath.Abs(dir)
	if err != nil {
		return Plan{}, err
	}
	root, err := os.OpenRoot(path)
	if err != nil {
		return Plan{}, fmt.Errorf("cannot open the project folder: %w. "+
			"Check the path, or run otelstarter init from your project folder", err)
	}
	defer root.Close()

	return buildPlan(path, root.FS())
}

func buildPlan(path string, fsys fs.FS) (Plan, error) {
	proj, err := detect.DetectFS(path, fsys)
	if err != nil {
		return Plan{}, err
	}
	lang, err := pythonLanguage(proj)
	if err != nil {
		return Plan{}, err
	}

	plan := Plan{
		Path:         path,
		Description:  describe(lang),
		StartCommand: "set -a && . ./" + EnvFileName + " && set +a && " + lang.Activation,
		Notes:        lang.Notes,
	}
	for _, other := range proj.Languages {
		if other.Name != lang.Name {
			plan.Notes = append(plan.Notes, fmt.Sprintf(
				"%s was also found but is not set up: init supports Python only for now.", other.Name))
		}
	}

	if err := planEnvFile(&plan, fsys, lang); err != nil {
		return Plan{}, err
	}
	if err := planPackages(&plan, fsys, lang); err != nil {
		return Plan{}, err
	}
	return plan, nil
}

func pythonLanguage(proj project.Project) (project.Language, error) {
	if len(proj.Languages) == 0 {
		return project.Language{}, detect.NoProjectError(proj.Path)
	}
	var found []string
	for _, l := range proj.Languages {
		if l.Name == "Python" {
			return l, nil
		}
		found = append(found, l.Name)
	}
	return project.Language{}, fmt.Errorf("otelstarter init supports Python projects only for now, "+
		"and this project uses %s. Support for more languages is planned", strings.Join(found, " and "))
}

// planEnvFile creates otel.env when it is missing. An existing file is never
// overwritten, because the user may have edited it.
func planEnvFile(plan *Plan, fsys fs.FS, lang project.Language) error {
	content := renderEnvFile(serviceName(lang.AppName, filepath.Base(plan.Path)), python.EnvVars)

	existing, exists, err := readFile(fsys, EnvFileName)
	if err != nil {
		return err
	}
	switch {
	case !exists:
		plan.Changes = append(plan.Changes, Change{
			Path:    EnvFileName,
			Summary: "OpenTelemetry settings for this service",
			Preview: strings.Split(strings.TrimSuffix(string(content), "\n"), "\n"),
			isNew:   true,
			content: content,
		})
	case !bytes.Equal(existing, content):
		plan.Notes = append(plan.Notes, EnvFileName+" already exists and was left unchanged. "+
			"To regenerate it, delete it and run otelstarter init again.")
	}
	return nil
}

// planPackages adds missing packages to requirements.txt for pip projects.
// Other package managers keep dependencies in pyproject.toml or Pipfile,
// which otelstarter does not rewrite (that would lose comments and
// formatting), so the user gets the exact command to run instead.
func planPackages(plan *Plan, fsys fs.FS, lang project.Language) error {
	reqs := python.MissingRequirements(lang)
	if len(reqs) == 0 {
		return nil
	}

	if lang.PackageManager == "pip" {
		existing, _, err := readFile(fsys, "requirements.txt")
		if err != nil {
			return err
		}
		plan.Changes = append(plan.Changes, Change{
			Path:     "requirements.txt",
			Summary:  fmt.Sprintf("add %d OpenTelemetry packages in a block marked \"otelstarter\"", len(reqs)),
			Preview:  reqs,
			original: existing,
			content:  []byte(python.AddToRequirements(string(existing), reqs)),
		})
	}
	plan.Commands = append(plan.Commands, python.InstallCommand(lang.PackageManager, reqs))

	if len(lang.OpenTelemetry) > 0 {
		plan.Notes = append(plan.Notes, "This project already lists OpenTelemetry packages. "+
			"The new ones are pinned to the versions otelstarter tests with; "+
			"if pip reports a version conflict, align the existing ones to match.")
	}
	return nil
}

func describe(lang project.Language) string {
	parts := []string{lang.Name}
	parts = append(parts, lang.Frameworks...)
	if lang.PackageManager != "" {
		parts = append(parts, lang.PackageManager)
	}
	return strings.Join(parts, ", ")
}

func readFile(fsys fs.FS, name string) ([]byte, bool, error) {
	data, err := fs.ReadFile(fsys, name)
	if errors.Is(err, fs.ErrNotExist) {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, fmt.Errorf("cannot read %s: %w", name, err)
	}
	return data, true, nil
}
