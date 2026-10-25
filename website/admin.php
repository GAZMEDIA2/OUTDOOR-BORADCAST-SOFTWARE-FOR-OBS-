<?php
require __DIR__ . '/inc/bootstrap.php';
require __DIR__ . '/inc/layout.php';
require __DIR__ . '/inc/paystack.php';
$me = require_admin();

$temp = null; // a reset password to show once
if ($_SERVER['REQUEST_METHOD'] === 'POST') {
    csrf_check();
    $act = (string)($_POST['act'] ?? '');
    $id = (int)($_POST['id'] ?? 0);
    $back = (string)($_POST['back'] ?? 'admin.php');
    if (!preg_match('#^admin\.php(\?[A-Za-z0-9_=&%-]*)?$#', $back)) $back = 'admin.php';
    switch ($act) {
        case 'approve':
            $p = one('SELECT * FROM payments WHERE id = ?', [$id]);
            if ($p && in_array($p['status'], ['pending', 'review'], true)) {
                q('UPDATE payments SET status = ?, paid_at = ? WHERE id = ?', ['paid', time(), $id]);
                $code = issue_code_for_payment($p);
                flash('Payment approved. Registration code ' . $code . ' was emailed to the customer and shows in their account.');
            }
            break;
        case 'reject':
            q('UPDATE payments SET status = ? WHERE id = ? AND status IN (?, ?)', ['rejected', $id, 'pending', 'review']);
            flash('Payment marked as rejected.');
            break;
        case 'verify':
            $p = one('SELECT * FROM payments WHERE id = ?', [$id]);
            try { $r = $p ? paystack_confirm($p['reference']) : 'failed'; flash('Paystack says: ' . $r . '.', $r === 'paid' ? 'ok' : 'err'); }
            catch (Throwable $e) { flash('Could not reach Paystack: ' . $e->getMessage(), 'err'); }
            break;
        case 'setplan':
            $plan = (string)($_POST['plan'] ?? '');
            $until = strtotime((string)($_POST['until'] ?? '') . ' 23:59:59');
            if (in_array($plan, ['trial', 'standard', 'pro'], true) && $until && $until > time()) {
                q('UPDATE licences SET status = ? WHERE user_id = ? AND status = ?', ['replaced', $id, 'active']);
                q('INSERT INTO licences (user_id, plan, starts_at, expires_at, status, note, created_at) VALUES (?, ?, ?, ?, ?, ?, ?)',
                    [$id, $plan, time(), $until, 'active', 'Set by admin ' . $me['email'], time()]);
                flash('Licence set to ' . plan_name($plan) . ' until ' . fmt_date($until) . '.');
            } else flash('Choose a plan and a date in the future.', 'err');
            break;
        case 'endlicence':
            q('UPDATE licences SET status = ? WHERE user_id = ? AND status = ?', ['cancelled', $id, 'active']);
            flash('Licence ended.');
            break;
        case 'block': case 'unblock':
            if ($id !== (int)$me['id']) {
                q('UPDATE users SET blocked = ? WHERE id = ?', [$act === 'block' ? 1 : 0, $id]);
                if ($act === 'block') q('UPDATE devices SET revoked = 1 WHERE user_id = ?', [$id]);
                flash($act === 'block' ? 'Account blocked and its computers signed out.' : 'Account unblocked.');
            }
            break;
        case 'resetpw':
            $temp = substr(strtr(base64_encode(random_bytes(12)), '+/=', 'xyz'), 0, 14);
            q('UPDATE users SET pass_hash = ? WHERE id = ?', [password_hash($temp, PASSWORD_DEFAULT), $id]);
            break;
        case 'rate':
            $r = (float)str_replace(',', '', (string)($_POST['rate'] ?? ''));
            if ($r >= 1 && $r <= 100000) { set_setting('usd_rate', (string)$r); flash('Exchange rate set: 1 USD = ' . number_format($r, 2) . ' ' . strtoupper($CFG['paystack_currency']) . '.'); }
            else flash('Enter a valid rate.', 'err');
            break;
        case 'newcode':
            $plan = (string)($_POST['plan'] ?? '');
            $days = max(1, min(3650, (int)($_POST['days'] ?? 365)));
            $to = 0;
            $email = strtolower(trim((string)($_POST['email'] ?? '')));
            if ($email !== '') {
                $tu = one('SELECT id FROM users WHERE email = ?', [$email]);
                if (!$tu) { flash('No customer with that email. Leave it empty for a code anyone can use.', 'err'); break; }
                $to = (int)$tu['id'];
            }
            if (!in_array($plan, ['standard', 'pro', 'trial'], true)) { flash('Choose a plan.', 'err'); break; }
            $code = new_reg_code($plan, $days, $to, 0, substr('Made by admin. ' . trim((string)($_POST['note'] ?? '')), 0, 255));
            flash('New registration code: ' . $code . ' (' . plan_name($plan) . ', ' . $days . ' days). Copy it and send it to the customer.');
            break;
        case 'voidcode':
            q('UPDATE reg_codes SET status = ? WHERE id = ? AND status = ?', ['void', $id, 'unused']);
            flash('Code cancelled.');
            break;
        case 'revoke':
            q('UPDATE devices SET revoked = 1 WHERE id = ?', [$id]);
            flash('Computer signed out.');
            break;
    }
    if ($temp === null) redirect($back);
}

$tab = (string)($_GET['tab'] ?? 'overview');
$uid = (int)($_GET['user'] ?? 0);
page_head('Admin', $me);
$now = time();

function status_pill(?array $l): string {
    if (!$l) return '<span class="pill">No licence</span>';
    if (licence_active($l)) return '<span class="pill on">' . e(plan_name($l['plan'])) . '</span>';
    return '<span class="pill off">' . e(plan_name($l['plan'])) . ' ended</span>';
}
?>
<main>
  <h1>Admin</h1>
  <div class="row" style="margin-bottom:20px">
    <?php foreach (['overview' => 'Overview', 'users' => 'Customers', 'payments' => 'Payments', 'codes' => 'Registration codes', 'devices' => 'Computers'] as $k => $v): ?>
      <a class="btn small <?= $tab === $k && !$uid ? '' : 'ghost' ?>" href="admin.php?tab=<?= $k ?>"><?= $v ?></a>
    <?php endforeach ?>
  </div>

<?php if ($temp !== null): $tu = one('SELECT email FROM users WHERE id = ?', [(int)$_POST['id']]); ?>
  <div class="ok-box">New temporary password for <b><?= e($tu['email'] ?? '') ?></b>: <code style="font-size:18px"><?= e($temp) ?></code><br>
  Copy it now and send it to the customer. It won't be shown again. They can change it on their account page.</div>
  <p><a href="admin.php?user=<?= (int)$_POST['id'] ?>">Back to this customer</a></p>

<?php elseif ($uid): $u = one('SELECT * FROM users WHERE id = ?', [$uid]); if (!$u) { echo '<p>Customer not found.</p>'; } else {
    $lic = effective_licence($u);
    $lics = q('SELECT * FROM licences WHERE user_id = ? ORDER BY id DESC', [$uid])->fetchAll();
    $pays = q('SELECT * FROM payments WHERE user_id = ? ORDER BY id DESC', [$uid])->fetchAll();
    $devs = q('SELECT * FROM devices WHERE user_id = ? ORDER BY last_seen DESC', [$uid])->fetchAll();
    $use = q('SELECT * FROM usage_log WHERE user_id = ? ORDER BY day DESC LIMIT 14', [$uid])->fetchAll();
    $back = 'admin.php?user=' . $uid; ?>
  <h2><?= e($u['name']) ?> <?= $u['is_admin'] ? '<span class="pill on">Owner</span>' : '' ?> <?= $u['blocked'] ? '<span class="pill off">Blocked</span>' : '' ?></h2>
  <div class="grid">
    <div class="card"><h3>Details</h3>
      <p>Email: <a href="mailto:<?= e($u['email']) ?>"><?= e($u['email']) ?></a><br>Company: <?= e($u['company'] ?: '-') ?><br>
      Phone / WhatsApp: <?= e($u['phone'] ?: '-') ?><br>Signed up: <?= date('j M Y H:i', (int)$u['created_at']) ?></p>
      <?php if (!$u['is_admin']): ?><div class="row" style="margin-top:12px">
        <form method="post" class="inline"><?= csrf_field() ?><input type="hidden" name="back" value="<?= e($back) ?>"><input type="hidden" name="id" value="<?= $uid ?>">
          <input type="hidden" name="act" value="<?= $u['blocked'] ? 'unblock' : 'block' ?>"><button class="btn small <?= $u['blocked'] ? '' : 'danger' ?>"><?= $u['blocked'] ? 'Unblock' : 'Block account' ?></button></form>
        <form method="post" class="inline"><?= csrf_field() ?><input type="hidden" name="id" value="<?= $uid ?>"><input type="hidden" name="act" value="resetpw"><button class="btn small ghost">Reset password</button></form>
      </div><?php endif ?>
    </div>
    <div class="card"><h3>Licence</h3>
      <p><?= status_pill($lic) ?> <?= $lic && $lic['plan'] !== 'owner' ? 'until ' . fmt_date($lic['expires_at']) : '' ?></p>
      <?php if (!$u['is_admin']): ?>
      <form method="post"><?= csrf_field() ?><input type="hidden" name="back" value="<?= e($back) ?>"><input type="hidden" name="id" value="<?= $uid ?>"><input type="hidden" name="act" value="setplan">
        <label>Plan<select name="plan"><option value="standard">Standard</option><option value="pro">Pro</option><option value="trial">Free trial</option></select></label>
        <label>Valid until<input type="date" name="until" value="<?= date('Y-m-d', $now + 365 * DAY) ?>"></label>
        <button class="btn small">Set licence</button></form>
      <form method="post" style="margin-top:8px"><?= csrf_field() ?><input type="hidden" name="back" value="<?= e($back) ?>"><input type="hidden" name="id" value="<?= $uid ?>"><input type="hidden" name="act" value="endlicence"><button class="btn small ghost">End licence now</button></form>
      <?php endif ?>
    </div>
  </div>
  <section><h2>Computers</h2>
    <?php if (!$devs): ?><p class="muted">None.</p><?php else: ?><div class="scroll"><table><tr><th>Computer</th><th>Role</th><th>First signed in</th><th>Last seen</th><th>Status</th><th></th></tr>
    <?php foreach ($devs as $d): ?><tr><td><?= e($d['name'] ?: '-') ?><br><small class="muted"><?= e($d['device_id']) ?></small></td><td><?= e(ucfirst($d['role'] ?: '-')) ?></td>
      <td><?= date('j M Y H:i', (int)$d['created_at']) ?></td><td><?= date('j M Y H:i', (int)$d['last_seen']) ?></td><td><?= $d['revoked'] ? '<span class="pill off">Signed out</span>' : '<span class="pill on">Active</span>' ?></td>
      <td><?php if (!$d['revoked']): ?><form method="post" class="inline"><?= csrf_field() ?><input type="hidden" name="back" value="<?= e($back) ?>"><input type="hidden" name="id" value="<?= (int)$d['id'] ?>"><input type="hidden" name="act" value="revoke"><button class="btn small ghost">Sign out</button></form><?php endif ?></td></tr>
    <?php endforeach ?></table></div><?php endif ?></section>
  <section><h2>Licence history</h2>
    <?php if (!$lics): ?><p class="muted">None.</p><?php else: ?><div class="scroll"><table><tr><th>Plan</th><th>From</th><th>Until</th><th>Status</th><th>Note</th></tr>
    <?php foreach ($lics as $l): ?><tr><td><?= e(plan_name($l['plan'])) ?></td><td><?= fmt_date($l['starts_at']) ?></td><td><?= fmt_date($l['expires_at']) ?></td><td><?= e($l['status']) ?></td><td><?= e($l['note']) ?></td></tr><?php endforeach ?>
    </table></div><?php endif ?></section>
  <section><h2>Payments</h2>
    <?php if (!$pays): ?><p class="muted">None.</p><?php else: ?><div class="scroll"><table><tr><th>Date</th><th>Plan</th><th>Amount</th><th>Method</th><th>Reference</th><th>Status</th><th>Customer's note</th></tr>
    <?php foreach ($pays as $p): ?><tr><td><?= date('j M Y H:i', (int)$p['created_at']) ?></td><td><?= e(plan_name($p['plan'])) ?></td><td><?= money((float)$p['amount_usd']) ?></td><td><?= e($p['method']) ?></td><td><?= e($p['reference']) ?></td><td><?= e($p['status']) ?></td><td><?= e($p['note']) ?></td></tr><?php endforeach ?>
    </table></div><?php endif ?></section>
  <section><h2>Trial broadcast time (last 14 days)</h2>
    <?php if (!$use): ?><p class="muted">None.</p><?php else: ?><div class="scroll"><table><tr><th>Day</th><th>Minutes</th></tr>
    <?php foreach ($use as $r): ?><tr><td><?= e($r['day']) ?></td><td><?= round($r['seconds'] / 60) ?></td></tr><?php endforeach ?></table></div><?php endif ?></section>
<?php } ?>

<?php elseif ($tab === 'users'):
    $s = trim((string)($_GET['q'] ?? ''));
    $rows = $s === '' ? q('SELECT * FROM users ORDER BY id DESC LIMIT 500')->fetchAll()
        : q('SELECT * FROM users WHERE email LIKE ? OR name LIKE ? OR company LIKE ? OR phone LIKE ? ORDER BY id DESC LIMIT 500', array_fill(0, 4, "%$s%"))->fetchAll(); ?>
  <form method="get" class="row"><input type="hidden" name="tab" value="users"><input name="q" value="<?= e($s) ?>" placeholder="Search name, email, company or phone" style="max-width:360px;margin:0"><button class="btn small">Search</button></form>
  <p class="muted"><?= count($rows) ?> customers</p>
  <div class="scroll"><table><tr><th>Name</th><th>Email</th><th>Company</th><th>Phone</th><th>Signed up</th><th>Licence</th><th>Until</th></tr>
  <?php foreach ($rows as $u): $l = effective_licence($u); ?>
    <tr><td><a href="admin.php?user=<?= (int)$u['id'] ?>"><?= e($u['name']) ?></a><?= $u['blocked'] ? ' <span class="pill off">Blocked</span>' : '' ?></td><td><?= e($u['email']) ?></td><td><?= e($u['company']) ?></td><td><?= e($u['phone']) ?></td>
    <td><?= fmt_date($u['created_at']) ?></td><td><?= status_pill($l) ?></td><td><?= $l && $l['plan'] !== 'owner' ? fmt_date($l['expires_at']) : '-' ?></td></tr>
  <?php endforeach ?></table></div>

<?php elseif ($tab === 'payments'):
    $rows = q('SELECT p.*, u.name, u.email FROM payments p JOIN users u ON u.id = p.user_id ORDER BY (p.status IN (\'pending\',\'review\')) DESC, p.id DESC LIMIT 500')->fetchAll(); ?>
  <div class="scroll"><table><tr><th>Date</th><th>Customer</th><th>Plan</th><th>Amount</th><th>Method</th><th>Status</th><th>Customer's note / reference</th><th></th></tr>
  <?php foreach ($rows as $p): ?>
    <tr><td><?= date('j M Y H:i', (int)$p['created_at']) ?></td><td><a href="admin.php?user=<?= (int)$p['user_id'] ?>"><?= e($p['name']) ?></a><br><small><?= e($p['email']) ?></small></td>
    <td><?= e(plan_name($p['plan'])) ?></td><td><?= money((float)$p['amount_usd']) ?></td><td><?= e($p['method'] === 'paystack' ? 'Paystack' : 'Other') ?></td>
    <td><span class="pill <?= $p['status'] === 'paid' ? 'on' : (in_array($p['status'], ['pending', 'review'], true) ? 'wait' : 'off') ?>"><?= e($p['status']) ?></span></td>
    <td><?= e($p['note']) ?><br><small class="muted"><?= e($p['reference']) ?></small></td>
    <td><?php if (in_array($p['status'], ['pending', 'review'], true)): ?>
      <?php if ($p['method'] === 'paystack'): ?><form method="post" class="inline"><?= csrf_field() ?><input type="hidden" name="back" value="admin.php?tab=payments"><input type="hidden" name="id" value="<?= (int)$p['id'] ?>"><input type="hidden" name="act" value="verify"><button class="btn small ghost">Check with Paystack</button></form><?php endif ?>
      <form method="post" class="inline"><?= csrf_field() ?><input type="hidden" name="back" value="admin.php?tab=payments"><input type="hidden" name="id" value="<?= (int)$p['id'] ?>"><input type="hidden" name="act" value="approve"><button class="btn small">Approve</button></form>
      <form method="post" class="inline"><?= csrf_field() ?><input type="hidden" name="back" value="admin.php?tab=payments"><input type="hidden" name="id" value="<?= (int)$p['id'] ?>"><input type="hidden" name="act" value="reject"><button class="btn small ghost">Reject</button></form>
    <?php endif ?></td></tr>
  <?php endforeach ?></table></div>

<?php elseif ($tab === 'codes'):
    $rows = q('SELECT c.*, a.email AS to_email, b.email AS by_email FROM reg_codes c LEFT JOIN users a ON a.id = c.issued_to LEFT JOIN users b ON b.id = c.used_by ORDER BY c.id DESC LIMIT 500')->fetchAll(); ?>
  <form method="post" class="card" style="max-width:520px"><?= csrf_field() ?><input type="hidden" name="back" value="admin.php?tab=codes"><input type="hidden" name="act" value="newcode">
    <h3>Create a registration code</h3><p>For gifts, partners or replacements. Paid customers get their code automatically from Paystack.</p>
    <label>Plan<select name="plan"><option value="standard">Standard</option><option value="pro">Pro</option><option value="trial">Free trial</option></select></label>
    <label>Length (days)<input type="number" name="days" value="365" min="1" max="3650"></label>
    <label>For customer email (optional: leave empty for a code anyone can use once)<input type="email" name="email"></label>
    <label>Note (optional)<input name="note" maxlength="200"></label>
    <button class="btn small">Create code</button></form>
  <section><div class="scroll"><table><tr><th>Code</th><th>Plan</th><th>Days</th><th>For</th><th>Status</th><th>Used by</th><th>Created</th><th>Note</th><th></th></tr>
  <?php foreach ($rows as $c): ?><tr><td><code><?= e($c['code']) ?></code></td><td><?= e(plan_name($c['plan'])) ?></td><td><?= (int)$c['days'] ?></td><td><?= e($c['to_email'] ?: 'Anyone') ?></td>
    <td><span class="pill <?= $c['status'] === 'unused' ? 'wait' : ($c['status'] === 'used' ? 'on' : 'off') ?>"><?= e($c['status']) ?></span></td>
    <td><?= e($c['by_email'] ?: '-') ?><?= $c['used_at'] ? '<br><small>' . date('j M Y H:i', (int)$c['used_at']) . '</small>' : '' ?></td><td><?= fmt_date($c['created_at']) ?></td><td><?= e($c['note']) ?></td>
    <td><?php if ($c['status'] === 'unused'): ?><form method="post" class="inline"><?= csrf_field() ?><input type="hidden" name="back" value="admin.php?tab=codes"><input type="hidden" name="id" value="<?= (int)$c['id'] ?>"><input type="hidden" name="act" value="voidcode"><button class="btn small ghost">Cancel</button></form><?php endif ?></td></tr>
  <?php endforeach ?></table></div></section>

<?php elseif ($tab === 'devices'):
    $rows = q('SELECT d.*, u.name, u.email FROM devices d JOIN users u ON u.id = d.user_id WHERE d.revoked = 0 ORDER BY d.last_seen DESC LIMIT 500')->fetchAll(); ?>
  <div class="scroll"><table><tr><th>Computer</th><th>Customer</th><th>Role</th><th>Last seen</th></tr>
  <?php foreach ($rows as $d): ?><tr><td><?= e($d['name'] ?: '-') ?></td><td><a href="admin.php?user=<?= (int)$d['user_id'] ?>"><?= e($d['name'] ? $d['email'] : $d['email']) ?></a></td><td><?= e(ucfirst($d['role'] ?: '-')) ?></td><td><?= date('j M Y H:i', (int)$d['last_seen']) ?></td></tr><?php endforeach ?>
  </table></div>

<?php else:
    $c = fn($sql, $a = []) => (int)one($sql, $a)['n'];
    $users = $c('SELECT COUNT(*) n FROM users WHERE is_admin = 0');
    $new7 = $c('SELECT COUNT(*) n FROM users WHERE is_admin = 0 AND created_at > ?', [$now - 7 * DAY]);
    $trial = $c('SELECT COUNT(DISTINCT user_id) n FROM licences WHERE plan = ? AND status = ? AND expires_at > ?', ['trial', 'active', $now]);
    $std = $c('SELECT COUNT(DISTINCT user_id) n FROM licences WHERE plan = ? AND status = ? AND expires_at > ?', ['standard', 'active', $now]);
    $pro = $c('SELECT COUNT(DISTINCT user_id) n FROM licences WHERE plan = ? AND status = ? AND expires_at > ?', ['pro', 'active', $now]);
    $pending = $c("SELECT COUNT(*) n FROM payments WHERE status IN ('pending','review')");
    $rev = (int)(one('SELECT COALESCE(SUM(amount_usd),0) n FROM payments WHERE status = ?', ['paid'])['n'] ?? 0);
    $online = $c('SELECT COUNT(*) n FROM devices WHERE revoked = 0 AND last_seen > ?', [$now - 900]);
    $expiring = q('SELECT u.id, u.name, u.email, l.plan, l.expires_at FROM licences l JOIN users u ON u.id = l.user_id WHERE l.status = ? AND l.plan <> ? AND l.expires_at BETWEEN ? AND ? ORDER BY l.expires_at', ['active', 'trial', $now, $now + 30 * DAY])->fetchAll();
    $latest = q('SELECT * FROM users WHERE is_admin = 0 ORDER BY id DESC LIMIT 10')->fetchAll(); ?>
  <div class="grid">
    <div class="card"><h3><?= $users ?></h3><p>Customers (<?= $new7 ?> new this week)</p></div>
    <div class="card"><h3><?= $trial ?> / <?= $std ?> / <?= $pro ?></h3><p>On trial / Standard / Pro</p></div>
    <div class="card"><h3><?= money($rev) ?></h3><p>Total paid (USD)</p></div>
    <div class="card"><h3><?= number_format(usd_rate(), 2) ?></h3><p><?= e(strtoupper($CFG['paystack_currency'])) ?> per USD at checkout</p>
      <form method="post" class="row" style="margin-top:8px"><?= csrf_field() ?><input type="hidden" name="act" value="rate"><input type="hidden" name="back" value="admin.php">
        <input name="rate" value="<?= e(usd_rate()) ?>" style="max-width:120px;margin:0"><button class="btn small">Update rate</button></form></div>
    <div class="card"><h3><?= $pending ?></h3><p>Payments waiting for you <?= $pending ? '· <a href="admin.php?tab=payments">review</a>' : '' ?></p></div>
    <div class="card"><h3><?= $online ?></h3><p>Computers online now</p></div>
  </div>
  <section><h2>Newest sign-ups</h2>
  <div class="scroll"><table><tr><th>Name</th><th>Email</th><th>Phone</th><th>Signed up</th><th>Licence</th></tr>
  <?php foreach ($latest as $u): ?><tr><td><a href="admin.php?user=<?= (int)$u['id'] ?>"><?= e($u['name']) ?></a></td><td><?= e($u['email']) ?></td><td><?= e($u['phone']) ?></td><td><?= date('j M Y H:i', (int)$u['created_at']) ?></td><td><?= status_pill(effective_licence($u)) ?></td></tr><?php endforeach ?>
  </table></div></section>
  <section><h2>Renewals due in the next 30 days</h2>
  <?php if (!$expiring): ?><p class="muted">None.</p><?php else: ?><div class="scroll"><table><tr><th>Customer</th><th>Plan</th><th>Ends</th></tr>
  <?php foreach ($expiring as $x): ?><tr><td><a href="admin.php?user=<?= (int)$x['id'] ?>"><?= e($x['name']) ?></a> <small><?= e($x['email']) ?></small></td><td><?= e(plan_name($x['plan'])) ?></td><td><?= fmt_date($x['expires_at']) ?></td></tr><?php endforeach ?>
  </table></div><?php endif ?></section>
<?php endif ?>
</main>
<?php page_foot();
