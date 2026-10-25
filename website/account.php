<?php
require __DIR__ . '/inc/bootstrap.php';
require __DIR__ . '/inc/layout.php';
require __DIR__ . '/inc/paystack.php';
$user = require_login();
$uid = (int)$user['id'];

if ($_SERVER['REQUEST_METHOD'] === 'POST') {
    csrf_check();
    $act = (string)($_POST['act'] ?? '');
    if ($act === 'check') {
        $n = 0;
        foreach (q('SELECT reference FROM payments WHERE user_id = ? AND method = ? AND status = ?', [$uid, 'paystack', 'pending'])->fetchAll() as $p) {
            try { if (paystack_confirm($p['reference']) === 'paid') $n++; } catch (Throwable $e) {}
        }
        flash($n ? 'Payment confirmed. Your licence is active.' : 'No completed payment found yet. If you paid, wait a minute and check again.', $n ? 'ok' : 'err');
    } elseif ($act === 'redeem') {
        $err = redeem_code($uid, (string)($_POST['code'] ?? ''));
        flash($err === '' ? 'Registration code accepted. Your licence is active. The Van Link app picks it up automatically.' : $err, $err === '' ? 'ok' : 'err');
    } elseif ($act === 'signout') {
        q('UPDATE devices SET revoked = 1 WHERE id = ? AND user_id = ?', [(int)($_POST['id'] ?? 0), $uid]);
        flash('That computer has been signed out of Van Link.');
    } elseif ($act === 'password') {
        $new = (string)($_POST['new'] ?? '');
        if (!password_verify((string)($_POST['old'] ?? ''), $user['pass_hash'])) flash('Your current password is wrong.', 'err');
        elseif (strlen($new) < 8) flash('Use a new password of at least 8 characters.', 'err');
        else { q('UPDATE users SET pass_hash = ? WHERE id = ?', [password_hash($new, PASSWORD_DEFAULT), $uid]); flash('Password changed.'); }
    }
    redirect('account.php');
}

$lic = effective_licence($user);
$active = licence_active($lic);
$plan = $lic['plan'] ?? '';
$devices = q('SELECT * FROM devices WHERE user_id = ? AND revoked = 0 ORDER BY last_seen DESC', [$uid])->fetchAll();
$payments = q('SELECT * FROM payments WHERE user_id = ? ORDER BY id DESC LIMIT 20', [$uid])->fetchAll();
$pendingPaystack = (bool)array_filter($payments, fn($p) => $p['method'] === 'paystack' && $p['status'] === 'pending');
$used = 0;
if ($plan === 'trial') $used = (int)(one('SELECT seconds FROM usage_log WHERE user_id = ? AND day = ?', [$uid, date('Y-m-d')])['seconds'] ?? 0);
$codes = q('SELECT * FROM reg_codes WHERE issued_to = ? ORDER BY id DESC', [$uid])->fetchAll();
$files = glob(__DIR__ . '/downloads/*.{exe,dmg,zip}', GLOB_BRACE) ?: [];
rsort($files);
page_head('My account', $user);
?>
<main>
  <h1>Hello, <?= e($user['name']) ?></h1>

  <div class="grid">
    <div class="card">
      <h3>Your licence</h3>
      <?php if ($active): ?>
        <p><span class="pill on">Active</span> <b><?= e(plan_name($plan)) ?></b></p>
        <?php if ($plan !== 'owner'): ?><p>Valid until <b><?= fmt_date($lic['expires_at']) ?></b></p><?php endif ?>
        <?php if ($plan === 'trial'): ?><p>Broadcast time used today: <?= intdiv($used, 60) ?> of <?= (int)$CFG['trial_minutes_per_day'] ?> minutes</p><?php endif ?>
      <?php elseif ($lic): ?>
        <p><span class="pill off">Expired</span> <?= e(plan_name($plan)) ?> ended <?= fmt_date($lic['expires_at']) ?></p>
      <?php else: ?>
        <p><span class="pill">No licence</span></p>
      <?php endif ?>
      <?php if ($plan !== 'owner'): ?>
      <div class="row" style="margin-top:12px">
        <a class="btn small" href="buy.php?plan=standard"><?= $plan === 'standard' && $active ? 'Renew' : 'Buy' ?> Standard</a>
        <a class="btn small ghost" href="buy.php?plan=pro"><?= $plan === 'pro' && $active ? 'Renew' : 'Buy' ?> Pro</a>
      </div>
      <?php endif ?>
    </div>

    <div class="card">
      <h3>Download Van Link</h3>
      <?php if (!$files): ?><p>Downloads will appear here soon.</p><?php endif ?>
      <?php foreach ($files as $f): $n = basename($f); ?>
        <p><a class="btn small" style="width:100%" href="download.php?f=<?= urlencode($n) ?>"><?= e($n) ?></a></p>
      <?php endforeach ?>
      <p class="muted" style="font-size:14px">Install on both the studio and the van computer, then sign in with this email and password.</p>
    </div>
  </div>

  <section>
    <h2>Registration codes</h2>
    <?php foreach ($codes as $c): if ($c['status'] !== 'unused') continue; ?>
      <div class="ok-box" style="margin:0 0 12px">Your code for <?= e(plan_name($c['plan'])) ?> (<?= (int)$c['days'] ?> days): <code style="font-size:18px"><?= e($c['code']) ?></code>
        <form method="post" class="inline"><?= csrf_field() ?><input type="hidden" name="act" value="redeem"><input type="hidden" name="code" value="<?= e($c['code']) ?>"><button class="btn small">Activate now</button></form></div>
    <?php endforeach ?>
    <form method="post" class="row"><?= csrf_field() ?><input type="hidden" name="act" value="redeem">
      <input name="code" placeholder="VLK-XXXX-XXXX-XXXX" style="max-width:280px;margin:0" required><button class="btn small">Activate code</button></form>
    <?php $usedC = array_filter($codes, fn($c) => $c['status'] === 'used'); if ($usedC): ?>
      <p class="muted" style="margin-top:10px">Used: <?php foreach ($usedC as $c) echo e($c['code']) . ' (' . e(plan_name($c['plan'])) . ', ' . fmt_date($c['used_at']) . ') '; ?></p>
    <?php endif ?>
  </section>

  <section>
    <h2>Computers signed in</h2>
    <?php $maxC = (int)plan_info($plan ?: 'trial')['max_computers']; ?>
    <p class="muted"><?= count($devices) ?> of <?= $maxC ?> allowed on your plan.</p>
    <?php if (!$devices): ?><p>None yet. Sign in to the Van Link app on your computers.</p><?php else: ?>
    <div class="scroll"><table><tr><th>Computer</th><th>Role</th><th>Last seen</th><th></th></tr>
    <?php foreach ($devices as $d): ?>
      <tr><td><?= e($d['name'] ?: 'Computer') ?></td><td><?= e(ucfirst($d['role'] ?: '-')) ?></td><td><?= date('j M Y H:i', (int)$d['last_seen']) ?></td>
      <td><form method="post" class="inline"><?= csrf_field() ?><input type="hidden" name="act" value="signout"><input type="hidden" name="id" value="<?= (int)$d['id'] ?>"><button class="btn small ghost">Sign out</button></form></td></tr>
    <?php endforeach ?></table></div><?php endif ?>
  </section>

  <section>
    <h2>Payments</h2>
    <?php if ($pendingPaystack): ?>
      <form method="post"><?= csrf_field() ?><input type="hidden" name="act" value="check"><button class="btn small">Check my payment</button></form>
    <?php endif ?>
    <?php if (!$payments): ?><p class="muted">No payments yet.</p><?php else: ?>
    <div class="scroll" style="margin-top:12px"><table><tr><th>Date</th><th>Plan</th><th>Amount</th><th>Method</th><th>Status</th></tr>
    <?php foreach ($payments as $p): ?>
      <tr><td><?= fmt_date($p['created_at']) ?></td><td><?= e(plan_name($p['plan'])) ?></td><td><?= money((float)$p['amount_usd']) ?></td>
      <td><?= e($p['method'] === 'paystack' ? 'Paystack' : 'Mobile money / bank') ?></td>
      <td><span class="pill <?= $p['status'] === 'paid' ? 'on' : ($p['status'] === 'pending' ? 'wait' : 'off') ?>"><?= e(['paid' => 'Paid', 'pending' => 'Waiting', 'failed' => 'Failed', 'rejected' => 'Rejected', 'review' => 'Being checked'][$p['status']] ?? $p['status']) ?></span></td></tr>
    <?php endforeach ?></table></div><?php endif ?>
  </section>

  <section>
    <h2>Change password</h2>
    <form method="post" class="card" style="max-width:420px"><?= csrf_field() ?><input type="hidden" name="act" value="password">
      <label>Current password<input type="password" name="old" required autocomplete="current-password"></label>
      <label>New password (at least 8 characters)<input type="password" name="new" minlength="8" required autocomplete="new-password"></label>
      <button class="btn">Change password</button>
    </form>
  </section>
</main>
<?php page_foot();
