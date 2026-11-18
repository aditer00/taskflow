<?php

namespace App\Console\Commands;

use App\Models\Task;
use Illuminate\Console\Command;
use Illuminate\Support\Facades\Log;

class TaskHeartbeatCommand extends Command
{
    protected $signature = 'tasks:heartbeat';

    protected $description = 'Emit periodic task summary to stderr log stream (Factor VIII & XI)';

    public function handle(): int
    {
        $total = Task::count();
        $open = Task::where('status', '!=', 'done')->count();
        $overdue = Task::where('status', '!=', 'done')
            ->whereNotNull('due_date')
            ->whereDate('due_date', '<', now()->toDateString())
            ->count();

        Log::info('Scheduler heartbeat', [
            'total_tasks' => $total,
            'open_tasks' => $open,
            'overdue_tasks' => $overdue,
        ]);

        $this->info("Heartbeat emitted: total={$total}, open={$open}, overdue={$overdue}");

        return self::SUCCESS;
    }
}
