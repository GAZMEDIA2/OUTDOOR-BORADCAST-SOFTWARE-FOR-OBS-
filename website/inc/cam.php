<?php
// Phone cameras: the website only passes the connection details between the
// phone and the studio (WebRTC offer/answer). The video goes directly from
// the phone to the studio's OBS, never through this website.

const CAM_ALPHABET = 'ABCDEFGHJKLMNPQRSTUVWXYZ23456789'; // no 0/O, 1/I

function cam_table(): void {
    $driver = db()->getAttribute(PDO::ATTR_DRIVER_NAME);
    $opts = $driver === 'sqlite' ? '' : ' ENGINE=InnoDB DEFAULT CHARSET=utf8mb4';
    $big = $driver === 'sqlite' ? 'TEXT' : 'MEDIUMTEXT';
    db()->exec("CREATE TABLE IF NOT EXISTS cam_rooms (room CHAR(8) NOT NULL PRIMARY KEY, host_hash CHAR(64) NOT NULL,
        host_seen INT NOT NULL DEFAULT 0, host_state VARCHAR(10) NOT NULL DEFAULT '', on_air TINYINT NOT NULL DEFAULT 0,
        offer $big, offer_id VARCHAR(32) NOT NULL DEFAULT '', phone_name VARCHAR(60) NOT NULL DEFAULT '',
        answer $big, answer_for VARCHAR(32) NOT NULL DEFAULT '', phone_seen INT NOT NULL DEFAULT 0, created_at INT NOT NULL)$opts");
}

// "abcd-efgh" / "ABCDEFGH" -> "ABCDEFGH", or '' if it isn't a valid code.
function cam_room(string $s): string {
    $s = strtoupper(preg_replace('/[^A-Za-z0-9]/', '', $s));
    return (strlen($s) === 8 && strspn($s, CAM_ALPHABET) === 8) ? $s : '';
}
function cam_pretty(string $room): string { return substr($room, 0, 4) . '-' . substr($room, 4); }

// Address lookup (STUN) servers for WebRTC; config.php can add a TURN relay
// for networks where a direct connection is impossible.
function cam_ice(): array {
    global $CFG;
    return $CFG['cam_ice_servers'] ?? [
        ['urls' => ['stun:stun.l.google.com:19302', 'stun:stun1.l.google.com:19302']],
        ['urls' => 'stun:stun.cloudflare.com:3478'],
    ];
}
