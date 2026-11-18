<?php

use App\Http\Controllers\HealthController;
use App\Livewire\Auth\Login;
use App\Livewire\Auth\Register;
use App\Livewire\Projects\ProjectIndex;
use App\Livewire\Tasks\TaskBoard;
use App\Models\Task;
use Illuminate\Http\Request;
use Illuminate\Support\Facades\Auth;
use Illuminate\Support\Facades\Route;
use Illuminate\Support\Facades\Storage;

Route::get('/health', HealthController::class)
    ->withoutMiddleware(['web'])
    ->name('health');

Route::get('/', fn () => view('landing'))->name('home');

Route::middleware('guest')->group(function () {
    Route::get('/login', Login::class)->name('login');
    Route::get('/register', Register::class)->name('register');
});

Route::middleware('auth')->group(function () {
    Route::get('/projects', ProjectIndex::class)->name('projects.index');
    Route::get('/projects/{project}', TaskBoard::class)->name('projects.show');

    Route::get('/tasks/{task}/attachment', function (Task $task) {
        abort_unless($task->user_id === Auth::id(), 403);
        abort_unless($task->attachment_path && Storage::disk('s3')->exists($task->attachment_path), 404);

        return Storage::disk('s3')->download($task->attachment_path);
    })->name('tasks.attachment');

    Route::post('/logout', function (Request $request) {
        Auth::logout();
        $request->session()->invalidate();
        $request->session()->regenerateToken();

        return redirect()->route('login');
    })->name('logout');
});
