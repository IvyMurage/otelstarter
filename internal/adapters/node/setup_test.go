package node

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/IvyMurage/otelstarter/internal/project"
)

func TestEnvVars(t *testing.T) {
	vars := EnvVars(project.Language{Enabled: []string{"http", "undici", "express"}})

	got := map[string]string{}
	for _, v := range vars {
		got[v.Name] = v.Value
	}
	want := map[string]string{"OTEL_NODE_ENABLED_INSTRUMENTATIONS": "http,undici,express"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("EnvVars() = %v, want %v", got, want)
	}
}

func TestMissingPackagesAndInstallCommand(t *testing.T) {
	lang := project.Language{Instrumentation: []project.Instrumentation{
		{Package: sdkPackage},
		{Package: "@opentelemetry/instrumentation-express", Installed: true},
		{Package: "@opentelemetry/instrumentation-pg"},
	}}
	pkgs := MissingPackages(lang)
	want := []string{"otelstarter@" + SDKVersion, "@opentelemetry/instrumentation-pg@0.75.0"}
	if !reflect.DeepEqual(pkgs, want) {
		t.Fatalf("MissingPackages() = %v, want %v", pkgs, want)
	}

	list := strings.Join(want, " ")
	tests := map[string]string{
		"npm":  "npm install --save-exact " + list,
		"yarn": "yarn add --exact " + list,
		"pnpm": "pnpm add --save-exact " + list,
		"bun":  "bun add --exact " + list,
	}
	for manager, want := range tests {
		if got := InstallCommand(manager, pkgs); got != want {
			t.Errorf("InstallCommand(%q) = %q, want %q", manager, got, want)
		}
	}
}

func TestLaunchesNode(t *testing.T) {
	tests := map[string]bool{
		"node app.js": true, "npm start": true, "/usr/bin/node x": true, "nest start": true,
		"uvicorn main:app": false, "python app.py": false,
	}
	for cmd, want := range tests {
		if got := LaunchesNode(strings.Fields(cmd)); got != want {
			t.Errorf("LaunchesNode(%q) = %v, want %v", cmd, got, want)
		}
	}
}

func TestWithRegister(t *testing.T) {
	flag := "--require " + RegisterModule
	legacy := "--require " + legacyRegisterModule
	tests := map[string]string{
		"":                          flag,
		"--max-old-space-size=4096": "--max-old-space-size=4096 " + flag, // existing options are kept
		flag:                        flag,                                // not added twice
		legacy:                      legacy,                              // never two loaders
	}
	for in, want := range tests {
		if got := WithRegister(in, RegisterModule); got != want {
			t.Errorf("WithRegister(%q) = %q, want %q", in, got, want)
		}
	}
}

// fakeInstall puts an empty package.json for each package in dir's node_modules.
func fakeInstall(t *testing.T, dir string, pkgs ...string) {
	t.Helper()
	for _, pkg := range pkgs {
		pkgDir := filepath.Join(dir, "node_modules", filepath.FromSlash(pkg))
		if err := os.MkdirAll(pkgDir, 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(pkgDir, "package.json"), []byte("{}"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
}

func TestLoader(t *testing.T) {
	tests := []struct {
		name      string
		installed []string
		want      string
	}{
		{"nothing installed", nil, ""},
		{"SDK", []string{sdkPackage}, RegisterModule},
		{"bundle from an earlier otelstarter", []string{legacyPackage}, legacyRegisterModule},
		{"both: the SDK wins", []string{legacyPackage, sdkPackage}, RegisterModule},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			fakeInstall(t, dir, tt.installed...)
			got, ok := Loader(dir)
			if got != tt.want || ok != (tt.want != "") {
				t.Errorf("Loader() = %q, %v, want %q", got, ok, tt.want)
			}
		})
	}
}

func TestNotInstalled(t *testing.T) {
	dir := t.TempDir()
	fakeInstall(t, dir, sdkPackage)
	lang := project.Language{Instrumentation: []project.Instrumentation{
		{Package: sdkPackage, Installed: true},
		{Package: "@opentelemetry/instrumentation-pg", Installed: true}, // declared, npm install not run
		{Package: "@opentelemetry/instrumentation-redis"},               // not declared: init's job, not this check's
	}}
	if got := NotInstalled(dir, lang); !reflect.DeepEqual(got, []string{"@opentelemetry/instrumentation-pg"}) {
		t.Errorf("NotInstalled() = %v", got)
	}
}

// init pins the SDK version that is in sdk/node, so a release cannot ship
// a CLI that installs a different SDK than the one it was tested with.
func TestSDKVersionMatchesPackageJSON(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("..", "..", "..", "sdk", "node", "package.json"))
	if err != nil {
		t.Fatal(err)
	}
	var pkg struct{ Version string }
	if err := json.Unmarshal(data, &pkg); err != nil {
		t.Fatal(err)
	}
	if pkg.Version != SDKVersion {
		t.Errorf("SDKVersion = %q, but sdk/node/package.json is %q", SDKVersion, pkg.Version)
	}
}
