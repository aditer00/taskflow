<?php

namespace App\Jobs;

use App\Mail\TaskNotificationMail;
use App\Models\Task;
use Illuminate\Contracts\Queue\ShouldQueue;
use Illuminate\Foundation\Queue\Queueable;
use Illuminate\Support\Facades\Log;
use Illuminate\Support\Facades\Mail;

class SendTaskNotification implements ShouldQueue
{
    use Queueable;

    public int $tries = 3;
    public int $backoff = 5;

    public function __construct(
        public Task $task,
        public string $event,
    ) {}

    public function handle(): void
    {
        $this->task->loadMissing(['user', 'project']);

        if (! $this->task->user?->email) {
            return;
        }

        Mail::to($this->task->user->email)
            ->send(new TaskNotificationMail($this->task, $this->event));

        Log::info('Async task notification sent', [
            'task_id' => $this->task->id,
            'event' => $this->event,
            'recipient' => $this->task->user->email,
        ]);
    }
}
