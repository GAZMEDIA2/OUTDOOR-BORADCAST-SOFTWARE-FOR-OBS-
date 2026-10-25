<?php
// Shared code for the app's sign-in API.

// Builds the signed licence a device receives.
function issue_for_device(array $u, array $dev): array {
    global $CFG;
    $lic = effective_licence($u);
    $active = licence_active($lic);
    $plan = $active ? $lic['plan'] : 'none';
    $info = $active ? plan_info($plan) : ['max_vans' => 0];
    $now = time();
    $remaining = null;
    if ($plan === 'trial') {
        $used = (int)(one('SELECT seconds FROM usage_log WHERE user_id = ? AND day = ?', [$u['id'], date('Y-m-d')])['seconds'] ?? 0);
        $remaining = max(0, (int)$CFG['trial_minutes_per_day'] * 60 - $used);
    }
    $payload = [
        'v' => 1,
        'email' => $u['email'],
        'name' => $u['name'],
        'plan' => $plan,
        'expires' => $active ? (int)$lic['expires_at'] : 0,
        'max_vans' => (int)$info['max_vans'],
        'trial_minutes_per_day' => $plan === 'trial' ? (int)$CFG['trial_minutes_per_day'] : 0,
        'trial_seconds_left_today' => $remaining,
        'device' => $dev['device_id'],
        'issued' => $now,
        'valid_until' => $now + (int)$CFG['offline_days'] * DAY, // works offline until then
    ];
    return ['ok' => true, 'licence' => sign_licence($payload), 'plan' => $plan, 'plan_name' => plan_name($plan),
            'active' => $active, 'expires' => $payload['expires'], 'account_url' => site_url('account.php')];
}

function device_from_token(string $token): ?array {
    if (strlen($token) < 40) return null;
    $d = one('SELECT * FROM devices WHERE token_hash = ?', [hash('sha256', $token)]);
    if (!$d || $d['revoked']) return null;
    return $d;
}
