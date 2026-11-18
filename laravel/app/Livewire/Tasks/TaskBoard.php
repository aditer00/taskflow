<?php

namespace App\Livewire\Tasks;

use App\Jobs\SendTaskNotification;
use App\Models\Project;
use App\Models\Task;
use Illuminate\Support\Facades\Auth;
use Illuminate\Support\Facades\Log;
use Illuminate\Support\Facades\Storage;
use Illuminate\Validation\Rule;
use Livewire\Attributes\Layout;
use Livewire\Component;
use Livewire\WithFileUploads;

#[Layout('components.layouts.app')]
class TaskBoard extends Component
{
    use WithFileUploads;

    public Project $project;
    public string $title = '';
    public string $description = '';
    public string $priority = 'medium';
    public ?string $due_date = null;
    public $attachment = null;

    public function mount(Project $project): void
    {
        abort_unless($project->user_id === Auth::id(), 403);
        $this->project = $project;
    }

    public function createTask(): void
    {
        abort_unless($this->project->user_id === Auth::id(), 403);

        $validated = $this->validate([
            'title' => ['required', 'string', 'max:160'],
            'description' => ['nullable', 'string', 'max:1000'],
            'priority' => ['required', Rule::in(Task::PRIORITIES)],
            'due_date' => ['nullable', 'date'],
            'attachment' => ['nullable', 'file', 'max:5120', 'mimes:pdf,png,jpg,jpeg,txt,md'],
        ]);

        $attachmentPath = null;
        if ($this->attachment) {
            $attachmentPath = Storage::disk('s3')->putFile('attachments', $this->attachment);
        }

        $task = $this->project->tasks()->create([
            'user_id' => Auth::id(),
            'title' => $validated['title'],
            'description' => $validated['description'] ?? null,
            'status' => 'todo',
            'priority' => $validated['priority'],
            'due_date' => $validated['due_date'] ?? null,
            'attachment_path' => $attachmentPath,
        ]);

        Log::info('Task created', [
            'task_id' => $task->id,
            'project_id' => $this->project->id,
            'has_attachment' => (bool) $attachmentPath,
        ]);

        SendTaskNotification::dispatch($task, 'created');

        $this->reset(['title', 'description', 'due_date', 'attachment']);
        $this->priority = 'medium';
    }

    public function updateStatus(int $taskId, string $status): void
    {
        abort_unless(in_array($status, Task::STATUSES, true), 422);

        $task = $this->project->tasks()->findOrFail($taskId);
        abort_unless($task->user_id === Auth::id(), 403);

        $task->update(['status' => $status]);

        Log::info('Task status updated', [
            'task_id' => $task->id,
            'status' => $status,
        ]);

        if ($status === 'done') {
            SendTaskNotification::dispatch($task, 'completed');
        }
    }

    public function deleteTask(int $taskId): void
    {
        $task = $this->project->tasks()->findOrFail($taskId);
        abort_unless($task->user_id === Auth::id(), 403);

        if ($task->attachment_path) {
            Storage::disk('s3')->delete($task->attachment_path);
        }

        $task->delete();
        Log::info('Task deleted', ['task_id' => $taskId]);
    }

    public function render()
    {
        $tasks = $this->project->tasks()->latest()->get();

        return view('livewire.tasks.task-board', [
            'tasks' => $tasks,
        ]);
    }
}
