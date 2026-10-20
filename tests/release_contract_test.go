package tests

import (
	"os"
	"slices"
	"strings"
	"testing"

	"go.yaml.in/yaml/v3"
)

func TestReleaseBindsTriggeringTag(t *testing.T) {
	var action struct {
		Runs struct {
			Steps []struct {
				Uses string            `yaml:"uses"`
				Env  map[string]string `yaml:"env"`
			} `yaml:"steps"`
		} `yaml:"runs"`
	}
	decodeYAML(t, "../action.yml", &action)
	for _, step := range action.Runs.Steps {
		if !strings.HasPrefix(step.Uses, "goreleaser/goreleaser-action@") {
			continue
		}
		want := "${{ github.ref_type == 'tag' && github.ref_name || env.GORELEASER_CURRENT_TAG }}"
		if got := step.Env["GORELEASER_CURRENT_TAG"]; got != want {
			t.Fatalf("release tag binding = %q; want the event tag on tag runs and the caller override otherwise", got)
		}
		return
	}
	t.Fatal("the action has no upstream GoReleaser step")
}

func TestGoCacheCanBeOwnedByCaller(t *testing.T) {
	var action struct {
		Inputs map[string]struct {
			Default string `yaml:"default"`
		} `yaml:"inputs"`
		Runs struct {
			Steps []struct {
				Uses string            `yaml:"uses"`
				With map[string]string `yaml:"with"`
			} `yaml:"steps"`
		} `yaml:"runs"`
	}
	decodeYAML(t, "../action.yml", &action)
	if action.Inputs["go-cache"].Default != "true" {
		t.Error("standalone callers must retain Go module/build caching by default")
	}
	for _, step := range action.Runs.Steps {
		if strings.HasPrefix(step.Uses, "actions/setup-go@") {
			if step.With["cache"] != "${{ inputs.go-cache }}" {
				t.Error("callers must be able to disable a duplicate cache restore")
			}
			return
		}
	}
	t.Fatal("the action has no setup-go step")
}

func TestBehaviorTestsRunFresh(t *testing.T) {
	var workflow struct {
		Jobs map[string]struct {
			Steps []struct {
				Run string `yaml:"run"`
			} `yaml:"steps"`
		} `yaml:"jobs"`
	}
	decodeYAML(t, "../.github/workflows/test.yml", &workflow)
	commands := 0
	for name, job := range workflow.Jobs {
		for _, step := range job.Steps {
			args := strings.Fields(step.Run)
			if len(args) < 2 || args[0] != "go" || args[1] != "test" {
				continue
			}
			commands++
			if !slices.Contains(args[2:], "-count=1") {
				t.Errorf("%s: shell behavior tests must bypass Go's result cache: %q", name, step.Run)
			}
		}
	}
	if commands == 0 {
		t.Fatal("the workflow has no Go behavior-test command")
	}
}

func decodeYAML(t *testing.T, path string, target any) {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := yaml.Unmarshal(raw, target); err != nil {
		t.Fatalf("%s: %v", path, err)
	}
}
