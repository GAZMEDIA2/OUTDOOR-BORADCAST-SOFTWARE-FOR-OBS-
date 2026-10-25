<?php
require __DIR__ . '/inc/bootstrap.php';
require __DIR__ . '/inc/layout.php';
if (current_user()) redirect('account.php');

$err = '';
$v = ['name' => '', 'email' => '', 'company' => '', 'phone' => ''];
if ($_SERVER['REQUEST_METHOD'] === 'POST') {
    csrf_check();
    foreach ($v as $k => $_) $v[$k] = trim((string)($_POST[$k] ?? ''));
    $v['email'] = strtolower($v['email']);
    $pass = (string)($_POST['password'] ?? '');
    if (too_many_attempts('signup:' . client_ip())) $err = 'Too many attempts. Wait 15 minutes and try again.';
    elseif ($v['name'] === '' || strlen($v['name']) > 120) $err = 'Enter your name.';
    elseif (!valid_email($v['email'])) $err = 'Enter a valid email address.';
    elseif (strlen($pass) < 8) $err = 'Use a password of at least 8 characters.';
    elseif (is_owner_email($v['email']) && !one('SELECT id FROM users WHERE is_admin = 1')) $err = 'This email is reserved for the site owner. Use install.php to set it up.';
    elseif (one('SELECT id FROM users WHERE email = ?', [$v['email']])) $err = 'An account with this email already exists. Log in instead.';
    else {
        note_failed_attempt('signup:' . client_ip()); // counts sign-ups too, to stop mass sign-ups
        q('INSERT INTO users (email, name, company, phone, pass_hash, created_at) VALUES (?, ?, ?, ?, ?, ?)',
            [$v['email'], $v['name'], substr($v['company'], 0, 160), substr($v['phone'], 0, 40), password_hash($pass, PASSWORD_DEFAULT), time()]);
        $u = one('SELECT * FROM users WHERE email = ?', [$v['email']]);
        start_trial((int)$u['id']); // admin rights are only ever given by install.php
        login_user($u);
        flash('Welcome! Your 14-day free trial has started. Download Van Link below.');
        redirect('account.php');
    }
}
page_head('Start your free trial');
?>
<main class="narrow">
  <h1>Start your free trial</h1>
  <p class="muted"><?= (int)$CFG['trial_days'] ?> days, <?= (int)$CFG['trial_minutes_per_day'] ?> minutes of broadcasting per 24 hours. No payment needed.</p>
  <?php if ($err): ?><p class="err-box"><?= e($err) ?></p><?php endif ?>
  <form method="post" class="card"><?= csrf_field() ?>
    <label>Your name<input name="name" value="<?= e($v['name']) ?>" required maxlength="120" autocomplete="name"></label>
    <label>Email<input name="email" type="email" value="<?= e($v['email']) ?>" required autocomplete="email"></label>
    <label>Company or station (optional)<input name="company" value="<?= e($v['company']) ?>" maxlength="160" autocomplete="organization"></label>
    <label>Phone / WhatsApp (optional)<input name="phone" value="<?= e($v['phone']) ?>" maxlength="40" autocomplete="tel"></label>
    <label>Password (at least 8 characters)<input name="password" type="password" minlength="8" required autocomplete="new-password"></label>
    <button class="btn">Create account and start trial</button>
  </form>
  <p class="muted">Already have an account? <a href="login.php">Log in</a></p>
</main>
<?php page_foot();
