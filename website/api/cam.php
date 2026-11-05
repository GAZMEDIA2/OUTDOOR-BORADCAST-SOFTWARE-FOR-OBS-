<?php
// Phone camera connection API, used by camera.php (the phone), studio-camera.php
// (the OBS browser source at the studio) and the Van Link app (status).
//
//   host   studio source: claims/keeps its room, posts its answer, reads the phone's offer
//   ice    phone: the address lookup servers to use
//   join   phone: posts its offer for a room code
//   answer phone: reads the studio's answer
//   bye    phone: says it stopped
//   status Van Link app: is the phone live?
require __DIR__ . '/../inc/bootstrap.php';
require __DIR__ . '/../inc/cam.php';
header('Access-Control-Allow-Origin: *');
header('Access-Control-Allow-Headers: Content-Type');
if ($_SERVER['REQUEST_METHOD'] === 'OPTIONS') exit;
if ($_SERVER['REQUEST_METHOD'] !== 'POST') api_out(['ok' => false, 'error' => 'POST only'], 405);

cam_table();
$in = api_in();
$room = cam_room((string)($in['room'] ?? ''));
$now = time();
if ($room === '') api_out(['ok' => false, 'error' => 'That camera code is not right. It looks like ABCD-EFGH.'], 400);
$r = one('SELECT * FROM cam_rooms WHERE room = ?', [$room]);
$id = substr(preg_replace('/[^A-Za-z0-9]/', '', (string)($in['id'] ?? '')), 0, 32);

// Studio side: the room belongs to whoever first opened it with its key.
function cam_host(?array $r, string $room, array $in, int $now): array {
    $key = (string)($in['key'] ?? '');
    if (strlen($key) < 24) api_out(['ok' => false, 'error' => 'missing key'], 400);
    $hash = hash('sha256', $key);
    if (!$r) {
        q('INSERT INTO cam_rooms (room, host_hash, created_at) VALUES (?, ?, ?)', [$room, $hash, $now]);
        $r = one('SELECT * FROM cam_rooms WHERE room = ?', [$room]);
    } elseif (!hash_equals($r['host_hash'], $hash)) {
        api_out(['ok' => false, 'error' => 'This camera code belongs to another studio.'], 403);
    }
    return $r;
}

switch ($in['action'] ?? '') {
    case 'ice':
        api_out(['ok' => true, 'ice' => cam_ice()]);

    case 'host':
        $r = cam_host($r, $room, $in, $now);
        $state = in_array($in['state'] ?? '', ['waiting', 'live'], true) ? $in['state'] : 'waiting';
        $answer = (string)($in['answer'] ?? '');
        if ($answer !== '' && $id !== '' && $id === $r['offer_id'] && strlen($answer) < 60000) {
            q('UPDATE cam_rooms SET answer = ?, answer_for = ? WHERE room = ?', [$answer, $id, $room]);
            $r['answer_for'] = $id;
        }
        q('UPDATE cam_rooms SET host_seen = ?, host_state = ?, on_air = ? WHERE room = ?', [$now, $state, empty($in['onAir']) ? 0 : 1, $room]);
        $pending = $r['offer_id'] !== '' && $r['answer_for'] !== $r['offer_id'] && $now - (int)$r['phone_seen'] < 60;
        api_out(['ok' => true, 'ice' => cam_ice(), 'name' => $r['phone_name'],
                 'offer' => $pending ? $r['offer'] : null, 'id' => $r['offer_id']]);

    case 'join':
        if (!$r) {
            if (too_many_attempts('cam:' . client_ip())) api_out(['ok' => false, 'error' => 'Too many wrong codes. Wait 15 minutes and try again.'], 429);
            note_failed_attempt('cam:' . client_ip());
            api_out(['ok' => false, 'error' => 'No studio has this camera code. Check it with the studio.'], 404);
        }
        if ($now - (int)$r['host_seen'] > 20) {
            api_out(['ok' => false, 'error' => 'The studio is not ready for this camera yet. Ask the studio to open OBS with Van Link.'], 409);
        }
        $offer = (string)($in['offer'] ?? '');
        if ($id === '' || $offer === '' || strlen($offer) > 60000) api_out(['ok' => false, 'error' => 'bad offer'], 400);
        $name = trim(mb_substr(preg_replace('/[\x00-\x1F]/', '', (string)($in['name'] ?? '')), 0, 60));
        q("UPDATE cam_rooms SET offer = ?, offer_id = ?, phone_name = ?, answer = NULL, answer_for = '', phone_seen = ? WHERE room = ?",
            [$offer, $id, $name, $now, $room]);
        api_out(['ok' => true, 'ice' => cam_ice()]);

    case 'answer':
        if (!$r || $id === '' || $r['offer_id'] !== $id) api_out(['ok' => false, 'error' => 'Another phone took over this camera code.'], 409);
        q('UPDATE cam_rooms SET phone_seen = ? WHERE room = ?', [$now, $room]);
        $ready = $r['answer_for'] === $id;
        api_out(['ok' => true, 'answer' => $ready ? $r['answer'] : null, 'onAir' => (bool)$r['on_air'],
                 'studio' => $now - (int)$r['host_seen'] <= 20]);

    case 'bye':
        if ($r && $id !== '' && $r['offer_id'] === $id) q("UPDATE cam_rooms SET offer = NULL, offer_id = '' WHERE room = ?", [$room]);
        api_out(['ok' => true]);

    case 'status':
        if (!$r) api_out(['ok' => true, 'sourceOpen' => false, 'state' => '', 'name' => '']);
        cam_host($r, $room, $in, $now);
        api_out(['ok' => true, 'sourceOpen' => $now - (int)$r['host_seen'] <= 20,
                 'state' => $now - (int)$r['host_seen'] <= 20 ? $r['host_state'] : '', 'name' => $r['phone_name'], 'onAir' => (bool)$r['on_air']]);
}
api_out(['ok' => false, 'error' => 'unknown action'], 400);
