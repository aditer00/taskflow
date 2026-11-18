<?php

namespace Tests\Feature;

use App\Livewire\Auth\Login;
use App\Livewire\Auth\Register;
use App\Models\User;
use Illuminate\Foundation\Testing\RefreshDatabase;
use Livewire\Livewire;
use Tests\TestCase;

class AuthTest extends TestCase
{
    use RefreshDatabase;

    public function test_unauthenticated_user_is_redirected_to_login(): void
    {
        $this->get('/projects')->assertRedirect('/login');
    }

    public function test_user_can_register_and_is_redirected_to_projects(): void
    {
        Livewire::test(Register::class)
            ->set('name', 'Aditya')
            ->set('email', 'aditya@taskflow.local')
            ->set('password', 'password123')
            ->set('password_confirmation', 'password123')
            ->call('register')
            ->assertRedirect('/projects');

        $this->assertAuthenticated();
        $this->assertDatabaseHas('users', ['email' => 'aditya@taskflow.local']);
    }

    public function test_user_can_login_and_logout(): void
    {
        $user = User::factory()->create([
            'email' => 'devops@taskflow.local',
            'password' => bcrypt('password123'),
        ]);

        Livewire::test(Login::class)
            ->set('email', 'devops@taskflow.local')
            ->set('password', 'password123')
            ->call('login')
            ->assertRedirect('/projects');

        $this->assertAuthenticatedAs($user);

        $this->post('/logout', ['_token' => csrf_token()])->assertRedirect('/login');
        $this->assertGuest();
    }
}
