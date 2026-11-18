<?php

namespace Tests\Feature;

use Tests\TestCase;

class ExampleTest extends TestCase
{
    public function test_the_landing_page_returns_200_and_shows_taskflow(): void
    {
        $response = $this->get('/');

        $response->assertStatus(200)
            ->assertSee('TaskFlow');
    }
}
