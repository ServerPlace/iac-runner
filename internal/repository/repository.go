package repository

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"
)

type ExecutionPlan struct {
	Timestamp       time.Time `json:"timestamp"`
	AffectedModules []string  `json:"affected_modules"`
}

type Repository interface {
	SavePlan(plan ExecutionPlan) error
}

type JSONFileRepository struct {
	OutputDir string
}

func NewJSONRepository(outputDir string) *JSONFileRepository {
	return &JSONFileRepository{OutputDir: outputDir}
}

func (r *JSONFileRepository) SavePlan(plan ExecutionPlan) error {
	if err := os.MkdirAll(r.OutputDir, 0755); err != nil {
		return err
	}

	fileName := filepath.Join(r.OutputDir, "affected_plan.json")
	file, err := os.Create(fileName)
	if err != nil {
		return err
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	return encoder.Encode(plan)
}
