<div class="space-y-8">
    <div class="flex flex-col sm:flex-row sm:items-end justify-between gap-2 border-b border-[#e5e3df] pb-5">
        <div>
            <h1 class="text-[28px] sm:text-[36px] font-semibold tracking-[-0.02em] leading-[1.2] text-[#1a1a1a]">
                Projects
            </h1>
            <p class="text-sm text-[#5d5b54] mt-1">Select a project to manage tasks and attachments.</p>
        </div>
        <span class="font-mono text-xs font-medium px-2.5 py-1 bg-[#f6f5f4] text-[#5d5b54] rounded-[6px]">
            {{ $projects->count() }} {{ $projects->count() === 1 ? 'project' : 'projects' }}
        </span>
    </div>

    <section aria-label="Create project" class="bg-[#fafaf9] border border-[#e5e3df] rounded-[12px] p-5 sm:p-6">
        <h2 class="text-[18px] font-semibold text-[#1a1a1a] mb-4">New project</h2>
        <form wire:submit="createProject" class="grid grid-cols-1 sm:grid-cols-12 gap-3 items-start">
            <div class="sm:col-span-4">
                <label for="project-name" class="block text-[13px] font-semibold text-[#1a1a1a] mb-1.5">Project name</label>
                <input id="project-name" type="text" wire:model="name" placeholder="Kubernetes Migration" required
                    class="w-full min-h-[44px] bg-[#ffffff] border border-[#c8c4be] rounded-[8px] px-3.5 py-2 text-sm text-[#1a1a1a] placeholder:text-[#787671] focus:border-[#0075de]">
                @error('name')
                    <p class="text-[#e03131] text-xs font-medium mt-1" role="alert">{{ $message }}</p>
                @enderror
            </div>

            <div class="sm:col-span-5">
                <label for="project-desc" class="block text-[13px] font-semibold text-[#1a1a1a] mb-1.5">Description (optional)</label>
                <input id="project-desc" type="text" wire:model="description" placeholder="Short scope or objective"
                    class="w-full min-h-[44px] bg-[#ffffff] border border-[#c8c4be] rounded-[8px] px-3.5 py-2 text-sm text-[#1a1a1a] placeholder:text-[#787671] focus:border-[#0075de]">
                @error('description')
                    <p class="text-[#e03131] text-xs font-medium mt-1" role="alert">{{ $message }}</p>
                @enderror
            </div>

            <div class="sm:col-span-3 sm:pt-6">
                <button type="submit" wire:loading.attr="disabled"
                    class="w-full min-h-[44px] px-[18px] py-2.5 bg-[#0075de] hover:bg-[#005bab] disabled:opacity-60 text-[#ffffff] text-sm font-medium rounded-[8px] cursor-pointer">
                    <span wire:loading.remove wire:target="createProject">Create project</span>
                    <span wire:loading wire:target="createProject">Saving...</span>
                </button>
            </div>
        </form>
    </section>

    <section aria-label="Project list">
        @if ($projects->isEmpty())
            <div class="bg-[#f6f5f4] border border-[#e5e3df] rounded-[12px] p-10 text-center">
                <p class="text-base font-semibold text-[#1a1a1a]">No projects yet</p>
                <p class="text-sm text-[#5d5b54] mt-1">Create your first project using the form above.</p>
            </div>
        @else
            <div class="space-y-3">
                @foreach ($projects as $project)
                    <div class="bg-[#ffffff] border border-[#e5e3df] hover:bg-[#fafaf9] rounded-[12px] p-5 flex flex-col sm:flex-row sm:items-center justify-between gap-4">
                        <div class="min-w-0">
                            <div class="flex items-center gap-3 flex-wrap">
                                <a href="{{ route('projects.show', $project) }}"
                                    class="text-base font-semibold text-[#1a1a1a] hover:text-[#0075de]">
                                    {{ $project->name }}
                                </a>
                                <span class="font-mono text-xs font-medium px-2.5 py-0.5 bg-[#e6e0f5] text-[#391c57] rounded-[6px]">
                                    {{ $project->tasks_count }} {{ $project->tasks_count === 1 ? 'task' : 'tasks' }}
                                </span>
                            </div>
                            @if ($project->description)
                                <p class="text-sm text-[#5d5b54] mt-1">{{ $project->description }}</p>
                            @endif
                        </div>

                        <div class="flex items-center gap-2 shrink-0">
                            <a href="{{ route('projects.show', $project) }}"
                                class="inline-flex items-center justify-center min-h-[44px] px-4 text-sm font-medium text-[#1a1a1a] border border-[#c8c4be] hover:bg-[#f6f5f4] rounded-[8px]">
                                Open board
                            </a>
                            <button type="button" wire:click="deleteProject({{ $project->id }})"
                                wire:confirm="Delete this project and all its tasks?"
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
