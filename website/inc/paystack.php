<?php
// Paystack: start a payment and confirm it. No webhook is used, so the
// account's existing webhook (for other websites) is left untouched.

function paystack_enabled(): bool {
    global $CFG;
    return str_starts_with((string)($CFG['paystack_secret_key'] ?? ''), 'sk_');
}

function paystack_call(string $method, string $path, ?array $body = null): array {
    global $CFG;
    $ch = curl_init('https://api.paystack.co' . $path);
    $headers = ['Authorization: Bearer ' . $CFG['paystack_secret_key'], 'Content-Type: application/json'];
    curl_setopt_array($ch, [CURLOPT_RETURNTRANSFER => true, CURLOPT_TIMEOUT => 25, CURLOPT_HTTPHEADER => $headers,
        CURLOPT_CUSTOMREQUEST => $method]);
    if ($body !== null) curl_setopt($ch, CURLOPT_POSTFIELDS, json_encode($body));
    $raw = curl_exec($ch);
    $err = curl_error($ch);
    curl_close($ch);
    if ($raw === false) throw new RuntimeException('Could not reach Paystack: ' . $err);
    $j = json_decode($raw, true);
    if (!is_array($j)) throw new RuntimeException('Unexpected reply from Paystack');
    return $j;
}

// Amount charged for a plan, in the Paystack currency's smallest unit (kobo).
function paystack_amount(string $plan): int {
    global $CFG;
    $usd = (int)plan_info($plan)['price'];
    $rate = strtoupper($CFG['paystack_currency']) === 'USD' ? 1 : (float)$CFG['usd_rate'];
    return (int)round($usd * $rate * 100);
}

function paystack_start(array $user, string $plan): string {
    global $CFG;
    $ref = 'VL-' . $user['id'] . '-' . bin2hex(random_bytes(6));
    q('INSERT INTO payments (user_id, plan, amount_usd, method, reference, status, created_at) VALUES (?, ?, ?, ?, ?, ?, ?)',
        [$user['id'], $plan, (int)plan_info($plan)['price'], 'paystack', $ref, 'pending', time()]);
    $r = paystack_call('POST', '/transaction/initialize', [
        'email' => $user['email'],
        'amount' => paystack_amount($plan),
        'currency' => strtoupper($CFG['paystack_currency']),
        'reference' => $ref,
        'callback_url' => site_url('paystack-callback.php'),
        'metadata' => ['product' => 'Van Link', 'plan' => $plan, 'user_id' => $user['id'],
            'custom_fields' => [['display_name' => 'Product', 'variable_name' => 'product', 'value' => 'Van Link ' . plan_name($plan) . ' (1 year)']]],
    ]);
    if (empty($r['status']) || empty($r['data']['authorization_url'])) {
        throw new RuntimeException('Paystack refused the payment: ' . ($r['message'] ?? 'unknown error'));
    }
    return $r['data']['authorization_url'];
}

// Confirms a payment with Paystack and activates the licence once.
// Returns 'paid', 'pending' or 'failed'.
function paystack_confirm(string $ref): string {
    global $CFG;
    $p = one('SELECT * FROM payments WHERE reference = ? AND method = ?', [$ref, 'paystack']);
    if (!$p) return 'failed';
    if ($p['status'] === 'paid') return 'paid';
    $r = paystack_call('GET', '/transaction/verify/' . rawurlencode($ref));
    $d = $r['data'] ?? [];
    $st = $d['status'] ?? '';
    if ($st === 'success') {
        $okAmount = (int)($d['amount'] ?? 0) >= paystack_amount($p['plan']);
        $okCur = strtoupper((string)($d['currency'] ?? '')) === strtoupper($CFG['paystack_currency']);
        if (!$okAmount || !$okCur) {
            q('UPDATE payments SET status = ?, note = ? WHERE id = ?', ['review', 'Amount or currency did not match; check in Paystack', $p['id']]);
            return 'failed';
        }
        // Mark paid first (only one request can win), then grant the plan.
        $won = q('UPDATE payments SET status = ?, paid_at = ? WHERE id = ? AND status <> ?', ['paid', time(), $p['id'], 'paid'])->rowCount();
        if ($won) issue_code_for_payment($p);
        return 'paid';
    }
    if (in_array($st, ['failed', 'abandoned', 'reversed'], true)) {
        q('UPDATE payments SET status = ? WHERE id = ? AND status = ?', ['failed', $p['id'], 'pending']);
        return 'failed';
    }
    return 'pending';
}
