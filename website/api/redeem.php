<?php
// The app activates a registration code for the signed-in account.
require __DIR__ . '/../inc/bootstrap.php';
require __DIR__ . '/../inc/api.php';
if ($_SERVER['REQUEST_METHOD'] !== 'POST') api_out(['ok' => false, 'error' => 'POST only'], 405);
$in = api_in();
$dev = device_from_token((string)($in['token'] ?? ''));
if (!$dev) api_out(['ok' => false, 'signed_out' => true, 'error' => 'Sign in first.'], 401);
$key = 'redeem:' . client_ip();
if (too_many_attempts($key)) api_out(['ok' => false, 'error' => 'Too many attempts. Wait 15 minutes and try again.'], 429);
$err = redeem_code((int)$dev['user_id'], (string)($in['code'] ?? ''));
if ($err !== '') { note_failed_attempt($key); api_out(['ok' => false, 'error' => $err], 400); }
$u = one('SELECT * FROM users WHERE id = ?', [$dev['user_id']]);
api_out(issue_for_device($u, $dev));
