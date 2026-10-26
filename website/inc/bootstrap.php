<?php
// Van Link website - shared setup for every page.
declare(strict_types=1);

if (!is_file(__DIR__ . '/../config.php')) {
    http_response_code(500);
    exit('Van Link website is not configured yet: copy config.sample.php to config.php and fill it in.');
}
$CFG = require __DIR__ . '/../config.php';
date_default_timezone_set($CFG['timezone'] ?? 'Africa/Lagos');

// ---------- database ----------
function db(): PDO {
    static $pdo = null;
    global $CFG;
    if ($pdo === null) {
        $pdo = new PDO($CFG['db_dsn'], $CFG['db_user'] ?? null, $CFG['db_pass'] ?? null, [
            PDO::ATTR_ERRMODE => PDO::ERRMODE_EXCEPTION,
            PDO::ATTR_DEFAULT_FETCH_MODE => PDO::FETCH_ASSOC,
            PDO::ATTR_EMULATE_PREPARES => false,
        ]);
    }
    return $pdo;
}
function q(string $sql, array $args = []): PDOStatement {
    $st = db()->prepare($sql);
    $st->execute($args);
    return $st;
}
function one(string $sql, array $args = []): ?array {
    $r = q($sql, $args)->fetch();
    return $r === false ? null : $r;
}

// ---------- sessions & security ----------
function start_session(): void {
    if (session_status() === PHP_SESSION_ACTIVE) return;
    $secure = (!empty($_SERVER['HTTPS']) && $_SERVER['HTTPS'] !== 'off');
    session_set_cookie_params(['lifetime' => 0, 'path' => '/', 'secure' => $secure, 'httponly' => true, 'samesite' => 'Lax']);
    session_name('vanlink');
    session_start();
}
function e($s): string { return htmlspecialchars((string)$s, ENT_QUOTES, 'UTF-8'); }
function csrf_token(): string {
    start_session();
    if (empty($_SESSION['csrf'])) $_SESSION['csrf'] = bin2hex(random_bytes(16));
    return $_SESSION['csrf'];
}
function csrf_field(): string { return '<input type="hidden" name="csrf" value="' . e(csrf_token()) . '">'; }
function csrf_check(): void {
    start_session();
    if (!hash_equals($_SESSION['csrf'] ?? '', (string)($_POST['csrf'] ?? ''))) {
        http_response_code(400);
        exit('This form has expired. Go back, refresh the page and try again.');
    }
}
function redirect(string $to): never { header('Location: ' . $to); exit; }
function flash(?string $msg = null, string $kind = 'ok'): ?array {
    start_session();
    if ($msg !== null) { $_SESSION['flash'] = [$kind, $msg]; return null; }
    $f = $_SESSION['flash'] ?? null; unset($_SESSION['flash']); return $f;
}

// Slow down password guessing: max 8 failed logins per 15 minutes per address.
function too_many_attempts(string $key): bool {
    $since = time() - 900;
    q('DELETE FROM login_attempts WHERE at < ?', [$since]);
    return (int)one('SELECT COUNT(*) n FROM login_attempts WHERE k = ? AND at >= ?', [$key, $since])['n'] >= 8;
}
function note_failed_attempt(string $key): void { q('INSERT INTO login_attempts (k, at) VALUES (?, ?)', [$key, time()]); }
function client_ip(): string { return $_SERVER['REMOTE_ADDR'] ?? '0.0.0.0'; }

// ---------- users ----------
function current_user(): ?array {
    start_session();
    if (empty($_SESSION['uid'])) return null;
    $u = one('SELECT * FROM users WHERE id = ?', [$_SESSION['uid']]);
    if (!$u || $u['blocked']) { unset($_SESSION['uid']); return null; }
    return $u;
}
function require_login(): array {
    $u = current_user();
    if (!$u) redirect('login.php?next=' . urlencode($_SERVER['REQUEST_URI'] ?? 'account.php'));
    return $u;
}
function require_admin(): array {
    $u = require_login();
    if (!$u['is_admin']) { http_response_code(403); exit('Not allowed.'); }
    return $u;
}
function login_user(array $u): void {
    start_session();
    session_regenerate_id(true);
    $_SESSION['uid'] = (int)$u['id'];
}
function valid_email(string $e): bool { return (bool)filter_var($e, FILTER_VALIDATE_EMAIL) && strlen($e) <= 190; }

// ---------- licences ----------
const DAY = 86400;

function plan_info(string $plan): array {
    global $CFG;
    $p = $CFG['plans'][$plan] ?? null;
    if (!$p) throw new RuntimeException('unknown plan');
    return $p;
}
function current_licence(int $uid): ?array {
    return one('SELECT * FROM licences WHERE user_id = ? ORDER BY expires_at DESC, id DESC LIMIT 1', [$uid]);
}
function licence_active(?array $l): bool {
    return $l && $l['status'] === 'active' && (int)$l['expires_at'] > time();
}
function start_trial(int $uid): void {
    global $CFG;
    $now = time();
    q('INSERT INTO licences (user_id, plan, starts_at, expires_at, status, created_at) VALUES (?, ?, ?, ?, ?, ?)',
        [$uid, 'trial', $now, $now + (int)$CFG['trial_days'] * DAY, 'active', $now]);
    q('UPDATE users SET trial_used = 1 WHERE id = ?', [$uid]);
}
// Grant (or renew) a paid plan for one year. Renewing the same plan adds a
// year to the current end date; a different plan starts a fresh year now.
function grant_plan(int $uid, string $plan, string $why, int $days = 365): void {
    plan_info($plan);
    $now = time();
    $cur = current_licence($uid);
    $start = $now;
    if ($cur && $cur['plan'] === $plan && licence_active($cur)) $start = (int)$cur['expires_at'];
    q('UPDATE licences SET status = ? WHERE user_id = ? AND status = ? AND plan = ?', ['replaced', $uid, 'active', 'trial']);
    if ($cur && $cur['plan'] !== $plan && $cur['plan'] !== 'trial' && licence_active($cur)) {
        q('UPDATE licences SET status = ? WHERE id = ?', ['replaced', $cur['id']]);
    }
    q('INSERT INTO licences (user_id, plan, starts_at, expires_at, status, note, created_at) VALUES (?, ?, ?, ?, ?, ?, ?)',
        [$uid, $plan, $now, $start + $days * DAY, 'active', $why, $now]);
}
function plan_name(string $p): string {
    return ['trial' => 'Free trial', 'standard' => 'Standard', 'pro' => 'Pro', 'owner' => 'Owner (unlimited)'][$p] ?? $p;
}
function is_owner_email(string $email): bool {
    global $CFG;
    return in_array(strtolower($email), array_map('strtolower', $CFG['admin_emails'] ?? []), true);
}
// The licence the app gets: owners/admins always get the unlimited Owner licence.
function effective_licence(array $u): ?array {
    if ($u['is_admin']) {
        return ['plan' => 'owner', 'status' => 'active', 'starts_at' => (int)$u['created_at'], 'expires_at' => 4102444800];
    }
    return current_licence((int)$u['id']);
}
function fmt_date($ts): string { return $ts ? date('j M Y', (int)$ts) : '-'; }
function money(float $usd): string { return '$' . number_format($usd, 0); }

// Signs what the app is allowed to do. The app checks this signature with the
// public key built into it, so licences cannot be faked or edited.
function sign_licence(array $payload): string {
    global $CFG;
    $json = json_encode($payload, JSON_UNESCAPED_SLASHES);
    $sk = base64_decode($CFG['licence_secret_key'], true);
    if ($sk === false || strlen($sk) !== SODIUM_CRYPTO_SIGN_SECRETKEYBYTES) throw new RuntimeException('licence key not configured');
    $sig = sodium_crypto_sign_detached($json, $sk);
    return rtrim(strtr(base64_encode($json), '+/', '-_'), '=') . '.' . rtrim(strtr(base64_encode($sig), '+/', '-_'), '=');
}

function site_url(string $path = ''): string {
    global $CFG;
    return rtrim($CFG['site_url'], '/') . '/' . ltrim($path, '/');
}

// ---------- JSON API helpers (for the app) ----------
function api_out(array $data, int $code = 200): never {
    http_response_code($code);
    header('Content-Type: application/json');
    header('Cache-Control: no-store');
    echo json_encode($data, JSON_UNESCAPED_SLASHES);
    exit;
}
function api_in(): array {
    $raw = file_get_contents('php://input', false, null, 0, 65536);
    $j = json_decode($raw ?: '[]', true);
    return is_array($j) ? $j : [];
}

// ---------- registration codes ----------
// A paid year comes as a registration code; entering it (in the app or on the
// website) activates the licence on that account.
function new_reg_code(string $plan, int $days, int $issuedTo = 0, int $paymentId = 0, string $note = ''): string {
    plan_info($plan);
    $alpha = 'ABCDEFGHJKLMNPQRSTUVWXYZ23456789';
    do {
        $c = 'VLK';
        for ($g = 0; $g < 3; $g++) {
            $c .= '-';
            for ($i = 0; $i < 4; $i++) $c .= $alpha[random_int(0, strlen($alpha) - 1)];
        }
    } while (one('SELECT id FROM reg_codes WHERE code = ?', [$c]));
    q('INSERT INTO reg_codes (code, plan, days, issued_to, payment_id, status, note, created_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?)',
        [$c, $plan, $days, $issuedTo, $paymentId, 'unused', $note, time()]);
    return $c;
}
function normalize_code(string $c): string {
    $c = strtoupper(preg_replace('/[^A-Za-z0-9]/', '', $c));
    if (strlen($c) === 15 && str_starts_with($c, 'VLK')) return 'VLK-' . substr($c, 3, 4) . '-' . substr($c, 7, 4) . '-' . substr($c, 11, 4);
    return '';
}
// Returns an error message, or '' when the code was activated.
function redeem_code(int $uid, string $raw): string {
    $code = normalize_code($raw);
    if ($code === '') return 'That is not a valid registration code. It looks like VLK-XXXX-XXXX-XXXX.';
    $r = one('SELECT * FROM reg_codes WHERE code = ?', [$code]);
    if (!$r) return 'Registration code not found. Check it and try again.';
    if ($r['status'] === 'used') return 'This registration code has already been used.';
    if ($r['status'] !== 'unused') return 'This registration code is no longer valid.';
    if ((int)$r['issued_to'] && (int)$r['issued_to'] !== $uid) return 'This registration code belongs to a different account.';
    $won = q('UPDATE reg_codes SET status = ?, used_at = ?, used_by = ? WHERE id = ? AND status = ?', ['used', time(), $uid, $r['id'], 'unused'])->rowCount();
    if (!$won) return 'This registration code has already been used.';
    grant_plan($uid, $r['plan'], 'Registration code ' . $code, (int)$r['days']);
    return '';
}
function send_code_email(array $u, string $code, string $plan): void {
    global $CFG;
    $host = parse_url($CFG['site_url'], PHP_URL_HOST) ?: 'localhost';
    $msg = "Hello {$u['name']},\r\n\r\nThank you for buying Van Link " . plan_name($plan) . ".\r\n\r\n"
        . "Your registration code is:\r\n\r\n    $code\r\n\r\n"
        . "To activate it, open Van Link, sign in with this email address and enter the code under \"Registration code\".\r\n"
        . "You can also activate it on your account page: " . site_url('account.php') . "\r\n\r\n"
        . "Keep this code safe.\r\n\r\nVan Link";
    @mail($u['email'], 'Your Van Link registration code', $msg,
        "From: Van Link <noreply@$host>\r\nReply-To: {$CFG['support_email']}\r\nContent-Type: text/plain; charset=UTF-8");
}
// Issue a code for a confirmed payment (once).
function issue_code_for_payment(array $p): string {
    $have = one('SELECT code FROM reg_codes WHERE payment_id = ?', [$p['id']]);
    if ($have) return $have['code'];
    $code = new_reg_code($p['plan'], 365, (int)$p['user_id'], (int)$p['id'], 'Payment ' . $p['reference']);
    $u = one('SELECT * FROM users WHERE id = ?', [$p['user_id']]);
    if ($u) send_code_email($u, $code, $p['plan']);
    return $code;
}

// ---------- settings changed from the admin page ----------
function setting(string $k, ?string $default = null): ?string {
    try { $r = one('SELECT v FROM settings WHERE k = ?', [$k]); } catch (Throwable $e) { return $default; }
    return $r ? $r['v'] : $default;
}
function set_setting(string $k, string $v): void {
    if (one('SELECT k FROM settings WHERE k = ?', [$k])) q('UPDATE settings SET v = ? WHERE k = ?', [$v, $k]);
    else q('INSERT INTO settings (k, v) VALUES (?, ?)', [$k, $v]);
}
// Naira (Paystack currency) per US dollar: admin setting, else config.php.
function usd_rate(): float {
    global $CFG;
    $r = (float)setting('usd_rate', (string)($CFG['usd_rate'] ?? 1600));
    return $r > 0 ? $r : 1600.0;
}
