<?php
// One-time setup: creates the database tables and your admin account.
// Open https://vanlink.gazgoc.com.ng/install.php once, then DELETE this file.
require __DIR__ . '/inc/bootstrap.php';

$driver = db()->getAttribute(PDO::ATTR_DRIVER_NAME);
$id = $driver === 'sqlite' ? 'INTEGER PRIMARY KEY AUTOINCREMENT' : 'INT UNSIGNED AUTO_INCREMENT PRIMARY KEY';
$opts = $driver === 'sqlite' ? '' : ' ENGINE=InnoDB DEFAULT CHARSET=utf8mb4';
$tables = [
    "CREATE TABLE IF NOT EXISTS users (id $id, email VARCHAR(190) NOT NULL UNIQUE, name VARCHAR(120) NOT NULL,
        company VARCHAR(160) NOT NULL DEFAULT '', phone VARCHAR(40) NOT NULL DEFAULT '', pass_hash VARCHAR(255) NOT NULL,
        is_admin TINYINT NOT NULL DEFAULT 0, blocked TINYINT NOT NULL DEFAULT 0, trial_used TINYINT NOT NULL DEFAULT 0,
        created_at INT NOT NULL)$opts",
    "CREATE TABLE IF NOT EXISTS licences (id $id, user_id INT NOT NULL, plan VARCHAR(20) NOT NULL, starts_at INT NOT NULL,
        expires_at INT NOT NULL, status VARCHAR(20) NOT NULL, note VARCHAR(255) NOT NULL DEFAULT '', created_at INT NOT NULL)$opts",
    "CREATE TABLE IF NOT EXISTS payments (id $id, user_id INT NOT NULL, plan VARCHAR(20) NOT NULL, amount_usd INT NOT NULL,
        method VARCHAR(20) NOT NULL, reference VARCHAR(100) NOT NULL UNIQUE, status VARCHAR(20) NOT NULL,
        note VARCHAR(500) NOT NULL DEFAULT '', created_at INT NOT NULL, paid_at INT NOT NULL DEFAULT 0)$opts",
    "CREATE TABLE IF NOT EXISTS devices (id $id, user_id INT NOT NULL, device_id VARCHAR(64) NOT NULL, name VARCHAR(120) NOT NULL DEFAULT '',
        role VARCHAR(10) NOT NULL DEFAULT '', token_hash CHAR(64) NOT NULL UNIQUE, created_at INT NOT NULL, last_seen INT NOT NULL,
        revoked TINYINT NOT NULL DEFAULT 0)$opts",
    "CREATE TABLE IF NOT EXISTS usage_log (id $id, user_id INT NOT NULL, day CHAR(10) NOT NULL, seconds INT NOT NULL DEFAULT 0)$opts",
    "CREATE TABLE IF NOT EXISTS reg_codes (id $id, code VARCHAR(24) NOT NULL UNIQUE, plan VARCHAR(20) NOT NULL, days INT NOT NULL,
        issued_to INT NOT NULL DEFAULT 0, payment_id INT NOT NULL DEFAULT 0, status VARCHAR(10) NOT NULL, note VARCHAR(255) NOT NULL DEFAULT '',
        created_at INT NOT NULL, used_at INT NOT NULL DEFAULT 0, used_by INT NOT NULL DEFAULT 0)$opts",
    "CREATE TABLE IF NOT EXISTS login_attempts (id $id, k VARCHAR(200) NOT NULL, at INT NOT NULL)$opts",
];
foreach ($tables as $sql) db()->exec($sql);
foreach (['CREATE INDEX lic_user ON licences (user_id)', 'CREATE INDEX dev_user ON devices (user_id)',
          'CREATE UNIQUE INDEX usage_day ON usage_log (user_id, day)', 'CREATE INDEX att_k ON login_attempts (k)'] as $ix) {
    try { db()->exec($ix); } catch (PDOException $e) { /* already exists */ }
}

$hasAdmin = (bool)one('SELECT id FROM users WHERE is_admin = 1 LIMIT 1');
$msg = '';
if (!$hasAdmin && $_SERVER['REQUEST_METHOD'] === 'POST') {
    csrf_check();
    $email = strtolower(trim($_POST['email'] ?? ''));
    $pass = (string)($_POST['password'] ?? '');
    if (!valid_email($email)) $msg = 'Enter a valid email address.';
    elseif (!is_owner_email($email)) $msg = 'The admin account must use the owner email set in config.php (admin_emails).';
    elseif (strlen($pass) < 10) $msg = 'Use a password of at least 10 characters.';
    else {
        q('INSERT INTO users (email, name, pass_hash, is_admin, created_at) VALUES (?, ?, ?, 1, ?)',
            [$email, trim($_POST['name'] ?? 'Admin') ?: 'Admin', password_hash($pass, PASSWORD_DEFAULT), time()]);
        $hasAdmin = true;
    }
}
?><!doctype html><html lang="en"><head><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1">
<title>Van Link setup</title><link rel="stylesheet" href="assets/style.css"></head><body><main class="narrow">
<h1>Van Link website setup</h1>
<p class="ok-box">Database tables are ready.</p>
<?php if ($hasAdmin): ?>
  <p class="ok-box">Your admin account exists. <b>Now delete install.php from the server</b> (cPanel › File Manager), then <a href="login.php">log in</a>.</p>
<?php else: ?>
  <h2>Create your admin account</h2>
  <?php if ($msg): ?><p class="err-box"><?= e($msg) ?></p><?php endif ?>
  <form method="post" class="card"><?= csrf_field() ?>
    <label>Your name<input name="name" required></label>
    <label>Email<input name="email" type="email" required value="<?= e($CFG['admin_emails'][0] ?? '') ?>"></label>
    <label>Password (at least 10 characters)<input name="password" type="password" minlength="10" required></label>
    <button class="btn">Create admin account</button>
  </form>
<?php endif ?>
</main></body></html>
