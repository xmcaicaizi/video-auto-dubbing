package worker

import (
	"context"
	"sort"

	"vedio/worker/internal/models"

	"github.com/google/uuid"
)

// StepProcessor processes a specific task step.
type StepProcessor interface {
	Name() string
	Process(ctx context.Context, taskID uuid.UUID, msg models.TaskMessage) error
}

// ProcessorRegistry holds all registered step processors.
type ProcessorRegistry struct {
	processors map[string]StepProcessor
}

// NewProcessorRegistry creates a new registry.
func NewProcessorRegistry() *ProcessorRegistry {
	return &ProcessorRegistry{processors: make(map[string]StepProcessor)}
}

// Register adds a processor to the registry.
func (r *ProcessorRegistry) Register(p StepProcessor) {
	if p == nil {
		return
	}
	r.processors[p.Name()] = p
}

// Get retrieves a processor by name.
func (r *ProcessorRegistry) Get(name string) (StepProcessor, bool) {
	p, ok := r.processors[name]
	return p, ok
}

// Names returns registered step names sorted alphabetically.
func (r *ProcessorRegistry) Names() []string {
	names := make([]string, 0, len(r.processors))
	for name := range r.processors {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}
