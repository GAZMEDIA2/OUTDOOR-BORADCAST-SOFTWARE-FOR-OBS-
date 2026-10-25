<?php
require __DIR__ . '/inc/bootstrap.php';
require __DIR__ . '/inc/layout.php';
require __DIR__ . '/inc/paystack.php';

$plan = (string)($_GET['plan'] ?? $_POST['plan'] ?? 'standard');
if (!in_array($plan, ['standard', 'pro'], true)) $plan = 'standard';
$user = require_login();
$info = plan_info($plan);
$err = '';

if ($_SERVER['REQUEST_METHOD'] === 'POST') {
    csrf_check();
    $how = (string)($_POST['how'] ?? '');
    if ($how === 'paystack' && paystack_enabled()) {
        try { redirect(paystack_start($user, $plan)); }
        catch (Throwable $ex) { $err = 'Card payment is not available right now (' . $ex->getMessage() . '). Try again, or pay manually below.'; }
    } elseif ($how === 'manual') {
        $note = trim((string)($_POST['note'] ?? ''));
        if ($note === '') $err = 'Tell us how you paid (e.g. MoMo number and transaction ID) so we can find your payment.';
        else {
            q('INSERT INTO payments (user_id, plan, amount_usd, method, reference, status, note, created_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?)',
                [$user['id'], $plan, (int)$info['price'], 'manual', 'MAN-' . $user['id'] . '-' . bin2hex(random_bytes(5)), 'pending', substr($note, 0, 500), time()]);
            flash('Thanks! We received your payment details. Your ' . plan_name($plan) . ' licence starts as soon as we confirm the payment.');
            redirect('account.php');
        }
    }
}

page_head('Buy ' . plan_name($plan), $user);
$cur = current_licence((int)$user['id']);
$local = strtoupper($CFG['paystack_currency']) === 'USD' ? '' : ' (about ' . strtoupper($CFG['paystack_currency']) . ' ' . number_format($info['price'] * (float)$CFG['usd_rate']) . ')';
?>
<main class="narrow">
  <h1>Van Link <?= e(plan_name($plan)) ?></h1>
  <p><b><?= money($info['price']) ?> per year</b><?= e($local) ?>. No monthly fees.</p>
  <?php if (licence_active($cur) && $cur['plan'] === $plan): ?>
    <p class="ok-box">You already have <?= e(plan_name($plan)) ?> until <?= fmt_date($cur['expires_at']) ?>. Paying now adds one more year.</p>
  <?php elseif (licence_active($cur) && $cur['plan'] !== 'trial'): ?>
    <p class="note">You have <?= e(plan_name($cur['plan'])) ?> until <?= fmt_date($cur['expires_at']) ?>. Buying <?= e(plan_name($plan)) ?> switches you to it for one year from today.</p>
  <?php endif ?>
  <p style="margin:8px 0"><a href="buy.php?plan=<?= $plan === 'pro' ? 'standard' : 'pro' ?>">Switch to <?= $plan === 'pro' ? 'Standard' : 'Pro' ?> instead</a></p>
  <?php if ($err): ?><p class="err-box"><?= e($err) ?></p><?php endif ?>

  <?php if (paystack_enabled()): ?>
  <form method="post" class="card"><?= csrf_field() ?>
    <input type="hidden" name="plan" value="<?= e($plan) ?>"><input type="hidden" name="how" value="paystack">
    <h3>Pay by card, bank transfer or USSD</h3>
    <p class="muted">Secure payment with Paystack. Your licence starts immediately.</p>
    <button class="btn">Pay with Paystack</button>
  </form>
  <?php endif ?>

  <form method="post" class="card" style="margin-top:16px"><?= csrf_field() ?>
    <input type="hidden" name="plan" value="<?= e($plan) ?>"><input type="hidden" name="how" value="manual">
    <h3>Pay by mobile money or bank transfer</h3>
    <pre class="pay"><?= e($CFG['manual_payment_text']) ?></pre>
    <?php if (!empty($CFG['support_whatsapp'])): ?><p class="muted">Questions: WhatsApp <?= e($CFG['support_whatsapp']) ?></p><?php endif ?>
    <label>After paying, tell us the details (amount, sender number or name, transaction ID)
      <textarea name="note" required maxlength="500"></textarea></label>
    <button class="btn dark">I have paid</button>
  </form>
</main>
<?php page_foot();
