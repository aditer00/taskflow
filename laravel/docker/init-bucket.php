<?php

require '/var/www/html/vendor/autoload.php';

$client = new Aws\S3\S3Client([
    'version' => 'latest',
    'region' => getenv('AWS_DEFAULT_REGION') ?: 'us-east-1',
    'endpoint' => getenv('AWS_ENDPOINT') ?: 'http://minio:9000',
    'use_path_style_endpoint' => true,
    'credentials' => [
        'key' => getenv('AWS_ACCESS_KEY_ID') ?: 'minioadmin',
        'secret' => getenv('AWS_SECRET_ACCESS_KEY') ?: 'minioadmin',
    ],
]);

$bucket = getenv('AWS_BUCKET') ?: 'taskflow';

for ($i = 0; $i < 15; $i++) {
    try {
        if (! $client->doesBucketExist($bucket)) {
            $client->createBucket(['Bucket' => $bucket]);
        }
        fwrite(STDOUT, "Bucket {$bucket} ready\n");
        exit(0);
    } catch (Throwable $e) {
        sleep(1);
    }
}

fwrite(STDERR, "Failed to initialize bucket {$bucket}\n");
exit(1);
