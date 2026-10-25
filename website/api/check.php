<?php
// The app checks in regularly: refreshes its licence and reports trial broadcast time.
require __DIR__ . '/../inc/bootstrap.php';
require __DIR__ . '/../inc/api.php';
if ($_SERVER['REQUEST_METHOD'] !== 'POST') api_out(['ok' => false, 'error' => 'POST only'], 405);

$in = api_in();
$dev = device_from_token((string)($in['token'] ?? ''));
if (!$dev) api_out(['ok' => false, 'signed_out' => true, 'error' => 'This computer has been signed out. Sign in again.'], 401);
$u = one('SELECT * FROM users WHERE id = ?', [$dev['user_id']]);
if (!$u || $u['blocked']) api_out(['ok' => false, 'signed_out' => true, 'error' => 'This account is disabled.'], 403);

// Trial time used since the last check (capped so a bad clock can't eat a whole day).
$used = max(0, min((int)($in['live_seconds'] ?? 0), 900));
if ($used > 0) {
    $day = date('Y-m-d');
    $row = one('SELECT id FROM usage_log WHERE user_id = ? AND day = ?', [$u['id'], $day]);
    if ($row) q('UPDATE usage_log SET seconds = seconds + ? WHERE id = ?', [$used, $row['id']]);
    else q('INSERT INTO usage_log (user_id, day, seconds) VALUES (?, ?, ?)', [$u['id'], $day, $used]);
}
$role = in_array($in['role'] ?? '', ['studio', 'van'], true) ? $in['role'] : $dev['role'];
q('UPDATE devices SET last_seen = ?, role = ? WHERE id = ?', [time(), $role, $dev['id']]);
api_out(issue_for_device($u, $dev));
