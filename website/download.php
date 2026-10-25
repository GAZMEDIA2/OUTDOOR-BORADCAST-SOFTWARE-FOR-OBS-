<?php
// Installers are only for signed-in customers.
require __DIR__ . '/inc/bootstrap.php';
require_login();
$n = basename((string)($_GET['f'] ?? ''));
$p = __DIR__ . '/downloads/' . $n;
if (!preg_match('/^[A-Za-z0-9._-]+\.(exe|dmg|zip)$/', $n) || !is_file($p)) { http_response_code(404); exit('File not found.'); }
header('Content-Type: application/octet-stream');
header('Content-Disposition: attachment; filename="' . $n . '"');
header('Content-Length: ' . filesize($p));
header('X-Content-Type-Options: nosniff');
readfile($p);
