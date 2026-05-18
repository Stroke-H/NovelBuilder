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
	Error        string `json:"error,omitempty"`
	StartedAt    string `json:"started_at"`
	UpdatedAt    string `json:"updated_at"`
	FinishedAt   string `json:"finished_at,omitempty"`
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
	novelRuntimeTaskRegistry.mu.Lock()
	pruneFinishedNovelRuntimeTasksLocked()
	novelRuntimeTaskRegistry.mu.Unlock()
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

func (entry *novelRuntimeTaskEntry) complete(status string, title string, errText string) {
	if entry == nil {
		return
	}
	if status == "" {
		status = "completed"
	}
	now := time.Now().Format("2006-01-02 15:04:05")
	novelRuntimeTaskRegistry.mu.Lock()
	defer novelRuntimeTaskRegistry.mu.Unlock()
	stored := novelRuntimeTaskRegistry.tasks[entry.task.ID]
	if stored == nil {
		return
	}
	if title != "" {
		stored.task.Title = title
	}
	stored.task.Status = status
	stored.task.Error = errText
	stored.task.UpdatedAt = now
	stored.task.FinishedAt = now
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

func (entry *novelRuntimeTaskEntry) update(title string, status string) {
	if entry == nil {
		return
	}
	novelRuntimeTaskRegistry.mu.Lock()
	defer novelRuntimeTaskRegistry.mu.Unlock()
	stored := novelRuntimeTaskRegistry.tasks[entry.task.ID]
	if stored == nil {
		return
	}
	if title != "" {
		stored.task.Title = title
	}
	if status != "" {
		stored.task.Status = status
	}
	stored.task.UpdatedAt = time.Now().Format("2006-01-02 15:04:05")
}

func listNovelRuntimeTasks() []NovelRuntimeTask {
	novelRuntimeTaskRegistry.mu.Lock()
	defer novelRuntimeTaskRegistry.mu.Unlock()
	pruneFinishedNovelRuntimeTasksLocked()

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
	if entry != nil && entry.task.FinishedAt != "" {
		novelRuntimeTaskRegistry.mu.Unlock()
		return fmt.Errorf("task already finished")
	}
	novelRuntimeTaskRegistry.mu.Unlock()
	if entry == nil {
		return fmt.Errorf("task not found")
	}
	entry.cancelTask()
	return nil
}

func pruneFinishedNovelRuntimeTasksLocked() {
	cutoff := time.Now().Add(-10 * time.Minute)
	for id, entry := range novelRuntimeTaskRegistry.tasks {
		if entry == nil || entry.task.FinishedAt == "" {
			continue
		}
		finishedAt, err := time.ParseInLocation("2006-01-02 15:04:05", entry.task.FinishedAt, time.Local)
		if err != nil || finishedAt.Before(cutoff) {
			delete(novelRuntimeTaskRegistry.tasks, id)
		}
	}
}
