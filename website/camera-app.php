<?php
// Public download of the Van Link Camera Android app. Sent as an attachment
// so phones save it instead of trying to open it in the browser.
$p = __DIR__ . '/apps/VanLink-Camera.apk';
if (!is_file($p)) { http_response_code(404); exit('The Android app is not available yet. Use the camera in the browser: camera.php'); }
header('Content-Type: application/vnd.android.package-archive');
header('Content-Disposition: attachment; filename="VanLink-Camera.apk"');
header('Content-Length: ' . filesize($p));
header('Cache-Control: no-cache');
header('X-Content-Type-Options: nosniff');
readfile($p);
