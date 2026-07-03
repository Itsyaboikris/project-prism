package models

import "encoding/json"

type Branch struct {
	ID           string          `json:"id"`
	ExperimentID string          `json:"experiment_id"`
	Key          string          `json:"key"`
	Name         string          `json:"name"`
	Weight       float64         `json:"weight"`
	MetadataJSON json.RawMessage `json:"metadata_json"`
}
