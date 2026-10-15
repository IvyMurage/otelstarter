package setup

import (
	"strings"
	"testing"
	"testing/fstest"
)

func files(m map[string]string) fstest.MapFS {
	fsys := fstest.MapFS{}
	for name, content := range m {
		fsys[name] = &fstest.MapFile{Data: []byte(content)}
	}
	return fsys
}

func changePaths(p Plan) []string {
	var paths []string
	for _, c := range p.Changes {
		paths = append(paths, c.Action()+" "+c.Path)
	}
	return paths
}

func TestBuildPlan(t *testing.T) {
	tests := []struct {
		name         string
		files        map[string]string
		wantChanges  []string
		wantCommand  string // prefix of the single install command; "" for none
		wantNote     string
		nothingToDo  bool
		wantStartCmd string
	}{
		{
			name:         "pip project gets otel.env and a requirements block",
			files:        map[string]string{"requirements.txt": "fastapi\n"},
			wantChanges:  []string{"create otel.env", "update requirements.txt"},
			wantCommand:  "pip install -r requirements.txt",
			wantStartCmd: "otelstarter run -- uvicorn main:app",
		},
		{
			name:        "uv project gets otel.env and a uv add command, pyproject untouched",
			files:       map[string]string{"pyproject.toml": "[project]\ndependencies = [\"flask\"]\n", "uv.lock": ""},
			wantChanges: []string{"create otel.env"},
			wantCommand: "uv add opentelemetry-distro==",
		},
		{
			name: "edited otel.env is left alone",
			files: map[string]string{
				"requirements.txt": "flask\n",
				"otel.env":         "OTEL_SERVICE_NAME=custom\n",
			},
			wantChanges: []string{"update requirements.txt"},
			wantCommand: "pip install",
			wantNote:    "otel.env already exists and was left unchanged",
		},
		{
			name:        "other languages are reported but not set up",
			files:       map[string]string{"requirements.txt": "flask\n", "package.json": "{}"},
			wantChanges: []string{"create otel.env", "update requirements.txt"},
			wantCommand: "pip install",
			wantNote:    "Node.js was also found but is not set up",
		},
		{
			name: "existing OpenTelemetry packages trigger a version note",
			files: map[string]string{
				"requirements.txt": "flask\nopentelemetry-sdk==1.20.0\n",
			},
			wantChanges: []string{"create otel.env", "update requirements.txt"},
			wantCommand: "pip install",
			wantNote:    "already lists OpenTelemetry packages",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			plan, err := buildPlan("/home/dev/shop", files(tt.files))
			if err != nil {
				t.Fatalf("buildPlan() error = %v", err)
			}

			if got := strings.Join(changePaths(plan), ", "); got != strings.Join(tt.wantChanges, ", ") {
				t.Errorf("changes = %q, want %q", got, strings.Join(tt.wantChanges, ", "))
			}
			if tt.wantCommand != "" && (len(plan.Commands) != 1 || !strings.HasPrefix(plan.Commands[0], tt.wantCommand)) {
				t.Errorf("commands = %q, want one starting with %q", plan.Commands, tt.wantCommand)
			}
			if tt.wantNote != "" && !strings.Contains(strings.Join(plan.Notes, "\n"), tt.wantNote) {
				t.Errorf("notes = %q, want one containing %q", plan.Notes, tt.wantNote)
			}
			if tt.wantStartCmd != "" && plan.StartCommand != tt.wantStartCmd {
				t.Errorf("StartCommand = %q, want %q", plan.StartCommand, tt.wantStartCmd)
			}
		})
	}
}

func TestBuildPlanUsesAppNameForService(t *testing.T) {
	plan, err := buildPlan("/home/dev/folder", files(map[string]string{
		"pyproject.toml": "[project]\nname = \"Shop API\"\n",
	}))
	if err != nil {
		t.Fatalf("buildPlan() error = %v", err)
	}
	if !strings.Contains(string(plan.Changes[0].content), "OTEL_SERVICE_NAME=shop-api\n") {
		t.Errorf("otel.env = %q, want OTEL_SERVICE_NAME=shop-api", plan.Changes[0].content)
	}
}

func TestBuildPlanErrors(t *testing.T) {
	tests := []struct {
		name    string
		files   map[string]string
		wantErr string
	}{
		{"no project", nil, "no supported project found"},
		{"only unsupported languages", map[string]string{"go.mod": "module x\n"}, "supports Python and Node.js projects for now, and this project uses Go"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := buildPlan("/app", files(tt.files))
			if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
				t.Errorf("buildPlan() error = %v, want it to contain %q", err, tt.wantErr)
			}
		})
	}
}

func TestBuildPlanNodeProject(t *testing.T) {
	plan, err := buildPlan("/home/dev/shop", files(map[string]string{
		"package.json": `{"name": "shop", "scripts": {"start": "node server.js"},
			"dependencies": {"express": "^4", "pg": "^8", "pino": "^9"}}`,
		"yarn.lock": "",
	}))
	if err != nil {
		t.Fatalf("buildPlan() error = %v", err)
	}

	if got := strings.Join(changePaths(plan), ", "); got != "create otel.env" {
		t.Errorf("changes = %q, want only otel.env: package.json is never edited", got)
	}
	env := string(plan.Changes[0].content)
	for _, want := range []string{
		"OTEL_SERVICE_NAME=shop\n",
		"OTEL_NODE_ENABLED_INSTRUMENTATIONS=http,undici,runtime-node,express,pg,pino\n",
	} {
		if !strings.Contains(env, want) {
			t.Errorf("otel.env is missing %q", want)
		}
	}
	// pino needs no package: the SDK includes its instrumentation.
	wantCmd := "yarn add --exact otelstarter@0.1.1 @opentelemetry/instrumentation-express@0.71.0 @opentelemetry/instrumentation-pg@0.75.0"
	if len(plan.Commands) != 1 || plan.Commands[0] != wantCmd {
		t.Errorf("Commands = %q, want [%q]", plan.Commands, wantCmd)
	}
	if plan.StartCommand != "otelstarter run -- yarn start" {
		t.Errorf("StartCommand = %q", plan.StartCommand)
	}
}

// A Python API with a package.json for frontend tooling is still set up as Python.
func TestBuildPlanPrefersPython(t *testing.T) {
	plan, err := buildPlan("/app", files(map[string]string{
		"requirements.txt": "fastapi\n",
		"package.json":     `{"dependencies": {"express": "^4"}}`,
	}))
	if err != nil {
		t.Fatalf("buildPlan() error = %v", err)
	}
	if !strings.HasPrefix(plan.Description, "Python") {
		t.Errorf("Description = %q, want the Python project", plan.Description)
	}
}
