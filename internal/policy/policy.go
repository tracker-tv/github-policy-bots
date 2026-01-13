package policy

import (
	"encoding/json"

	"github.com/tracker-tv/github-policy-bots/models"
)

func FromJSON(data []byte) ([]models.PolicyWorkflow, error) {
	var workflows []models.PolicyWorkflow
	if err := json.Unmarshal(data, &workflows); err != nil {
		return nil, err
	}
	return workflows, nil
}
