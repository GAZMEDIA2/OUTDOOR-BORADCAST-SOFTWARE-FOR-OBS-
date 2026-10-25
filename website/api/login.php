<?php
// The Van Link app signs in here: email + password -> device token + signed licence.
require __DIR__ . '/../inc/bootstrap.php';
require __DIR__ . '/../inc/api.php';
if ($_SERVER['REQUEST_METHOD'] !== 'POST') api_out(['ok' => false, 'error' => 'POST only'], 405);

$in = api_in();
$email = strtolower(trim((string)($in['email'] ?? '')));
$pass = (string)($in['password'] ?? '');
$deviceId = preg_replace('/[^A-Za-z0-9_-]/', '', (string)($in['device_id'] ?? ''));
$deviceName = substr(trim((string)($in['device_name'] ?? '')), 0, 120);
$role = in_array($in['role'] ?? '', ['studio', 'van'], true) ? $in['role'] : '';
if (strlen($deviceId) < 16) api_out(['ok' => false, 'error' => 'Update Van Link and try again.'], 400);

$key = 'api-login:' . client_ip();
if (too_many_attempts($key)) api_out(['ok' => false, 'error' => 'Too many attempts. Wait 15 minutes and try again.'], 429);
$u = one('SELECT * FROM users WHERE email = ?', [$email]);
if (!$u || !password_verify($pass, $u['pass_hash'])) {
    note_failed_attempt($key);
    api_out(['ok' => false, 'error' => 'Wrong email or password.'], 401);
}
if ($u['blocked']) api_out(['ok' => false, 'error' => 'This account is disabled. Contact ' . $CFG['support_email'] . '.'], 403);

$lic = effective_licence($u);
$plan = licence_active($lic) ? $lic['plan'] : 'trial';
$max = (int)plan_info($plan)['max_computers'];

$token = bin2hex(random_bytes(32));
$now = time();
$existing = one('SELECT * FROM devices WHERE user_id = ? AND device_id = ? AND revoked = 0', [$u['id'], $deviceId]);
if ($existing) {
    q('UPDATE devices SET token_hash = ?, name = ?, role = ?, last_seen = ? WHERE id = ?', [hash('sha256', $token), $deviceName, $role, $now, $existing['id']]);
} else {
    $n = (int)one('SELECT COUNT(*) n FROM devices WHERE user_id = ? AND revoked = 0', [$u['id']])['n'];
    if ($n >= $max) api_out(['ok' => false, 'error' => "Your plan allows $max computers and they are all in use. Sign one out at " . site_url('account.php') . '.'], 403);
    q('INSERT INTO devices (user_id, device_id, name, role, token_hash, created_at, last_seen) VALUES (?, ?, ?, ?, ?, ?, ?)',
        [$u['id'], $deviceId, $deviceName, $role, hash('sha256', $token), $now, $now]);
}
$dev = one('SELECT * FROM devices WHERE token_hash = ?', [hash('sha256', $token)]);
api_out(['token' => $token] + issue_for_device($u, $dev));
