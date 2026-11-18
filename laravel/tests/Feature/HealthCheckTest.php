<?php

namespace Tests\Feature;

use Illuminate\Support\Facades\Redis;
use Mockery;
use Tests\TestCase;

class HealthCheckTest extends TestCase
{
    public function test_health_endpoint_returns_200_when_services_are_healthy(): void
    {
        $redisConnection = Mockery::mock();
        $redisConnection->shouldReceive('ping')->once()->andReturn('PONG');
        Redis::shouldReceive('connection')->once()->andReturn($redisConnection);

        $response = $this->getJson('/health');

        $response->assertStatus(200)
            ->assertJson([
                'status' => 'ok',
                'checks' => [
                    'database' => 'ok',
                    'redis' => 'ok',
                ],
            ]);
    }

    public function test_health_endpoint_returns_503_when_redis_is_unreachable(): void
    {
        Redis::shouldReceive('connection')->once()->andThrow(new \RuntimeException('Redis down'));

        $response = $this->getJson('/health');

        $response->assertStatus(503)
            ->assertJson([
                'status' => 'degraded',
                'checks' => [
                    'database' => 'ok',
                    'redis' => 'error',
                ],
            ]);
    }
}
