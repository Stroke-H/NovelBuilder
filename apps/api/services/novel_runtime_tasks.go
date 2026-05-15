package services

import (
	"context"
	"fmt"
	"sort"
	"sync"
	"time"

	"github.com/google/uuid"
)

type NovelRuntimeTask struct {
	ID           string `json:"id"`
	ProjectID    string `json:"project_id"`
	ProjectTitle string `json:"project_title"`
	Kind         string `json:"kind"`
	Title        string `json:"title"`
	Status       string `json:"status"`
	StartedAt    string `json:"started_at"`
	UpdatedAt    string `json:"updated_at"`
}

type novelRuntimeTaskEntry struct {
	task   NovelRuntimeTask
	cancel context.CancelFunc
}

var novelRuntimeTaskRegistry = struct {
	mu    sync.Mutex
	tasks map[string]*novelRuntimeTaskEntry
}{
	tasks: map[string]*novelRuntimeTaskEntry{},
}

func beginNovelRuntimeTask(project NovelProject, kind string, title string) (context.Context, *novelRuntimeTaskEntry) {
	ctx, cancel := context.WithCancel(context.Background())
	now := time.Now().Format("2006-01-02 15:04:05")
	entry := &novelRuntimeTaskEntry{
		task: NovelRuntimeTask{
			ID:           "TASK-" + uuid.NewString(),
			ProjectID:    project.ID,
			ProjectTitle: project.Title,
			Kind:         kind,
			Title:        title,
			Status:       "running",
			StartedAt:    now,
			UpdatedAt:    now,
		},
		cancel: cancel,
	}

	novelRuntimeTaskRegistry.mu.Lock()
	novelRuntimeTaskRegistry.tasks[entry.task.ID] = entry
	novelRuntimeTaskRegistry.mu.Unlock()

	return ctx, entry
}

func (entry *novelRuntimeTaskEntry) finish() {
	if entry == nil {
		return
	}
	novelRuntimeTaskRegistry.mu.Lock()
	delete(novelRuntimeTaskRegistry.tasks, entry.task.ID)
	novelRuntimeTaskRegistry.mu.Unlock()
}

func (entry *novelRuntimeTaskEntry) cancelTask() {
	if entry == nil {
		return
	}
	novelRuntimeTaskRegistry.mu.Lock()
	stored := novelRuntimeTaskRegistry.tasks[entry.task.ID]
	if stored != nil {
		stored.task.Status = "cancelling"
		stored.task.UpdatedAt = time.Now().Format("2006-01-02 15:04:05")
	}
	novelRuntimeTaskRegistry.mu.Unlock()
	entry.cancel()
}

func listNovelRuntimeTasks() []NovelRuntimeTask {
	novelRuntimeTaskRegistry.mu.Lock()
	defer novelRuntimeTaskRegistry.mu.Unlock()

	tasks := make([]NovelRuntimeTask, 0, len(novelRuntimeTaskRegistry.tasks))
	for _, entry := range novelRuntimeTaskRegistry.tasks {
		tasks = append(tasks, entry.task)
	}

	sort.SliceStable(tasks, func(i, j int) bool {
		return tasks[i].StartedAt > tasks[j].StartedAt
	})
	return tasks
}

func cancelNovelRuntimeTask(taskID string) error {
	novelRuntimeTaskRegistry.mu.Lock()
	entry := novelRuntimeTaskRegistry.tasks[taskID]
	novelRuntimeTaskRegistry.mu.Unlock()
	if entry == nil {
		return fmt.Errorf("task not found")
	}
	entry.cancelTask()
	return nil
}
