<?php
// Paystack sends the customer back here after paying.
require __DIR__ . '/inc/bootstrap.php';
require __DIR__ . '/inc/paystack.php';

$ref = (string)($_GET['reference'] ?? $_GET['trxref'] ?? '');
if (!preg_match('/^VL-\d+-[0-9a-f]{12}$/', $ref)) redirect('account.php');
try {
    $r = paystack_confirm($ref);
} catch (Throwable $e) {
    $r = 'pending';
}
if ($r === 'paid') flash('Payment received, thank you! Your registration code is below and has been emailed to you. Enter it in the Van Link app, or click Activate here.');
elseif ($r === 'pending') flash('Your payment is still being confirmed. Click "Check my payment" on this page in a minute.', 'err');
else flash('The payment did not go through. You have not been charged for Van Link. Please try again.', 'err');
redirect('account.php');
