<div class="space-y-8">
    <div class="border-b border-[#e5e3df] pb-5">
        <nav aria-label="Breadcrumb" class="mb-2">
            <a href="{{ route('projects.index') }}" class="inline-flex items-center min-h-[44px] text-sm font-medium text-[#0075de] hover:text-[#005bab] hover:underline underline-offset-4">
                Back to projects
            </a>
        </nav>

        <div class="flex flex-col sm:flex-row sm:items-end justify-between gap-3">
            <div>
                <h1 class="text-[28px] sm:text-[36px] font-semibold tracking-[-0.02em] leading-[1.2] text-[#1a1a1a]">
                    {{ $project->name }}
                </h1>
                @if ($project->description)
                    <p class="text-sm text-[#5d5b54] mt-1">{{ $project->description }}</p>
                @endif
            </div>

            <div class="flex items-center gap-2 font-mono text-xs font-medium">
                <span class="px-2.5 py-1 bg-[#f0eeec] text-[#37352f] rounded-[6px]">Total: {{ $tasks->count() }}</span>
                <span class="px-2.5 py-1 bg-[#dcecfa] text-[#0a1530] rounded-[6px]">Open: {{ $tasks->where('status', '!=', 'done')->count() }}</span>
                <span class="px-2.5 py-1 bg-[#d9f3e1] text-[#14532d] rounded-[6px]">Done: {{ $tasks->where('status', 'done')->count() }}</span>
            </div>
        </div>
    </div>

    <section aria-label="Add task" class="bg-[#fafaf9] border border-[#e5e3df] rounded-[12px] p-5 sm:p-6">
        <h2 class="text-[18px] font-semibold text-[#1a1a1a] mb-4">Add task</h2>
        <form wire:submit="createTask" class="grid grid-cols-1 sm:grid-cols-2 gap-4">
            <div>
                <label for="task-title" class="block text-[13px] font-semibold text-[#1a1a1a] mb-1.5">Title</label>
                <input id="task-title" type="text" wire:model="title" required placeholder="Configure stderr logging"
                    class="w-full min-h-[44px] bg-[#ffffff] border border-[#c8c4be] rounded-[8px] px-3.5 py-2 text-sm text-[#1a1a1a] placeholder:text-[#787671] focus:border-[#0075de]">
                @error('title')
                    <p class="text-[#e03131] text-xs font-medium mt-1" role="alert">{{ $message }}</p>
                @enderror
            </div>

            <div class="grid grid-cols-2 gap-3">
                <div>
                    <label for="task-priority" class="block text-[13px] font-semibold text-[#1a1a1a] mb-1.5">Priority</label>
                    <select id="task-priority" wire:model="priority"
                        class="w-full min-h-[44px] bg-[#ffffff] border border-[#c8c4be] rounded-[8px] px-3 py-2 text-sm text-[#1a1a1a] focus:border-[#0075de]">
                        <option value="low">Low</option>
                        <option value="medium">Medium</option>
                        <option value="high">High</option>
                    </select>
                </div>
                <div>
                    <label for="task-due" class="block text-[13px] font-semibold text-[#1a1a1a] mb-1.5">Due date</label>
                    <input id="task-due" type="date" wire:model="due_date"
                        class="w-full min-h-[44px] bg-[#ffffff] border border-[#c8c4be] rounded-[8px] px-3 py-2 text-sm text-[#1a1a1a] focus:border-[#0075de]">
                </div>
            </div>

            <div>
                <label for="task-desc" class="block text-[13px] font-semibold text-[#1a1a1a] mb-1.5">Description (optional)</label>
                <input id="task-desc" type="text" wire:model="description" placeholder="Acceptance criteria or notes"
                    class="w-full min-h-[44px] bg-[#ffffff] border border-[#c8c4be] rounded-[8px] px-3.5 py-2 text-sm text-[#1a1a1a] placeholder:text-[#787671] focus:border-[#0075de]">
            </div>

            <div>
                <label for="task-attachment" class="block text-[13px] font-semibold text-[#1a1a1a] mb-1.5">Attachment (S3, max 5MB)</label>
                <input id="task-attachment" type="file" wire:model="attachment"
                    class="w-full min-h-[44px] bg-[#ffffff] border border-[#c8c4be] rounded-[8px] px-2 py-1.5 text-xs text-[#5d5b54] file:mr-3 file:py-1.5 file:px-3 file:rounded-[6px] file:border-0 file:bg-[#f0eeec] file:text-[#1a1a1a] file:font-medium hover:file:bg-[#e5e3df]">
                <div wire:loading wire:target="attachment" class="text-xs text-[#0075de] mt-1">
                    Uploading file...
                </div>
                @error('attachment')
                    <p class="text-[#e03131] text-xs font-medium mt-1" role="alert">{{ $message }}</p>
                @enderror
            </div>

            <div class="sm:col-span-2 pt-1">
                <button type="submit" wire:loading.attr="disabled"
                    class="min-h-[44px] px-[18px] py-2.5 bg-[#0075de] hover:bg-[#005bab] disabled:opacity-60 text-[#ffffff] text-sm font-medium rounded-[8px] cursor-pointer">
                    <span wire:loading.remove wire:target="createTask">Create task</span>
                    <span wire:loading wire:target="createTask">Creating task...</span>
                </button>
            </div>
        </form>
    </section>

    <section aria-label="Task list">
        @if ($tasks->isEmpty())
            <div class="bg-[#f6f5f4] border border-[#e5e3df] rounded-[12px] p-10 text-center">
                <p class="text-base font-semibold text-[#1a1a1a]">No tasks in this project yet</p>
                <p class="text-sm text-[#5d5b54] mt-1">Add a task above to trigger queue notifications and S3 storage.</p>
            </div>
        @else
            <div class="space-y-3">
                @foreach ($tasks as $task)
                    @php
                        $priorityStyle = match ($task->priority) {
                            'high' => 'bg-[#fde0ec] text-[#a02e6d]',
                            'medium' => 'bg-[#ffe8d4] text-[#793400]',
                            default => 'bg-[#f0eeec] text-[#5d5b54]',
                        };
                    @endphp
                    <div class="bg-[#ffffff] border border-[#e5e3df] hover:bg-[#fafaf9] rounded-[12px] p-5 flex flex-col sm:flex-row sm:items-center justify-between gap-4">
                        <div class="min-w-0 space-y-1.5">
                            <div class="flex items-center gap-2.5 flex-wrap">
                                <span class="text-base font-medium {{ $task->status === 'done' ? 'line-through text-[#787671]' : 'text-[#1a1a1a]' }}">
                                    {{ $task->title }}
                                </span>
                                <span class="font-mono text-xs font-medium capitalize px-2.5 py-0.5 rounded-[6px] {{ $priorityStyle }}">
                                    {{ $task->priority }}
                                </span>
                            </div>

                            @if ($task->description)
                                <p class="text-sm text-[#5d5b54]">{{ $task->description }}</p>
                            @endif

                            @if ($task->due_date || $task->attachment_path)
                                <div class="flex items-center gap-4 flex-wrap text-xs text-[#5d5b54] pt-0.5">
                                    @if ($task->due_date)
                                        <span class="font-mono">Due: {{ $task->due_date->format('Y-m-d') }}</span>
                                    @endif
                                    @if ($task->attachment_path)
                                        <a href="{{ route('tasks.attachment', $task) }}" target="_blank"
                                            class="inline-flex items-center min-h-[44px] sm:min-h-0 font-medium text-[#0075de] hover:text-[#005bab] hover:underline underline-offset-4">
                                            Download attachment
                                        </a>
                                    @endif
                                </div>
                            @endif
                        </div>

                        <div class="flex items-center gap-2 shrink-0">
                            <label for="status-{{ $task->id }}" class="sr-only">Task status</label>
                            <select id="status-{{ $task->id }}"
                                wire:change="updateStatus({{ $task->id }}, $event.target.value)"
                                class="min-h-[44px] bg-[#ffffff] border border-[#c8c4be] rounded-[8px] px-3 py-1.5 text-sm font-medium text-[#1a1a1a] focus:border-[#0075de]">
                                <option value="todo" @selected($task->status === 'todo')>Todo</option>
                                <option value="in_progress" @selected($task->status === 'in_progress')>In Progress</option>
                                <option value="done" @selected($task->status === 'done')>Done</option>
                            </select>

                            <button type="button" wire:click="deleteTask({{ $task->id }})"
                                class="inline-flex items-center justify-center min-h-[44px] px-3.5 text-sm font-medium text-[#e03131] border border-[#e5e3df] hover:bg-[#fde0ec] rounded-[8px] cursor-pointer">
                                Delete
                            </button>
                        </div>
                    </div>
                @endforeach
            </div>
        @endif
    </section>
</div>
