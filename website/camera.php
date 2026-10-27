<?php
// Van Link phone camera: opens the phone's camera in the browser (iPhone or
// Android, no app to install) and sends it straight to the studio's OBS.
require __DIR__ . '/inc/bootstrap.php';
require __DIR__ . '/inc/cam.php';
$code = cam_room((string)($_GET['c'] ?? ''));
?><!doctype html>
<html lang="en">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1, viewport-fit=cover">
<title>Van Link Camera</title>
<meta name="description" content="Send your phone's camera live to the studio's OBS with Van Link.">
<meta name="theme-color" content="#0B1020">
<link rel="manifest" href="camera.webmanifest">
<meta name="apple-mobile-web-app-capable" content="yes">
<meta name="mobile-web-app-capable" content="yes">
<meta name="apple-mobile-web-app-status-bar-style" content="black-translucent">
<meta name="apple-mobile-web-app-title" content="VL Camera">
<link rel="apple-touch-icon" href="assets/icon.png">
<link rel="icon" href="assets/icon.png">
<style>
:root{color-scheme:dark;--ink:#EEF2F7;--muted:#A3AEC2;--line:#26304A;--bg:#0B1020;--card:#121A2E;--card2:#18223A;--green:#22C55E;--green2:#4ADE80;--red:#F43F5E;--amber:#FBBF24}
*{box-sizing:border-box;-webkit-tap-highlight-color:transparent}
html,body{height:100%}
body{margin:0;background:var(--bg);color:var(--ink);font:16px/1.5 system-ui,-apple-system,"Segoe UI",Roboto,sans-serif}
button,input,select{font:inherit;color:inherit}
.hide{display:none!important}
/* join screen */
#join{max-width:440px;margin:0 auto;padding:max(24px,env(safe-area-inset-top)) 16px 32px;display:flex;flex-direction:column;gap:16px}
.brand{display:flex;align-items:center;gap:10px;font-weight:700;font-size:18px}
.brand img{width:36px;height:36px;border-radius:9px}
h1{font-size:26px;line-height:1.2;margin:8px 0 0}
p{margin:0;color:var(--muted)}
.card{background:var(--card);border:1px solid var(--line);border-radius:14px;padding:18px;display:flex;flex-direction:column;gap:10px}
label{font-size:14px;color:var(--muted)}
input{width:100%;background:var(--bg);border:1px solid var(--line);border-radius:10px;padding:12px;font-size:17px}
input#code{font-size:26px;letter-spacing:4px;text-transform:uppercase;text-align:center;font-family:ui-monospace,Menlo,Consolas,monospace}
.btn{border:0;border-radius:12px;padding:15px 18px;font-weight:700;font-size:17px;background:var(--green);color:#04130A;cursor:pointer;width:100%}
.btn:disabled{opacity:.6}
.err{color:#FDA4AF;font-size:15px}
.tips{font-size:14px;color:var(--muted);padding-left:18px;margin:0}
/* live screen */
#live{position:fixed;inset:0;background:#000}
#preview{position:absolute;inset:0;width:100%;height:100%;object-fit:contain;background:#000}
.bar{position:absolute;left:0;right:0;display:flex;align-items:center;gap:8px;padding:10px max(12px,env(safe-area-inset-right)) 10px max(12px,env(safe-area-inset-left))}
.bar.top{top:0;padding-top:max(10px,env(safe-area-inset-top));background:linear-gradient(#000a,#0000)}
.bar.bottom{bottom:0;padding-bottom:max(12px,env(safe-area-inset-bottom));background:linear-gradient(#0000,#000a);justify-content:center;flex-wrap:wrap}
.pill{display:inline-flex;align-items:center;gap:8px;padding:6px 12px;border-radius:999px;font-weight:700;font-size:14px;background:#0009;border:1px solid #fff3}
.pill .dot{width:10px;height:10px;border-radius:50%;background:currentColor}
.pill.wait{color:var(--amber)}
.pill.ok{color:var(--green2)}
.pill.air{background:var(--red);border-color:var(--red);color:#fff}
.pill.bad{color:#FDA4AF}
.stats{margin-left:auto;font-size:13px;color:#ddd;text-align:right;text-shadow:0 1px 2px #000}
.ctl{min-width:64px;min-height:56px;border-radius:14px;border:1px solid #fff4;background:#000a;padding:6px 12px;font-size:13px;font-weight:600;display:flex;flex-direction:column;align-items:center;justify-content:center;gap:2px;cursor:pointer}
.ctl svg{width:22px;height:22px}
.ctl.off{color:#FDA4AF;border-color:#FDA4AF88}
.ctl.stop{background:var(--red);border-color:var(--red);color:#fff}
select.ctl{appearance:none;-webkit-appearance:none;text-align:center}
#msg{position:absolute;left:50%;top:50%;transform:translate(-50%,-50%);max-width:min(90vw,420px);background:#000c;border:1px solid #fff3;border-radius:14px;padding:16px 18px;text-align:center;font-size:16px}
#msg:empty{display:none}
</style>
</head>
<body>
<section id="join">
  <div class="brand"><img src="assets/icon.png" alt=""> Van Link Camera</div>
  <h1>Send this phone's camera to the studio</h1>
  <p>Works in the browser on iPhone and Android. The picture goes straight to the studio's OBS.</p>
  <div class="card">
    <label for="code">Camera code from the studio</label>
    <input id="code" inputmode="text" autocomplete="off" autocapitalize="characters" spellcheck="false" maxlength="9" placeholder="ABCD-EFGH" value="<?= e($code ? cam_pretty($code) : '') ?>">
    <label for="name">Name of this camera</label>
    <input id="name" maxlength="40" placeholder="e.g. Camera 2 – touchline">
    <button class="btn" id="start">Open camera and go live</button>
    <div class="err" id="joinErr" role="alert"></div>
  </div>
  <ul class="tips">
    <li>Hold the phone sideways for a wide picture.</li>
    <li>Keep this screen open while live. The phone stays awake.</li>
    <li>Wi-Fi or a strong 4G/5G signal gives the best picture.</li>
    <li>Tip: add this page to your home screen to open it like an app.</li>
  </ul>
  <p id="getApp" class="hide"><a href="apps/VanLink-Camera.apk" download style="color:var(--green2)">Android: get the Van Link Camera app (APK)</a></p>
</section>

<section id="live" class="hide">
  <video id="preview" autoplay playsinline muted></video>
  <div class="bar top">
    <span class="pill wait" id="status"><span class="dot"></span><span id="statusText">Connecting…</span></span>
    <span class="stats" id="stats"></span>
  </div>
  <div id="msg"></div>
  <div class="bar bottom">
    <button class="ctl" id="flip" aria-label="Switch camera"><svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" aria-hidden="true"><path d="M4 7h3l2-3h6l2 3h3v12H4z"/><path d="M9 13a3 3 0 0 0 5.6 1.5M15 12a3 3 0 0 0-5.6-1.5"/></svg>Flip</button>
    <button class="ctl" id="mic" aria-label="Microphone on or off"><svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" aria-hidden="true"><rect x="9" y="3" width="6" height="11" rx="3"/><path d="M5 11a7 7 0 0 0 14 0M12 18v3"/></svg><span>Mic on</span></button>
    <select class="ctl" id="quality" aria-label="Picture quality">
      <option value="1080">1080p</option>
      <option value="720" selected>720p</option>
      <option value="480">480p</option>
    </select>
    <button class="ctl stop" id="stop">Stop</button>
  </div>
</section>

<script>
"use strict";
const $ = s => document.querySelector(s);
const API = "api/cam.php";
const QUALITY = {1080: {w: 1920, h: 1080, kbps: 6000}, 720: {w: 1280, h: 720, kbps: 3500}, 480: {w: 854, h: 480, kbps: 1500}};
let room = "", stream = null, pc = null, offerId = "", facing = "environment", running = false;
let videoSender = null, ice = [], wakeLock = null, retryTimer = null, pollTimer = null, statsTimer = null, lastBytes = null, attempt = 0;

const store = {get: k => { try { return localStorage.getItem(k) || ""; } catch { return ""; } },
               set: (k, v) => { try { localStorage.setItem(k, v); } catch {} }};
$("#name").value = store.get("vlcam.name");
if (/Android/i.test(navigator.userAgent) && !/VanLinkCamera/.test(navigator.userAgent)) $("#getApp").classList.remove("hide");
if (!$("#code").value) $("#code").value = store.get("vlcam.code");

async function api(body) {
  const r = await fetch(API, {method: "POST", headers: {"Content-Type": "application/json"}, body: JSON.stringify(body)});
  const j = await r.json().catch(() => ({ok: false, error: "The Van Link website did not answer."}));
  if (!r.ok || !j.ok) { const e = new Error(j.error || "Something went wrong"); e.status = r.status; throw e; }
  return j;
}
const cleanCode = s => s.toUpperCase().replace(/[^A-Z0-9]/g, "");
$("#code").addEventListener("input", e => {
  const c = cleanCode(e.target.value).slice(0, 8);
  e.target.value = c.length > 4 ? c.slice(0, 4) + "-" + c.slice(4) : c;
});

function setStatus(kind, text) { $("#status").className = "pill " + kind; $("#statusText").textContent = text; }
function say(text) { $("#msg").textContent = text || ""; }

// ---------------------------------------------------------------- camera
function videoConstraints() {
  const q = QUALITY[$("#quality").value];
  return {facingMode: {ideal: facing}, width: {ideal: q.w}, height: {ideal: q.h}, frameRate: {ideal: 30, max: 30}};
}
async function openCamera() {
  return navigator.mediaDevices.getUserMedia({video: videoConstraints(),
    audio: {echoCancellation: false, noiseSuppression: false, autoGainControl: true}});
}
async function setBitrate() {
  const s = pc && videoSender; if (!s) return;
  const p = s.getParameters(); if (!p.encodings || !p.encodings.length) p.encodings = [{}];
  p.encodings[0].maxBitrate = QUALITY[$("#quality").value].kbps * 1000;
  p.encodings[0].maxFramerate = 30;
  try { await s.setParameters(p); } catch {}
}

$("#start").addEventListener("click", async () => {
  room = cleanCode($("#code").value);
  $("#joinErr").textContent = "";
  if (room.length !== 8) { $("#joinErr").textContent = "Enter the 8-letter camera code from the studio, like ABCD-EFGH."; return; }
  if (!navigator.mediaDevices || !window.RTCPeerConnection) { $("#joinErr").textContent = "This browser can't send video. Use Safari on iPhone or Chrome on Android."; return; }
  store.set("vlcam.name", $("#name").value.trim()); store.set("vlcam.code", room);
  $("#start").disabled = true; $("#start").textContent = "Opening camera…";
  try { stream = await openCamera(); }
  catch (e) {
    $("#joinErr").textContent = e.name === "NotAllowedError" ? "Allow the camera and microphone for this page, then try again." : "Could not open the camera: " + e.message;
    $("#start").disabled = false; $("#start").textContent = "Open camera and go live"; return;
  }
  $("#join").classList.add("hide"); $("#live").classList.remove("hide");
  $("#preview").srcObject = stream;
  running = true; keepAwake(); connect();
  statsTimer = setInterval(showStats, 1000);
});

// ---------------------------------------------------------------- connection
function waitForCandidates(p) {
  return new Promise(res => {
    if (p.iceGatheringState === "complete") return res();
    const t = setTimeout(res, 3000);
    p.addEventListener("icegatheringstatechange", () => { if (p.iceGatheringState === "complete") { clearTimeout(t); res(); } });
  });
}
function closePC() { if (pc) { pc.onconnectionstatechange = null; pc.close(); pc = null; videoSender = null; } clearTimeout(pollTimer); }
function retry(why) {
  if (!running) return;
  closePC(); setStatus("wait", "Reconnecting…"); say(why || "");
  clearTimeout(retryTimer);
  retryTimer = setTimeout(connect, Math.min(1000 * 2 ** attempt++, 10000));
}

async function connect() {
  if (!running) return;
  closePC();
  setStatus("wait", "Connecting to the studio…");
  try {
    if (!ice.length) ice = (await api({action: "ice", room}).catch(() => ({}))).ice || [];
    pc = new RTCPeerConnection({iceServers: ice});
    for (const t of stream.getTracks()) { const s = pc.addTrack(t, stream); if (t.kind === "video") videoSender = s; }
    const me = pc;
    pc.onconnectionstatechange = () => {
      if (me !== pc) return;
      const s = pc.connectionState;
      if (s === "connected") { attempt = 0; say(""); setStatus("ok", "Live at the studio"); setBitrate(); }
      else if (s === "failed") retry("Lost the connection. Trying again…");
      else if (s === "disconnected") { setStatus("wait", "Weak connection…"); setTimeout(() => { if (me === pc && pc.connectionState === "disconnected") retry(); }, 6000); }
    };
    await pc.setLocalDescription(await pc.createOffer());
    await waitForCandidates(pc);
    offerId = Array.from(crypto.getRandomValues(new Uint8Array(12)), b => b.toString(16).padStart(2, "0")).join("");
    const j = await api({action: "join", room, id: offerId, offer: pc.localDescription.sdp, name: $("#name").value.trim()});
    if (j.ice && j.ice.length && !ice.length) ice = j.ice;
    say("Waiting for the studio to answer…");
    poll(Date.now());
  } catch (e) {
    if (e.status === 404 || e.status === 429) return stop(e.message);
    retry(e.message);
  }
}

// Reads the studio's answer, then keeps asking now and then for the ON AIR light.
async function poll(started) {
  const me = pc; if (!running || !me) return;
  let wait = 1000;
  try {
    const j = await api({action: "answer", room, id: offerId});
    if (me !== pc) return;
    if (j.answer && !me.remoteDescription) {
      await me.setRemoteDescription({type: "answer", sdp: j.answer});
      say("Connecting the picture…");
    }
    if (me.remoteDescription) {
      wait = 3000;
      if (me.connectionState === "connected") setStatus(j.onAir ? "air" : "ok", j.onAir ? "ON AIR" : "Live at the studio");
    } else if (!j.studio) say("The studio's OBS is closed. Waiting for it…");
    if (!me.remoteDescription && Date.now() - started > 45000) return retry("The studio did not answer. Trying again…");
    if (me.remoteDescription && me.connectionState !== "connected" && Date.now() - started > 30000) return retry("Could not reach the studio directly. Trying again…");
  } catch (e) {
    if (e.status === 409) return stop("Another phone is now using this camera code.");
    wait = 3000;
  }
  pollTimer = setTimeout(() => poll(started), wait);
}

function stop(why) {
  running = false;
  clearTimeout(retryTimer); clearInterval(statsTimer);
  if (offerId) fetch(API, {method: "POST", keepalive: true, headers: {"Content-Type": "application/json"}, body: JSON.stringify({action: "bye", room, id: offerId})}).catch(() => {});
  closePC();
  if (stream) stream.getTracks().forEach(t => t.stop());
  stream = null; offerId = "";
  if (wakeLock) wakeLock.release().catch(() => {});
  $("#live").classList.add("hide"); $("#join").classList.remove("hide");
  $("#start").disabled = false; $("#start").textContent = "Open camera and go live";
  $("#joinErr").textContent = why || "";
}

// ---------------------------------------------------------------- controls
$("#stop").addEventListener("click", () => { if (confirm("Stop sending this camera to the studio?")) stop(); });
$("#mic").addEventListener("click", () => {
  const a = stream && stream.getAudioTracks()[0]; if (!a) return;
  a.enabled = !a.enabled;
  $("#mic").classList.toggle("off", !a.enabled);
  $("#mic span").textContent = a.enabled ? "Mic on" : "Mic off";
});
async function replaceVideo() {
  const old = stream.getVideoTracks()[0];
  old.stop(); // some phones can only open one camera at a time
  const ns = await navigator.mediaDevices.getUserMedia({video: videoConstraints()});
  const t = ns.getVideoTracks()[0];
  stream.removeTrack(old); stream.addTrack(t);
  $("#preview").srcObject = stream;
  if (pc && videoSender) await videoSender.replaceTrack(t);
  setBitrate();
}
$("#flip").addEventListener("click", async () => {
  facing = facing === "environment" ? "user" : "environment";
  $("#preview").style.transform = facing === "user" ? "scaleX(-1)" : "";
  try { await replaceVideo(); } catch (e) { say("Could not switch camera: " + e.message); }
});
$("#quality").addEventListener("change", async () => {
  const t = stream && stream.getVideoTracks()[0]; if (!t) return;
  const q = QUALITY[$("#quality").value];
  try { await t.applyConstraints({width: {ideal: q.w}, height: {ideal: q.h}, frameRate: {ideal: 30, max: 30}}); } catch { try { await replaceVideo(); } catch {} }
  setBitrate();
});

async function showStats() {
  if (!pc || pc.connectionState !== "connected") { $("#stats").textContent = ""; lastBytes = null; return; }
  let bytes = 0, at = 0, rtt = null, w = 0, h = 0;
  (await pc.getStats()).forEach(r => {
    if (r.type === "outbound-rtp" && r.kind === "video") { bytes += r.bytesSent; at = r.timestamp; w = r.frameWidth || w; h = r.frameHeight || h; }
    if (r.type === "candidate-pair" && r.nominated && r.state === "succeeded" && r.currentRoundTripTime != null) rtt = r.currentRoundTripTime;
  });
  let kbps = 0;
  if (lastBytes) kbps = (bytes - lastBytes.b) * 8 / Math.max(1, at - lastBytes.t);
  lastBytes = {b: bytes, t: at};
  $("#stats").textContent = [h ? h + "p" : "", kbps ? (kbps >= 1000 ? (kbps / 1000).toFixed(1) + " Mbps" : Math.round(kbps) + " kbps") : "",
    rtt != null ? "delay " + (rtt / 2).toFixed(2) + " s" : ""].filter(Boolean).join(" · ");
}

// Keep the screen on while live (the camera stops if the phone locks).
async function keepAwake() {
  try { if ("wakeLock" in navigator) wakeLock = await navigator.wakeLock.request("screen"); } catch {}
}
document.addEventListener("visibilitychange", () => {
  if (document.visibilityState === "visible" && running) {
    keepAwake();
    const v = stream && stream.getVideoTracks()[0];
    if (v && v.readyState === "ended") replaceVideo().catch(() => {});
  }
});
window.addEventListener("pagehide", () => { if (running && offerId) navigator.sendBeacon && navigator.sendBeacon(API, new Blob([JSON.stringify({action: "bye", room, id: offerId})], {type: "application/json"})); });
</script>
</body>
</html>
