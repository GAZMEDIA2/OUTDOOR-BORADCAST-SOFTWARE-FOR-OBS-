<?php
require __DIR__ . '/inc/bootstrap.php';
require __DIR__ . '/inc/layout.php';
if (current_user()) redirect('account.php');

$next = (string)($_GET['next'] ?? $_POST['next'] ?? 'account.php');
if (!preg_match('#^/?[A-Za-z0-9_\-./?=&%]*$#', $next) || str_contains($next, '//')) $next = 'account.php';

$err = '';
$email = '';
if ($_SERVER['REQUEST_METHOD'] === 'POST') {
    csrf_check();
    $email = strtolower(trim((string)($_POST['email'] ?? '')));
    $key = 'login:' . client_ip();
    if (too_many_attempts($key)) $err = 'Too many attempts. Wait 15 minutes and try again.';
    else {
        $u = one('SELECT * FROM users WHERE email = ?', [$email]);
        if ($u && password_verify((string)($_POST['password'] ?? ''), $u['pass_hash'])) {
            if ($u['blocked']) $err = 'This account is disabled. Contact ' . $CFG['support_email'] . '.';
            else {
                if (password_needs_rehash($u['pass_hash'], PASSWORD_DEFAULT)) {
                    q('UPDATE users SET pass_hash = ? WHERE id = ?', [password_hash($_POST['password'], PASSWORD_DEFAULT), $u['id']]);
                }
                login_user($u);
                redirect($next);
            }
        } else {
            note_failed_attempt($key);
            $err = 'Wrong email or password.';
        }
    }
}
page_head('Log in');
?>
<main class="narrow">
  <h1>Log in</h1>
  <?php if ($err): ?><p class="err-box"><?= e($err) ?></p><?php endif ?>
  <form method="post" class="card"><?= csrf_field() ?>
    <input type="hidden" name="next" value="<?= e($next) ?>">
    <label>Email<input name="email" type="email" value="<?= e($email) ?>" required autocomplete="email"></label>
    <label>Password<input name="password" type="password" required autocomplete="current-password"></label>
    <button class="btn">Log in</button>
  </form>
  <p class="muted">New to Van Link? <a href="signup.php">Start a free trial</a>.<br>Forgot your password? Contact <a href="mailto:<?= e($CFG['support_email']) ?>"><?= e($CFG['support_email']) ?></a>.</p>
</main>
<?php page_foot();
