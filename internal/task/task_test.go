package task

import (
	"testing"
	"time"
)

func TestNewTask(t *testing.T) {
	tool := "test-tool"
	command := "echo"
	args := []string{"hello", "world"}

	task := NewTask(tool, command, args)

	if task.Tool != tool {
		t.Errorf("Expected tool %s, got %s", tool, task.Tool)
	}

	if task.Command != command {
		t.Errorf("Expected command %s, got %s", command, task.Command)
	}

	if len(task.Args) != len(args) {
		t.Errorf("Expected %d args, got %d", len(args), len(task.Args))
	}

	if task.Status != StatusQueued {
		t.Errorf("Expected status %s, got %s", StatusQueued, task.Status)
	}

	if task.ID == "" {
		t.Error("Expected task ID to be generated")
	}

	if task.CreatedAt.IsZero() {
		t.Error("Expected CreatedAt to be set")
	}
}

func TestTaskAppendOutput(t *testing.T) {
	task := NewTask("test", "echo", []string{})
	
	output1 := "Line 1"
	output2 := "Line 2"
	
	task.AppendOutput(output1)
	task.AppendOutput(output2)
	
	if len(task.Output) != 2 {
		t.Errorf("Expected 2 output lines, got %d", len(task.Output))
	}
	
	if task.Output[0] != output1 {
		t.Errorf("Expected first output %s, got %s", output1, task.Output[0])
	}
	
	if task.Output[1] != output2 {
		t.Errorf("Expected second output %s, got %s", output2, task.Output[1])
	}
}

func TestTaskSetStatus(t *testing.T) {
	task := NewTask("test", "echo", []string{})
	
	// Test setting to running
	task.SetStatus(StatusRunning)
	if task.Status != StatusRunning {
		t.Errorf("Expected status %s, got %s", StatusRunning, task.Status)
	}
	if task.StartedAt.IsZero() {
		t.Error("Expected StartedAt to be set when status is running")
	}
	
	// Test setting to complete
	task.SetStatus(StatusComplete)
	if task.Status != StatusComplete {
		t.Errorf("Expected status %s, got %s", StatusComplete, task.Status)
	}
	if task.EndedAt.IsZero() {
		t.Error("Expected EndedAt to be set when status is complete")
	}
}

func TestTaskSetError(t *testing.T) {
	task := NewTask("test", "echo", []string{})
	
	errorMsg := "Test error message"
	task.SetError(errorMsg)
	
	if task.Error != errorMsg {
		t.Errorf("Expected error %s, got %s", errorMsg, task.Error)
	}
}

func TestTaskGetStatus(t *testing.T) {
	task := NewTask("test", "echo", []string{})
	
	status := task.GetStatus()
	if status != StatusQueued {
		t.Errorf("Expected status %s, got %s", StatusQueued, status)
	}
	
	task.SetStatus(StatusRunning)
	status = task.GetStatus()
	if status != StatusRunning {
		t.Errorf("Expected status %s, got %s", StatusRunning, status)
	}
}

func TestTaskClone(t *testing.T) {
	task := NewTask("test", "echo", []string{"arg1", "arg2"})
	task.SetStatus(StatusRunning)
	task.AppendOutput("output line")
	task.SetError("test error")
	
	clone := task.Clone()
	
	// Verify all fields are copied
	if clone.ID != task.ID {
		t.Error("Clone ID doesn't match")
	}
	if clone.Tool != task.Tool {
		t.Error("Clone Tool doesn't match")
	}
	if clone.Command != task.Command {
		t.Error("Clone Command doesn't match")
	}
	if clone.Status != task.Status {
		t.Error("Clone Status doesn't match")
	}
	if clone.Error != task.Error {
		t.Error("Clone Error doesn't match")
	}
	if len(clone.Args) != len(task.Args) {
		t.Error("Clone Args length doesn't match")
	}
	if len(clone.Output) != len(task.Output) {
		t.Error("Clone Output length doesn't match")
	}
	
	// Verify slices are independent copies
	if len(clone.Args) > 0 {
		clone.Args[0] = "modified"
		if task.Args[0] == "modified" {
			t.Error("Modifying clone Args affected original")
		}
	}
	
	if len(clone.Output) > 0 {
		clone.Output[0] = "modified"
		if task.Output[0] == "modified" {
			t.Error("Modifying clone Output affected original")
		}
	}
}

func TestStatusValues(t *testing.T) {
	// Test that all status constants have expected values
	tests := []struct {
		status   Status
		expected string
	}{
		{StatusQueued, "queued"},
		{StatusRunning, "running"},
		{StatusComplete, "complete"},
		{StatusFailed, "failed"},
		{StatusCanceled, "canceled"},
	}
	
	for _, test := range tests {
		if string(test.status) != test.expected {
			t.Errorf("Status %s has unexpected value: %s", test.expected, test.status)
		}
	}
}

func TestTaskConcurrency(t *testing.T) {
	task := NewTask("test", "echo", []string{})
	
	// Test concurrent access to task methods
	done := make(chan bool, 3)
	
	// Goroutine 1: Append output
	go func() {
		for i := 0; i < 100; i++ {
			task.AppendOutput("output")
		}
		done <- true
	}()
	
	// Goroutine 2: Get status
	go func() {
		for i := 0; i < 100; i++ {
			_ = task.GetStatus()
		}
		done <- true
	}()
	
	// Goroutine 3: Clone task
	go func() {
		for i := 0; i < 100; i++ {
			_ = task.Clone()
		}
		done <- true
	}()
	
	// Wait for all goroutines with timeout
	timeout := time.After(2 * time.Second)
	for i := 0; i < 3; i++ {
		select {
		case <-done:
		case <-timeout:
			t.Fatal("Test timed out - possible deadlock")
		}
	}
}
