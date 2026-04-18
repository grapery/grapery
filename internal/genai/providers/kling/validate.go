package kling

import (
	"fmt"
	"strings"
)

// ValidateCreateTask ensures the async create response includes a task id.
func ValidateCreateTask(operation string, d *CreateTaskData) error {
	if d == nil || strings.TrimSpace(d.TaskID) == "" {
		return fmt.Errorf("kling %s: empty task_id in API response", operation)
	}
	return nil
}
