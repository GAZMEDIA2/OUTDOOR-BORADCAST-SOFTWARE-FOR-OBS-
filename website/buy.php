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
    if (!paystack_enabled()) $err = 'Online payment is being set up. Please try again later.';
    else {
        try { redirect(paystack_start($user, $plan)); }
        catch (Throwable $ex) { $err = 'Payment could not start (' . $ex->getMessage() . '). Please try again in a moment.'; }
    }
}

page_head('Buy ' . plan_name($plan), $user);
$cur = current_licence((int)$user['id']);
$cur3 = strtoupper($CFG['paystack_currency']);
$local = $cur3 === 'USD' ? '' : '₦' . number_format(paystack_amount($plan) / 100);
?>
<main class="narrow">
  <h1>Van Link <?= e(plan_name($plan)) ?></h1>
  <p><b><?= money($info['price']) ?> per year.</b> No monthly fees.</p>
  <?php if (licence_active($cur) && $cur['plan'] === $plan): ?>
    <p class="ok-box">You already have <?= e(plan_name($plan)) ?> until <?= fmt_date($cur['expires_at']) ?>. Paying now adds one more year.</p>
  <?php elseif (licence_active($cur) && $cur['plan'] !== 'trial'): ?>
    <p class="note">You have <?= e(plan_name($cur['plan'])) ?> until <?= fmt_date($cur['expires_at']) ?>. Buying <?= e(plan_name($plan)) ?> switches you to it for one year from today.</p>
  <?php endif ?>
  <p style="margin:8px 0"><a href="buy.php?plan=<?= $plan === 'pro' ? 'standard' : 'pro' ?>">Switch to <?= $plan === 'pro' ? 'Standard' : 'Pro' ?> instead</a></p>
  <?php if ($err): ?><p class="err-box"><?= e($err) ?></p><?php endif ?>

  <form method="post" class="card"><?= csrf_field() ?>
    <input type="hidden" name="plan" value="<?= e($plan) ?>">
    <h3><?= money($info['price']) ?> <small class="muted">/ year</small></h3>
    <p class="cards-line">We accept all cards: <b>Visa</b>, <b>Mastercard</b>, <b>Verve</b> and <b>American Express</b>, from any country. Bank transfer and USSD also work in Nigeria.</p>
    <p class="muted">Your registration code appears as soon as the payment is confirmed, and is emailed to you.</p>
    <button class="btn">Pay <?= money($info['price']) ?> by card</button>
    <p class="muted" style="font-size:.85em;margin-top:10px">Secure checkout by Paystack.<?php if ($local): ?> Payments are settled in Nigeria, so the checkout page and your bank statement may show the naira equivalent (<?= e($local) ?>).<?php endif ?></p>
  </form>
</main>
<?php page_foot();
