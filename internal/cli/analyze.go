package cli

import (
	"fmt"
	"io"
	"strings"

	"github.com/IvyMurage/otelstarter/internal/detect"
	"github.com/IvyMurage/otelstarter/internal/project"
)

func analyze(args []string, stdout io.Writer) error {
	dir := "."
	if len(args) == 1 {
		dir = args[0]
	}

	p, err := detect.Detect(dir)
	if err != nil {
		return err
	}
	if len(p.Languages) == 0 {
		return fmt.Errorf("no supported project found in %s.\n\n"+
			"Looked for: %s\n"+
			"Run otelstarter analyze from your project's root folder, or pass its path: otelstarter analyze ./my-app",
			p.Path, strings.Join(detect.AllManifests(), ", "))
	}

	printProject(stdout, p)
	return nil
}

func printProject(w io.Writer, p project.Project) {
	fmt.Fprintf(w, "Project: %s\n", p.Path)
	for _, lang := range p.Languages {
		fmt.Fprintln(w)
		printField(w, "Language", lang.Name)
		printField(w, "Found", strings.Join(lang.Manifests, ", "))

		if !lang.DependenciesDetected {
			printField(w, "Dependencies", "detection for "+lang.Name+" is not supported yet")
			continue
		}

		printField(w, "Package manager", orDefault(lang.PackageManager, "unknown"))
		printField(w, "Framework", orDefault(strings.Join(lang.Frameworks, ", "), "none detected"))

		if len(lang.Dependencies) == 0 {
			printField(w, "Dependencies", "none of the supported dependencies found")
		} else {
			fmt.Fprintln(w, "  Dependencies:")
			for _, d := range lang.Dependencies {
				fmt.Fprintf(w, "    %-16s %s\n", d.Category, d.Package)
			}
		}

		printField(w, "OpenTelemetry", orDefault(strings.Join(lang.OpenTelemetry, ", "), "not installed"))

		if len(lang.Instrumentation) > 0 {
			fmt.Fprintln(w)
			fmt.Fprintln(w, "  Recommended instrumentation (✓ installed, + to add):")
			for _, inst := range lang.Instrumentation {
				mark := "+"
				if inst.Installed {
					mark = "✓"
				}
				fmt.Fprintf(w, "    %s %-45s %s\n", mark, inst.Package, inst.Purpose)
			}
		}
		if lang.Activation != "" {
			fmt.Fprintln(w)
			printField(w, "Start with", lang.Activation)
			fmt.Fprintf(w, "  %-18s%s\n", "", "(replace the example with your own start command)")
		}

		if len(lang.Notes) > 0 {
			fmt.Fprintln(w)
		}
		for _, note := range lang.Notes {
			fmt.Fprintf(w, "  Note: %s\n", note)
		}
	}
}

func printField(w io.Writer, label, value string) {
	fmt.Fprintf(w, "  %-18s%s\n", label+":", value)
}

func orDefault(value, fallback string) string {
	if value == "" {
		return fallback
	}
	return value
}
