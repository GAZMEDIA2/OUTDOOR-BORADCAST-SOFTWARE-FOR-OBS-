<?php
// Studio end of a Van Link phone camera. Van Link adds this page to OBS as a
// Browser Source; it receives the phone's picture and sound directly (WebRTC).
// Opened without a code, it makes a new camera and shows the OBS address to use.
require __DIR__ . '/inc/bootstrap.php';
require __DIR__ . '/inc/cam.php';
$room = cam_room((string)($_GET['room'] ?? ''));
$key = preg_replace('/[^A-Za-z0-9]/', '', (string)($_GET['key'] ?? ''));
$label = trim(mb_substr((string)($_GET['name'] ?? ''), 0, 60));
$base = site_url('');
?><!doctype html>
<html lang="en">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>Van Link Camera – Studio</title>
<meta name="robots" content="noindex">
<style>
:root{color-scheme:dark;--ink:#EEF2F7;--muted:#A3AEC2;--line:#26304A;--bg:#0B1020;--card:#121A2E;--green:#22C55E;--green2:#4ADE80;--red:#F43F5E}
*{box-sizing:border-box}
html,body{margin:0;height:100%;overflow:hidden;background:#000;color:var(--ink);font:16px/1.5 system-ui,-apple-system,"Segoe UI",Roboto,sans-serif}
video{position:fixed;inset:0;width:100%;height:100%;object-fit:contain;background:#000}
#wait{position:fixed;inset:0;display:flex;align-items:center;justify-content:center;background:var(--bg);padding:4vh 4vw}
#wait .box{display:flex;gap:5vw;align-items:center;max-width:1500px}
#wait h1{font-size:4.2vh;margin:0 0 1vh}
#wait p{font-size:2.6vh;color:var(--muted);margin:0 0 2vh}
#wait .code{font:700 9vh/1 ui-monospace,Menlo,Consolas,monospace;letter-spacing:.6vh;color:var(--green2);margin:1vh 0 2vh}
#wait .url{font-size:2.8vh;color:var(--ink)}
#qr{background:#fff;padding:2vh;border-radius:2vh;flex:none}
#qr img,#qr canvas{display:block;width:32vh!important;height:32vh!important}
#state{font-size:2.4vh;color:var(--muted);margin-top:2vh}
.hide{display:none!important}
/* set-up page (opened in a normal browser without a code) */
#setup{max-width:620px;margin:0 auto;padding:32px 16px;overflow:auto;height:100%}
#setup .card{background:var(--card);border:1px solid var(--line);border-radius:14px;padding:18px;margin-top:16px}
#setup input{width:100%;background:var(--bg);color:var(--ink);border:1px solid var(--line);border-radius:10px;padding:10px;font:13px ui-monospace,Menlo,Consolas,monospace}
#setup ol{color:var(--muted);padding-left:20px}
#setup b{color:var(--ink)}
</style>
</head>
<body>
<?php if ($room === '' || strlen($key) < 24): ?>
<div id="setup">
  <h1>Add a phone camera to OBS</h1>
  <p style="color:var(--muted)">Van Link at the studio does this for you (Studio › <b>Add a phone camera</b>). Without Van Link, set it up by hand:</p>
  <div class="card">
    <ol>
      <li>In OBS, add a <b>Browser</b> source. Width <b>1920</b>, height <b>1080</b>. Tick <b>Control audio via OBS</b>.</li>
      <li>Paste this address as its URL:<br><input id="obsUrl" readonly onclick="this.select()"></li>
      <li>On the phone, open <b><?= e(preg_replace('#^https?://#', '', $base)) ?>camera.php</b> and enter the code <b id="newCode"></b>, or scan the QR code shown in OBS.</li>
    </ol>
    <p style="color:var(--muted);font-size:14px">Keep this address private: it is the studio's key for this camera.</p>
  </div>
</div>
<script>
const A = "ABCDEFGHJKLMNPQRSTUVWXYZ23456789", rnd = n => Array.from(crypto.getRandomValues(new Uint8Array(n)));
const room = rnd(8).map(b => A[b % 32]).join(""), key = rnd(16).map(b => b.toString(16).padStart(2, "0")).join("");
document.getElementById("obsUrl").value = location.href.split("?")[0] + "?room=" + room + "&key=" + key;
document.getElementById("newCode").textContent = room.slice(0, 4) + "-" + room.slice(4);
</script>
<?php else: ?>
<video id="v" autoplay playsinline></video>
<div id="wait">
  <div class="box">
    <div id="qr"></div>
    <div>
      <h1><?= e($label !== '' ? $label : 'Phone camera') ?> · waiting for the phone</h1>
      <p>On the phone, scan the code or open <span class="url"><?= e(preg_replace('#^https?://#', '', $base)) ?>camera.php</span> and enter:</p>
      <div class="code"><?= e(cam_pretty($room)) ?></div>
      <p>Works on iPhone and Android, in the browser. Nothing to install.</p>
      <div id="state"></div>
    </div>
  </div>
</div>
<script src="https://cdnjs.cloudflare.com/ajax/libs/qrcodejs/1.0.0/qrcode.min.js"></script>
<script>
"use strict";
const ROOM = <?= json_encode($room) ?>, KEY = <?= json_encode($key) ?>, API = <?= json_encode(site_url('api/cam.php')) ?>;
const PHONE_URL = <?= json_encode(site_url('camera.php?c=' . $room)) ?>;
const v = document.getElementById("v"), wait = document.getElementById("wait"), stateEl = document.getElementById("state");
let pc = null, answering = "", answer = null, hadVideo = false, onAir = false, timer = null;

try { new QRCode(document.getElementById("qr"), {text: PHONE_URL, width: 512, height: 512, correctLevel: QRCode.CorrectLevel.M}); }
catch { document.getElementById("qr").classList.add("hide"); }

// OBS tells browser sources when they are on Program: that lights ON AIR on the phone.
window.addEventListener("obsSourceActiveChanged", e => { onAir = !!(e.detail && e.detail.active); showWait(); });

const live = () => pc && pc.connectionState === "connected";
function showWait() {
  // Never put the waiting card on air once a picture has been seen: keep the last frame.
  wait.classList.toggle("hide", live() || (hadVideo && onAir));
}

async function host() {
  let next = 2000;
  try {
    const body = {action: "host", room: ROOM, key: KEY, state: live() ? "live" : "waiting", onAir};
    if (answer) { body.answer = answer.sdp; body.id = answer.id; }
    const r = await fetch(API, {method: "POST", headers: {"Content-Type": "application/json"}, body: JSON.stringify(body)});
    const j = await r.json();
    if (!j.ok) { stateEl.textContent = j.error || "Problem with this camera code."; next = 10000; }
    else {
      if (answer && answer.id === j.id) answer = null; // delivered
      if (j.offer && j.id && j.id !== answering) { await accept(j.offer, j.id, j.ice || []); next = 300; }
      else if (live()) next = 4000;
      if (!live()) stateEl.textContent = j.name && answering ? "Connecting to " + j.name + "…" : "";
    }
  } catch (e) { stateEl.textContent = "Can't reach the Van Link website. Check the studio's internet."; next = 5000; }
  timer = setTimeout(host, next);
}

function waitForCandidates(p) {
  return new Promise(res => {
    if (p.iceGatheringState === "complete") return res();
    const t = setTimeout(res, 3000);
    p.addEventListener("icegatheringstatechange", () => { if (p.iceGatheringState === "complete") { clearTimeout(t); res(); } });
  });
}

async function accept(sdp, id, ice) {
  if (pc) { pc.close(); pc = null; }
  answering = id;
  const me = pc = new RTCPeerConnection({iceServers: ice});
  me.ontrack = e => {
    if (v.srcObject !== e.streams[0]) {
      v.srcObject = e.streams[0];
      v.muted = false;
      v.play().catch(() => { v.muted = true; v.play().catch(() => {}); });
    }
  };
  me.onconnectionstatechange = () => {
    if (me !== pc) return;
    if (me.connectionState === "connected") hadVideo = true;
    if (me.connectionState === "failed" || me.connectionState === "closed") answering = "";
    showWait();
  };
  // Show frames as soon as they arrive (lowest delay).
  me.addEventListener("track", e => { try { e.receiver.playoutDelayHint = 0; e.receiver.jitterBufferTarget = 0; } catch {} });
  await me.setRemoteDescription({type: "offer", sdp});
  await me.setLocalDescription(await me.createAnswer());
  await waitForCandidates(me);
  if (me === pc) answer = {sdp: me.localDescription.sdp, id};
}

host();
</script>
<?php endif ?>
</body>
</html>
