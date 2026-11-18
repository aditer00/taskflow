<?php

namespace App\Http\Controllers;

use Illuminate\Http\JsonResponse;
use Illuminate\Support\Facades\DB;
use Illuminate\Support\Facades\Log;
use Illuminate\Support\Facades\Redis;
use Throwable;

class HealthController extends Controller
{
    /**
     * Check connectivity to backing services (PostgreSQL & Redis).
     */
    public function __invoke(): JsonResponse
    {
        $checks = [
            'database' => 'ok',
            'redis' => 'ok',
        ];

        try {
            DB::connection()->getPdo();
        } catch (Throwable $e) {
            $checks['database'] = 'error';
            Log::error('Healthcheck database failure', ['error' => $e->getMessage()]);
        }

        try {
            Redis::connection()->ping();
        } catch (Throwable $e) {
            $checks['redis'] = 'error';
            Log::error('Healthcheck redis failure', ['error' => $e->getMessage()]);
        }

        $allHealthy = ! in_array('error', $checks, true);

        return response()->json([
            'status' => $allHealthy ? 'ok' : 'degraded',
            'checks' => $checks,
        ], $allHealthy ? 200 : 503);
    }
}
