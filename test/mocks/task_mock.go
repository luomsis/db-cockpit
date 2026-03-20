package mocks

import (
	"context"

	"github.com/db-cockpit/pkg/common/task"
	taskengine "github.com/db-cockpit/pkg/task"
)

// MockTaskHandler is a mock implementation of TaskHandler
type MockTaskHandler struct {
	HandleFunc   func(ctx context.Context, t *task.Task) (*task.TaskResult, error)
	TaskTypeFunc func() task.TaskType
}

func (m *MockTaskHandler) Handle(ctx context.Context, t *task.Task) (*task.TaskResult, error) {
	if m.HandleFunc != nil {
		return m.HandleFunc(ctx, t)
	}
	return &task.TaskResult{Data: []byte("mock result")}, nil
}

func (m *MockTaskHandler) TaskType() task.TaskType {
	if m.TaskTypeFunc != nil {
		return m.TaskTypeFunc()
	}
	return task.TaskTypeSQLAnalysis
}

// Ensure MockTaskHandler implements taskengine.TaskHandler
var _ taskengine.TaskHandler = (*MockTaskHandler)(nil)