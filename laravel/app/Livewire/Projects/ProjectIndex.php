<?php

namespace App\Livewire\Projects;

use App\Models\Project;
use Illuminate\Support\Facades\Auth;
use Illuminate\Support\Facades\Log;
use Livewire\Attributes\Layout;
use Livewire\Component;

#[Layout('components.layouts.app')]
class ProjectIndex extends Component
{
    public string $name = '';
    public string $description = '';

    public function createProject(): void
    {
        $validated = $this->validate([
            'name' => ['required', 'string', 'max:120'],
            'description' => ['nullable', 'string', 'max:500'],
        ]);

        $project = Auth::user()->projects()->create($validated);

        Log::info('Project created', ['project_id' => $project->id, 'user_id' => Auth::id()]);

        $this->reset(['name', 'description']);
    }

    public function deleteProject(int $projectId): void
    {
        $project = Project::findOrFail($projectId);
        abort_unless($project->user_id === Auth::id(), 403);

        $project->delete();
        Log::info('Project deleted', ['project_id' => $projectId, 'user_id' => Auth::id()]);
    }

    public function render()
    {
        $projects = Auth::user()
            ->projects()
            ->withCount('tasks')
            ->latest()
            ->get();

        return view('livewire.projects.project-index', [
            'projects' => $projects,
        ]);
    }
}
