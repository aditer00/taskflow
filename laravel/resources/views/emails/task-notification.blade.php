<div style="font-family: sans-serif; color: #1e293b;">
    <h2>TaskFlow Notification</h2>
    <p>
        Task <strong>{{ $task->title }}</strong> in project
        <strong>{{ $task->project?->name ?? 'Unknown' }}</strong>
        was <strong>{{ $event }}</strong>.
    </p>
    <ul>
        <li>Status: {{ $task->status }}</li>
        <li>Priority: {{ $task->priority }}</li>
        @if ($task->due_date)
            <li>Due Date: {{ $task->due_date->format('Y-m-d') }}</li>
        @endif
    </ul>
</div>
