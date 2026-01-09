package policy

import "testing"

func TestFromJSON(t *testing.T) {
	data := []byte(`[
		{
			"name": "Test Workflow",
			"match_file": "test.yml",
			"source": "https://example.com/workflow"
		},
		{
			"name": "Dockerfile",
			"match_file": "**/Dockerfile*",
			"source": "https://example.com/another-workflow"
		}
	]`)

	workflows, err := FromJSON(data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(workflows) != 2 {
		t.Fatalf("expected 2 workflows, got %d", len(workflows))
	}

	tests := []struct {
		name       string
		wantName   string
		wantMatch  string
		wantSource string
	}{
		{
			wantName:   "Test Workflow",
			wantMatch:  "test.yml",
			wantSource: "https://example.com/workflow",
		},
		{
			wantName:   "Dockerfile",
			wantMatch:  "**/Dockerfile*",
			wantSource: "https://example.com/another-workflow",
		},
	}

	for index, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			wf := workflows[index]

			if wf.Name != tt.wantName {
				t.Errorf("Name: expected %q, got %q", tt.wantName, wf.Name)
			}
			if wf.MatchFile != tt.wantMatch {
				t.Errorf("MatchFiles: expected %q, got %q", tt.wantMatch, wf.MatchFile)
			}
			if wf.Source != tt.wantSource {
				t.Errorf("Source: expected %q, got %q", tt.wantSource, wf.Source)
			}
		})
	}

}
