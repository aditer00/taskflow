<?php

namespace Tests\Feature;

use App\Jobs\SendTaskNotification;
use App\Livewire\Tasks\TaskBoard;
use App\Mail\TaskNotificationMail;
use App\Models\Project;
use App\Models\Task;
use App\Models\User;
use Illuminate\Foundation\Testing\RefreshDatabase;
use Illuminate\Support\Facades\Mail;
use Illuminate\Support\Facades\Queue;
use Livewire\Livewire;
use Tests\TestCase;

class AsyncAndScheduleTest extends TestCase
{
    use RefreshDatabase;

    public function test_creating_or_completing_task_dispatches_notification_job(): void
    {
        Queue::fake();

        $user = User::factory()->create();
        $project = Project::create([
            'user_id' => $user->id,
            'name' => 'Queue Test Project',
        ]);

        Livewire::actingAs($user)
            ->test(TaskBoard::class, ['project' => $project])
            ->set('title', 'Test async queue')
            ->call('createTask')
            ->assertHasNoErrors();

        Queue::assertPushed(SendTaskNotification::class, function (SendTaskNotification $job) {
            return $job->event === 'created' && $job->task->title === 'Test async queue';
        });

        $task = Task::where('title', 'Test async queue')->firstOrFail();

        Livewire::actingAs($user)
            ->test(TaskBoard::class, ['project' => $project])
            ->call('updateStatus', $task->id, 'done')
            ->assertHasNoErrors();

        Queue::assertPushed(SendTaskNotification::class, function (SendTaskNotification $job) use ($task) {
            return $job->event === 'completed' && $job->task->id === $task->id;
        });
    }

    public function test_send_task_notification_job_sends_email(): void
    {
        Mail::fake();

        $user = User::factory()->create(['email' => 'devops@taskflow.local']);
        $project = Project::create([
            'user_id' => $user->id,
            'name' => 'Mailpit Project',
        ]);
        $task = Task::create([
            'project_id' => $project->id,
            'user_id' => $user->id,
            'title' => 'Verify Mailpit delivery',
            'status' => 'todo',
            'priority' => 'high',
        ]);

        SendTaskNotification::dispatchSync($task, 'created');

        Mail::assertSent(TaskNotificationMail::class, function (TaskNotificationMail $mail) use ($user) {
            return $mail->hasTo($user->email);
        });
    }

    public function test_heartbeat_command_runs_successfully(): void
    {
        $this->artisan('tasks:heartbeat')->assertExitCode(0);
    }
}
