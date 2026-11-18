<?php

namespace Tests\Feature;

use App\Livewire\Projects\ProjectIndex;
use App\Livewire\Tasks\TaskBoard;
use App\Models\Project;
use App\Models\Task;
use App\Models\User;
use Illuminate\Foundation\Testing\RefreshDatabase;
use Illuminate\Http\UploadedFile;
use Illuminate\Support\Facades\Storage;
use Livewire\Livewire;
use Tests\TestCase;

class ProjectAndTaskTest extends TestCase
{
    use RefreshDatabase;

    public function test_user_can_create_and_delete_own_project(): void
    {
        $user = User::factory()->create();

        Livewire::actingAs($user)
            ->test(ProjectIndex::class)
            ->set('name', 'Kubernetes Migration')
            ->set('description', 'Move workloads to 12-factor')
            ->call('createProject')
            ->assertHasNoErrors();

        $project = Project::where('name', 'Kubernetes Migration')->firstOrFail();
        $this->assertSame($user->id, $project->user_id);

        Livewire::actingAs($user)
            ->test(ProjectIndex::class)
            ->call('deleteProject', $project->id)
            ->assertHasNoErrors();

        $this->assertDatabaseMissing('projects', ['id' => $project->id]);
    }

    public function test_user_cannot_view_another_users_project(): void
    {
        $owner = User::factory()->create();
        $other = User::factory()->create();

        $project = Project::create([
            'user_id' => $owner->id,
            'name' => 'Private Infra',
        ]);

        $this->actingAs($other)
            ->get(route('projects.show', $project))
            ->assertStatus(403);
    }

    public function test_user_can_create_task_with_s3_attachment_and_update_status(): void
    {
        Storage::fake('s3');
        $user = User::factory()->create();
        $project = Project::create([
            'user_id' => $user->id,
            'name' => '12-Factor Lab',
        ]);

        $file = UploadedFile::fake()->createWithContent('spec.pdf', '%PDF-1.4 sample content');

        Livewire::actingAs($user)
            ->test(TaskBoard::class, ['project' => $project])
            ->set('title', 'Configure stderr logs')
            ->set('priority', 'high')
            ->set('attachment', $file)
            ->call('createTask')
            ->assertHasNoErrors();

        $task = Task::where('title', 'Configure stderr logs')->firstOrFail();
        $this->assertSame('todo', $task->status);
        $this->assertNotNull($task->attachment_path);
        Storage::disk('s3')->assertExists($task->attachment_path);

        Livewire::actingAs($user)
            ->test(TaskBoard::class, ['project' => $project])
            ->call('updateStatus', $task->id, 'done')
            ->assertHasNoErrors();

        $this->assertSame('done', $task->fresh()->status);
    }

    public function test_task_attachment_rejects_oversized_or_disallowed_file(): void
    {
        Storage::fake('s3');
        $user = User::factory()->create();
        $project = Project::create([
            'user_id' => $user->id,
            'name' => 'Security Check',
        ]);

        $badFile = UploadedFile::fake()->create('payload.exe', 100, 'application/octet-stream');

        Livewire::actingAs($user)
            ->test(TaskBoard::class, ['project' => $project])
            ->set('title', 'Invalid attachment task')
            ->set('attachment', $badFile)
            ->call('createTask')
            ->assertHasErrors(['attachment']);

        $this->assertDatabaseMissing('tasks', ['title' => 'Invalid attachment task']);
        $this->assertCount(0, Storage::disk('s3')->allFiles('attachments'));
    }
}
