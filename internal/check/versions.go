package check

import (
	"fmt"
	"slices"
	"strconv"
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
		if pin.Package == "otelstarter" {
			continue // the SDK has its own versions; see checkSDKVersion
		}
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

// checkSDKVersion warns when the otelstarter SDK is pinned to a version older
// than the one this otelstarter pins. init adds missing packages but does not
// change existing pins, so an older pin stays until someone updates it.
func checkSDKVersion(lang project.Language) []Result {
	tested := python.TestedSDKVersion()
	for _, pin := range lang.Pins {
		if pin.Package != "otelstarter" || !olderVersion(pin.Version, tested) {
			continue
		}
		return []Result{warn(
			fmt.Sprintf("otelstarter==%s is older than %s, the version this otelstarter pins", pin.Version, tested),
			"Newer SDK versions fix problems and add features, such as printing events to the terminal during otelstarter run.",
			fmt.Sprintf("Change the pin to otelstarter==%s and reinstall your requirements.", tested))}
	}
	return nil
}

// olderVersion reports whether version a is older than b. Both are plain
// release versions such as 0.1.0; anything else is never reported as older.
func olderVersion(a, b string) bool {
	pa, okA := parseRelease(a)
	pb, okB := parseRelease(b)
	if !okA || !okB {
		return false
	}
	return slices.Compare(pa, pb) < 0
}

func parseRelease(v string) ([]int, bool) {
	var parts []int
	for _, field := range strings.Split(v, ".") {
		n, err := strconv.Atoi(field)
		if err != nil {
			return nil, false
		}
		parts = append(parts, n)
	}
	return parts, len(parts) > 0
}
