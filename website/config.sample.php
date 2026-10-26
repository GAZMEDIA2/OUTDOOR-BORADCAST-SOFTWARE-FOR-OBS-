<?php
// Van Link website settings. Copy this file to config.php and fill it in.
// Keep config.php private: it holds your database password and licence key.
return [
    'site_url' => 'https://vanlink.gazgoc.com.ng',
    'company'  => 'Van Link',
    'support_email' => 'support@gazgoc.com.ng',
    'support_whatsapp' => '',            // e.g. +2376xxxxxxxx (shown on the payment page)
    'timezone' => 'Africa/Lagos',

    // Owner/admin account(s): full admin page access and an unlimited Owner licence.
    'admin_emails' => ['ioghenevwogaga@gmail.com'],

    // MySQL database from cPanel (MySQL Database Wizard).
    'db_dsn'  => 'mysql:host=localhost;dbname=CPANELUSER_vanlink;charset=utf8mb4',
    'db_user' => 'CPANELUSER_vanlink',
    'db_pass' => 'PUT-YOUR-DATABASE-PASSWORD-HERE',

    // Licence signing key (secret). It must match the public key built into
    // the Van Link app. Never share it.
    'licence_secret_key' => 'PUT-THE-SECRET-KEY-HERE',

    // Plans. Prices in US dollars per year.
    'trial_days' => 14,
    'trial_minutes_per_day' => 30,
    'plans' => [
        'trial'    => ['price' => 0,    'max_vans' => 1,  'max_computers' => 2],
        'standard' => ['price' => 1100, 'max_vans' => 1,  'max_computers' => 2],
        'pro'      => ['price' => 1900, 'max_vans' => 10, 'max_computers' => 12],
        'owner'    => ['price' => 0,    'max_vans' => 99, 'max_computers' => 99],
    ],

    // How long the app keeps working without reaching this website (days).
    'offline_days' => 7,

    // Paystack (https://dashboard.paystack.com > Settings > API Keys & Webhooks).
    // All payments go through Paystack. Prices and records are in US dollars;
    // customers are charged the naira equivalent at usd_rate (change it any time in the admin page).
    // No Paystack webhook is needed: Van Link confirms each payment with Paystack directly
    // (so your existing webhook for other sites stays as it is).
    'paystack_secret_key' => '',          // sk_live_... (or sk_test_... while testing)
    'paystack_currency'   => 'NGN',       // the currency your Paystack account charges in
    'usd_rate'            => 1600,        // how many NGN per 1 USD when charging (update as needed)

];
