<?php

namespace App\Livewire\Auth;

use Illuminate\Support\Facades\Auth;
use Illuminate\Support\Facades\Log;
use Livewire\Attributes\Layout;
use Livewire\Component;

#[Layout('components.layouts.app')]
class Login extends Component
{
    public string $email = '';
    public string $password = '';

    public function login(): void
    {
        $credentials = $this->validate([
            'email' => ['required', 'email'],
            'password' => ['required', 'string'],
        ]);

        if (! Auth::attempt($credentials)) {
            $this->addError('email', 'Invalid email or password.');

            return;
        }

        session()->regenerate();
        Log::info('User logged in', ['user_id' => Auth::id(), 'email' => $this->email]);

        $this->redirect(route('projects.index'), navigate: false);
    }

    public function render()
    {
        return view('livewire.auth.login');
    }
}
