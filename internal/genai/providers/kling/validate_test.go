package kling

import "testing"

func TestValidateCreateTask(t *testing.T) {
	if err := ValidateCreateTask("x", &CreateTaskData{TaskID: "a"}); err != nil {
		t.Fatal(err)
	}
	if err := ValidateCreateTask("x", &CreateTaskData{TaskID: ""}); err == nil {
		t.Fatal("expected error")
	}
	if err := ValidateCreateTask("x", nil); err == nil {
		t.Fatal("expected error")
	}
}
