# syntax=docker/dockerfile:1

FROM composer:2.11.2 AS vendor
WORKDIR /app
COPY composer.json composer.lock ./
RUN composer install --no-dev --no-interaction --prefer-dist --optimize-autoloader --no-scripts

FROM node:22.23.5-alpine AS assets
WORKDIR /app
COPY package.json package-lock.json* ./
RUN npm ci
COPY resources ./resources
COPY vite.config.js ./
RUN npm run build

FROM php:8.4.27-fpm-alpine AS runtime

RUN apk add --no-cache \
    libpq-dev \
    libzip-dev \
    oniguruma-dev \
    && docker-php-ext-install -j$(nproc) \
    pdo_pgsql \
    pcntl \
    zip \
    bcmath \
    && rm -rf /var/cache/apk/*

RUN sed -i 's|;catch_workers_output = yes|catch_workers_output = yes|' /usr/local/etc/php-fpm.d/www.conf \
    && sed -i 's|;decorate_workers_output = no|decorate_workers_output = no|' /usr/local/etc/php-fpm.d/www.conf

WORKDIR /var/www/html

COPY --chown=www-data:www-data . .
COPY --from=vendor --chown=www-data:www-data /app/vendor ./vendor
COPY --from=assets --chown=www-data:www-data /app/public/build ./public/build

RUN mkdir -p storage/framework/cache storage/framework/sessions storage/framework/views bootstrap/cache \
    && chown -R www-data:www-data storage bootstrap/cache \
    && php artisan package:discover --ansi || true

USER www-data
EXPOSE 9000
CMD ["php-fpm"]

FROM runtime AS dev
USER root
RUN apk add --no-cache sqlite-dev \
    && docker-php-ext-install -j$(nproc) pdo_sqlite \
    && rm -rf /var/cache/apk/*
COPY --from=composer:2.11.2 /usr/bin/composer /usr/bin/composer
RUN composer install --no-interaction --prefer-dist --no-scripts \
    && chown -R www-data:www-data vendor
USER www-data

FROM nginx:1.27.5-alpine AS web
COPY docker/nginx/default.conf /etc/nginx/conf.d/default.conf
COPY --from=runtime /var/www/html/public /var/www/html/public
EXPOSE 80
