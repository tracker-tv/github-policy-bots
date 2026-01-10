package policy

import (
	"encoding/json"

	"github.com/tracker-tv/github-policy-bots/models"
)

func FromJSON(data []byte) ([]models.Workflow, error) {
	var workflows []models.Workflow
	if err := json.Unmarshal(data, &workflows); err != nil {
		return nil, err
	}
	return workflows, nil
}
