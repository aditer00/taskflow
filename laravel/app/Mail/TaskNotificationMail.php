<?php

namespace App\Mail;

use App\Models\Task;
use Illuminate\Bus\Queueable;
use Illuminate\Mail\Mailable;
use Illuminate\Mail\Mailables\Content;
use Illuminate\Mail\Mailables\Envelope;
use Illuminate\Queue\SerializesModels;

class TaskNotificationMail extends Mailable
{
    use Queueable, SerializesModels;

    public function __construct(
        public Task $task,
        public string $event,
    ) {}

    public function envelope(): Envelope
    {
        $action = $this->event === 'completed' ? 'Completed' : 'Created';

        return new Envelope(
            subject: "[TaskFlow] Task {$action}: {$this->task->title}",
        );
    }

    public function content(): Content
    {
        return new Content(
            view: 'emails.task-notification',
        );
    }
}
