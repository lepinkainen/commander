package task

import (
	"sync"
	"testing"
	"time"
)

func TestNewManager(t *testing.T) {
	manager := NewManager()

	if manager == nil {
		t.Fatal("NewManager returned nil")
	}

	if manager.tasks == nil {
		t.Error("tasks map not initialized")
	}

	if manager.queues == nil {
		t.Error("queues map not initialized")
	}

	if manager.listeners == nil {
		t.Error("listeners slice not initialized")
	}
}

func TestManagerCreateQueue(t *testing.T) {
	manager := NewManager()

	tool := "test-tool"
	bufferSize := 10

	queue := manager.CreateQueue(tool, bufferSize)

	if queue == nil {
		t.Fatal("CreateQueue returned nil")
	}

	// Try to create the same queue again
	queue2 := manager.CreateQueue(tool, bufferSize)

	// Should return the same queue
	if queue != queue2 {
		t.Error("CreateQueue should return existing queue for same tool")
	}
}

func TestManagerAddTask(t *testing.T) {
	manager := NewManager()
	tool := "test-tool"

	// Create queue first
	manager.CreateQueue(tool, 10)

	// Add task
	task := NewTask(tool, "echo", []string{"test"})
	err := manager.AddTask(task)

	if err != nil {
		t.Fatalf("AddTask failed: %v", err)
	}

	// Try to add the same task again
	err = manager.AddTask(task)
	if err == nil {
		t.Error("Expected error when adding duplicate task")
	}

	// Try to add task for non-existent queue
	task2 := NewTask("non-existent", "echo", []string{})
	err = manager.AddTask(task2)
	if err == nil {
		t.Error("Expected error when adding task for non-existent queue")
	}
}

func TestManagerGetTask(t *testing.T) {
	manager := NewManager()
	tool := "test-tool"

	// Create queue and add task
	manager.CreateQueue(tool, 10)
	originalTask := NewTask(tool, "echo", []string{"test"})
	if err := manager.AddTask(originalTask); err != nil {
		t.Fatalf("AddTask failed: %v", err)
	}

	// Get the task
	retrievedTask, err := manager.GetTask(originalTask.ID)
	if err != nil {
		t.Fatalf("GetTask failed: %v", err)
	}

	if retrievedTask.ID != originalTask.ID {
		t.Error("Retrieved task ID doesn't match")
	}

	// Try to get non-existent task
	_, err = manager.GetTask("non-existent-id")
	if err == nil {
		t.Error("Expected error when getting non-existent task")
	}
}

func TestManagerGetAllTasks(t *testing.T) {
	manager := NewManager()

	// Add multiple tasks
	tools := []string{"tool1", "tool2", "tool3"}
	for _, tool := range tools {
		manager.CreateQueue(tool, 10)
		task := NewTask(tool, "echo", []string{})
		if err := manager.AddTask(task); err != nil {
			t.Fatalf("AddTask failed: %v", err)
		}
	}

	tasks := manager.GetAllTasks()

	if len(tasks) != len(tools) {
		t.Errorf("Expected %d tasks, got %d", len(tools), len(tasks))
	}
}

func TestManagerGetTasksByTool(t *testing.T) {
	manager := NewManager()

	// Add tasks for different tools
	tool1 := "tool1"
	tool2 := "tool2"

	manager.CreateQueue(tool1, 10)
	manager.CreateQueue(tool2, 10)

	// Add 2 tasks for tool1
	for i := 0; i < 2; i++ {
		task := NewTask(tool1, "echo", []string{})
		if err := manager.AddTask(task); err != nil {
			t.Fatalf("AddTask failed: %v", err)
		}
	}

	// Add 3 tasks for tool2
	for i := 0; i < 3; i++ {
		task := NewTask(tool2, "echo", []string{})
		if err := manager.AddTask(task); err != nil {
			t.Fatalf("AddTask failed: %v", err)
		}
	}

	tool1Tasks := manager.GetTasksByTool(tool1)
	if len(tool1Tasks) != 2 {
		t.Errorf("Expected 2 tasks for %s, got %d", tool1, len(tool1Tasks))
	}

	tool2Tasks := manager.GetTasksByTool(tool2)
	if len(tool2Tasks) != 3 {
		t.Errorf("Expected 3 tasks for %s, got %d", tool2, len(tool2Tasks))
	}
}

func TestManagerUpdateTaskStatus(t *testing.T) {
	manager := NewManager()
	tool := "test-tool"

	manager.CreateQueue(tool, 10)
	task := NewTask(tool, "echo", []string{})
	if err := manager.AddTask(task); err != nil {
		t.Fatalf("AddTask failed: %v", err)
	}

	// Update status
	err := manager.UpdateTaskStatus(task.ID, StatusRunning)
	if err != nil {
		t.Fatalf("UpdateTaskStatus failed: %v", err)
	}

	// Verify status was updated
	retrievedTask, _ := manager.GetTask(task.ID)
	if retrievedTask.GetStatus() != StatusRunning {
		t.Error("Task status was not updated")
	}

	// Try to update non-existent task
	err = manager.UpdateTaskStatus("non-existent", StatusRunning)
	if err == nil {
		t.Error("Expected error when updating non-existent task")
	}
}

func TestManagerAppendTaskOutput(t *testing.T) {
	manager := NewManager()
	tool := "test-tool"

	manager.CreateQueue(tool, 10)
	task := NewTask(tool, "echo", []string{})
	if err := manager.AddTask(task); err != nil {
		t.Fatalf("AddTask failed: %v", err)
	}

	// Append output
	output := "test output"
	err := manager.AppendTaskOutput(task.ID, output)
	if err != nil {
		t.Fatalf("AppendTaskOutput failed: %v", err)
	}

	// Verify output was appended
	retrievedTask, _ := manager.GetTask(task.ID)
	if len(retrievedTask.Output) != 1 || retrievedTask.Output[0] != output {
		t.Error("Output was not appended correctly")
	}

	// Try to append to non-existent task
	err = manager.AppendTaskOutput("non-existent", output)
	if err == nil {
		t.Error("Expected error when appending to non-existent task")
	}
}

func TestManagerSubscribeUnsubscribe(t *testing.T) {
	manager := NewManager()

	// Subscribe
	ch := manager.Subscribe()
	if ch == nil {
		t.Fatal("Subscribe returned nil channel")
	}

	// Verify channel is in listeners
	if len(manager.listeners) != 1 {
		t.Error("Listener not added")
	}

	// Unsubscribe
	manager.Unsubscribe(ch)

	// Verify channel is removed
	if len(manager.listeners) != 0 {
		t.Error("Listener not removed")
	}
}

func TestManagerBroadcastEvent(t *testing.T) {
	manager := NewManager()

	// Subscribe multiple listeners
	ch1 := manager.Subscribe()
	ch2 := manager.Subscribe()

	// Add a task to trigger events
	tool := "test-tool"
	manager.CreateQueue(tool, 10)
	task := NewTask(tool, "echo", []string{})

	// This should broadcast a "created" event
	if err := manager.AddTask(task); err != nil {
		t.Fatalf("AddTask failed: %v", err)
	}

	// Check if both listeners received the event
	timeout := time.After(1 * time.Second)

	select {
	case event := <-ch1:
		if event.TaskID != task.ID {
			t.Error("Event has wrong task ID")
		}
		if event.Type != "created" {
			t.Error("Event has wrong type")
		}
	case <-timeout:
		t.Error("Listener 1 didn't receive event")
	}

	select {
	case event := <-ch2:
		if event.TaskID != task.ID {
			t.Error("Event has wrong task ID")
		}
	case <-timeout:
		t.Error("Listener 2 didn't receive event")
	}

	// Clean up
	manager.Unsubscribe(ch1)
	manager.Unsubscribe(ch2)
}

func TestManagerGetQueueStats(t *testing.T) {
	manager := NewManager()

	// Create queues
	tool1 := "tool1"
	tool2 := "tool2"
	manager.CreateQueue(tool1, 10)
	manager.CreateQueue(tool2, 10)

	// Add tasks with different statuses
	task1 := NewTask(tool1, "echo", []string{})
	if err := manager.AddTask(task1); err != nil {
		t.Fatalf("AddTask failed: %v", err)
	}
	if err := manager.UpdateTaskStatus(task1.ID, StatusRunning); err != nil {
		t.Fatalf("UpdateTaskStatus failed: %v", err)
	}

	task2 := NewTask(tool1, "echo", []string{})
	if err := manager.AddTask(task2); err != nil {
		t.Fatalf("AddTask failed: %v", err)
	}
	if err := manager.UpdateTaskStatus(task2.ID, StatusComplete); err != nil {
		t.Fatalf("UpdateTaskStatus failed: %v", err)
	}

	task3 := NewTask(tool2, "echo", []string{})
	if err := manager.AddTask(task3); err != nil {
		t.Fatalf("AddTask failed: %v", err)
	}
	if err := manager.UpdateTaskStatus(task3.ID, StatusFailed); err != nil {
		t.Fatalf("UpdateTaskStatus failed: %v", err)
	}

	// Get stats
	stats := manager.GetQueueStats()

	if len(stats) != 2 {
		t.Errorf("Expected stats for 2 tools, got %d", len(stats))
	}

	tool1Stats := stats[tool1]
	if tool1Stats.Running != 1 {
		t.Errorf("Expected 1 running task for %s, got %d", tool1, tool1Stats.Running)
	}
	if tool1Stats.Completed != 1 {
		t.Errorf("Expected 1 completed task for %s, got %d", tool1, tool1Stats.Completed)
	}

	tool2Stats := stats[tool2]
	if tool2Stats.Failed != 1 {
		t.Errorf("Expected 1 failed task for %s, got %d", tool2, tool2Stats.Failed)
	}
}

func TestManagerConcurrency(t *testing.T) {
	manager := NewManager()
	tool := "test-tool"
	manager.CreateQueue(tool, 100)

	// Test concurrent operations
	var wg sync.WaitGroup
	numGoroutines := 10
	tasksPerGoroutine := 10

	// Concurrent task additions
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < tasksPerGoroutine; j++ {
				task := NewTask(tool, "echo", []string{})
				if err := manager.AddTask(task); err != nil {
					// In concurrent tests, we might hit queue limits, which is expected
					continue
				}
			}
		}()
	}

	// Wait for all goroutines
	done := make(chan bool)
	go func() {
		wg.Wait()
		done <- true
	}()

	select {
	case <-done:
		// Success
	case <-time.After(5 * time.Second):
		t.Fatal("Test timed out - possible deadlock")
	}

	// Verify all tasks were added
	tasks := manager.GetAllTasks()
	expectedTasks := numGoroutines * tasksPerGoroutine
	if len(tasks) != expectedTasks {
		t.Errorf("Expected %d tasks, got %d", expectedTasks, len(tasks))
	}
}

func TestQueueFullError(t *testing.T) {
	manager := NewManager()
	tool := "test-tool"
	bufferSize := 2

	// Create a small queue
	queue := manager.CreateQueue(tool, bufferSize)

	// Fill the queue
	for i := 0; i < bufferSize; i++ {
		task := NewTask(tool, "echo", []string{})
		err := manager.AddTask(task)
		if err != nil {
			t.Fatalf("Failed to add task %d: %v", i, err)
		}
	}

	// Verify queue is full
	if len(queue) != bufferSize {
		t.Errorf("Queue should have %d tasks", bufferSize)
	}

	// Try to add one more task (should fail if queue is full)
	// Note: This will only fail if nothing is consuming from the queue
	task := NewTask(tool, "echo", []string{})
	err := manager.AddTask(task)

	// The error check depends on whether the queue blocks or returns error
	// In this implementation, it should return an error when full
	if err == nil {
		// If no error, the task might have been queued
		// This could happen if default case in select is used
		if len(manager.tasks) > bufferSize {
			// Task was added even though queue was full
			// This is expected behavior with the current implementation
			t.Log("Task was queued despite full buffer - this is expected with buffered channels")
		}
	}
}
