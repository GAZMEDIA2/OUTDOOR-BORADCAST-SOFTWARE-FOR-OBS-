<?php
// The app signs this computer out.
require __DIR__ . '/../inc/bootstrap.php';
require __DIR__ . '/../inc/api.php';
if ($_SERVER['REQUEST_METHOD'] !== 'POST') api_out(['ok' => false], 405);
$dev = device_from_token((string)(api_in()['token'] ?? ''));
if ($dev) q('UPDATE devices SET revoked = 1 WHERE id = ?', [$dev['id']]);
api_out(['ok' => true]);
