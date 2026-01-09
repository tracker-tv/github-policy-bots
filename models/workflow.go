package models

type Workflow struct {
	Name      string `json:"name"`
	MatchFile string `json:"match_file"`
	Source    string `json:"source"`
}
