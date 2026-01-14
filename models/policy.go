package models

type PolicyAction string

const (
	PolicyActionCreate PolicyAction = "create"
	PolicyActionUpdate PolicyAction = "update"
)

type PolicyWorkflow struct {
	Name      string `json:"name"`
	MatchFile string `json:"match_file"`
	Source    string `json:"source"`
}

type PolicyViolation struct {
	Repository     Repository
	Policy         PolicyWorkflow
	Action         PolicyAction
	TargetPath     string // e.g., ".github/workflows/dockerfile.yml"
	ExpectedSource string // URL to fetch expected content
	CurrentContent string // Current content (empty for create)
}
