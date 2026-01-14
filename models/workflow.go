package models

type PolicyWorkflow struct {
	Name      string `json:"name"`
	MatchFile string `json:"match_file"`
	Source    string `json:"source"`
}

type WorkflowFile struct {
	Name    string
	Path    string
	Content string
}

type Repository struct {
	Name     string
	FullName string
	Private  bool
	Archived bool
}
