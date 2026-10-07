package node

import (
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
	want := map[string]string{
		"OTEL_NODE_ENABLED_INSTRUMENTATIONS": "http,undici,express",
		"OTEL_NODE_RESOURCE_DETECTORS":       "env,host,os,process",
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("EnvVars() = %v, want %v", got, want)
	}
}

func TestMissingPackagesAndInstallCommand(t *testing.T) {
	lang := project.Language{Instrumentation: []project.Instrumentation{
		{Package: apiPackage},
		{Package: autoPackage, Installed: true},
	}}
	pkgs := MissingPackages(lang)
	if !reflect.DeepEqual(pkgs, []string{"@opentelemetry/api@" + apiVersion}) {
		t.Fatalf("MissingPackages() = %v", pkgs)
	}

	tests := map[string]string{
		"npm":  "npm install --save-exact @opentelemetry/api@" + apiVersion,
		"yarn": "yarn add --exact @opentelemetry/api@" + apiVersion,
		"pnpm": "pnpm add --save-exact @opentelemetry/api@" + apiVersion,
		"bun":  "bun add --exact @opentelemetry/api@" + apiVersion,
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
	tests := map[string]string{
		"":                          flag,
		"--max-old-space-size=4096": "--max-old-space-size=4096 " + flag, // existing options are kept
		flag:                        flag,                                // not added twice
	}
	for in, want := range tests {
		if got := WithRegister(in); got != want {
			t.Errorf("WithRegister(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestInstalled(t *testing.T) {
	dir := t.TempDir()
	if Installed(dir) {
		t.Error("Installed() = true for an empty folder")
	}
	pkgDir := filepath.Join(dir, "node_modules", "@opentelemetry", "auto-instrumentations-node")
	if err := os.MkdirAll(pkgDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(pkgDir, "package.json"), []byte("{}"), 0o644); err != nil {
		t.Fatal(err)
	}
	if !Installed(dir) {
		t.Error("Installed() = false although the package is in node_modules")
	}
}
