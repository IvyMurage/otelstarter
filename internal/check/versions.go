package check

import (
	"fmt"
	"slices"
	"strings"

	"github.com/IvyMurage/otelstarter/internal/adapters/python"
	"github.com/IvyMurage/otelstarter/internal/project"
)

// checkVersions reports OpenTelemetry packages pinned to versions that do
// not belong together.
//
// OpenTelemetry's Python packages come in two version families: beta
// packages (distro, instrumentation-*) numbered 0.NNbN, and stable ones
// (API, SDK, exporters) numbered 1.NN.N. Packages in a family require each
// other at exactly the same version, so every pin in a family must match.
// Grouping by version shape, not package name, keeps working as new
// packages appear.
func checkVersions(lang project.Language) []Result {
	if len(lang.Pins) == 0 {
		return nil // nothing pinned, nothing to compare
	}
	testedBeta, testedStable := python.TestedVersions()

	families := []struct {
		name   string
		tested string
		pins   []project.Pin
	}{
		{name: "beta", tested: testedBeta},
		{name: "stable", tested: testedStable},
	}
	for _, pin := range lang.Pins {
		i := 1
		if strings.HasPrefix(pin.Version, "0.") {
			i = 0
		}
		families[i].pins = append(families[i].pins, pin)
	}

	var results []Result
	var versions []string
	for _, f := range families {
		distinct := distinctVersions(f.pins)
		versions = append(versions, distinct...)
		if len(distinct) <= 1 {
			continue
		}
		results = append(results, fail(
			fmt.Sprintf("OpenTelemetry %s packages are pinned to different versions: %s", f.name, byVersion(f.pins, distinct)),
			"These packages require each other at exactly the same version, so pip either refuses to install them "+
				"or the app fails when OpenTelemetry starts.",
			fmt.Sprintf("Pin all of them to one version, for example %s, the version otelstarter tests with.", f.tested)))
	}
	if len(results) == 0 {
		results = append(results, pass("OpenTelemetry versions match: "+strings.Join(versions, ", ")))
	}
	return results
}

func distinctVersions(pins []project.Pin) []string {
	var versions []string
	for _, p := range pins {
		if !slices.Contains(versions, p.Version) {
			versions = append(versions, p.Version)
		}
	}
	slices.Sort(versions)
	return versions
}

// byVersion summarises pins per version, naming the packages when there are
// one or two so the odd one out stands out: "0.60b0 (opentelemetry-
// instrumentation-httpx), 0.66b0 (4 packages)".
func byVersion(pins []project.Pin, versions []string) string {
	var parts []string
	for _, v := range versions {
		var pkgs []string
		for _, p := range pins {
			if p.Version == v {
				pkgs = append(pkgs, p.Package)
			}
		}
		who := strings.Join(pkgs, ", ")
		if len(pkgs) > 2 {
			who = fmt.Sprintf("%d packages", len(pkgs))
		}
		parts = append(parts, fmt.Sprintf("%s (%s)", v, who))
	}
	return strings.Join(parts, ", ")
}
