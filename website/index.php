<?php
require __DIR__ . '/inc/bootstrap.php';
require __DIR__ . '/inc/layout.php';
$user = current_user();
page_head('Live video from your OB van to your studio', $user);
$P = $CFG['plans'];
?>
<main>
  <div class="hero">
    <div>
      <span class="tag">Early access · by <?= e($CFG['company']) ?></span>
      <h1>Live video from your OB van to your studio. No satellite dish.</h1>
      <p class="lead">Van Link sends your OBS programme straight from the van to the studio over Starlink, 4G or 5G, encrypted end to end, with live monitoring at both ends.</p>
      <div class="cta">
        <a class="btn" href="<?= $user ? 'account.php' : 'signup.php' ?>">Start your free 14-day trial</a>
        <a class="btn ghost" href="#pricing">See pricing</a>
      </div>
    </div>
    <img class="shot" src="assets/monitoring.png" alt="Van Link monitoring screens at the studio and in the van">
  </div>

  <section>
    <h2>The problem with satellite uplinks</h2>
    <div class="grid">
      <div class="card"><h3>Bulky</h3><p>A dish and uplink gear take space, power and setup time at every location.</p></div>
      <div class="card"><h3>Expensive</h3><p>Equipment, maintenance and booked satellite time add up for every broadcast.</p></div>
      <div class="card"><h3>Slow to deploy</h3><p>Aligning a dish and booking capacity delays getting on air.</p></div>
      <div class="card"><h3>No feedback</h3><p>The crew often can't see whether the studio is actually receiving them.</p></div>
    </div>
  </section>

  <section>
    <h2>How it works</h2>
    <div class="steps">
      <div class="card"><h3>Install and choose a role</h3><p>Install Van Link on the van computer and the studio computer. One click sets up the firewall and connects to OBS automatically.</p></div>
      <div class="card"><h3>Pair the two ends</h3><p>The studio sends a pairing code to the van by WhatsApp or SMS. The van sends a reply code back, and the two connect directly.</p></div>
      <div class="card"><h3>Go live</h3><p>The van clicks GO LIVE. The picture appears at the studio as its own OBS source, ready for any scene.</p></div>
    </div>
    <p style="margin-top:20px"><img class="shot" src="assets/how-it-works.png" alt="Video flows from the van to the studio while status flows both ways" style="border-radius:14px;border:1px solid var(--line);background:#fff"></p>
  </section>

  <section id="features">
    <h2>Built for real field conditions</h2>
    <div class="grid">
      <div class="card"><h3>Direct van-to-studio link</h3><p>Video goes straight to your studio computer. No relay or cloud server in between.</p></div>
      <div class="card"><h3>Inside OBS Studio</h3><p>The van streams from OBS; the studio receives each van as a normal OBS source, plus a Van Link panel docked in OBS.</p></div>
      <div class="card"><h3>Two-way monitoring</h3><p>Signal bars, delay, data repaired and live status for both ends, on both screens.</p></div>
      <div class="card"><h3>"Live at studio" confirmation</h3><p>The van sees it is live only when the studio confirms it is receiving the picture.</p></div>
      <div class="card"><h3>Encrypted</h3><p>Every feed is encrypted; only your paired studio can decode it.</p></div>
      <div class="card"><h3>Plain-language alerts</h3><p>Clear messages instead of technical jargon. No networking knowledge needed.</p></div>
    </div>
  </section>

  <section id="pricing">
    <h2>Pricing</h2>
    <p class="muted">Paid yearly. No monthly fees.</p>
    <div class="price">
      <div class="card">
        <h3>Free trial</h3>
        <div class="amount">$0 <small>for <?= (int)$CFG['trial_days'] ?> days</small></div>
        <ul><li><?= (int)$CFG['trial_minutes_per_day'] ?> minutes of broadcasting per 24 hours</li><li>One studio and one van</li><li>All Standard features</li></ul>
        <a class="btn ghost" href="signup.php">Start free trial</a>
      </div>
      <div class="card best">
        <h3>Standard</h3>
        <div class="amount"><?= money($P['standard']['price']) ?> <small>/ year</small></div>
        <ul><li>Unlimited broadcasting</li><li>One studio and one van</li><li>Two-way monitoring, auto reconnect</li><li>OBS panel and one-click setup</li></ul>
        <a class="btn" href="buy.php?plan=standard">Buy Standard</a>
      </div>
      <div class="card">
        <h3>Pro</h3>
        <div class="amount"><?= money($P['pro']['price']) ?> <small>/ year</small></div>
        <ul><li>Everything in Standard</li><li>Up to <?= (int)$P['pro']['max_vans'] ?> vans per studio</li><li>Extra camera feeds, with the Program always primary (rolling out)</li><li>Backup internet connection (rolling out)</li></ul>
        <a class="btn" href="buy.php?plan=pro">Buy Pro</a>
      </div>
    </div>
    <p class="note" style="margin-top:16px"><b>Our advice:</b> keep the Program as your primary feed. Upload capacity changes during the day, and Van Link always protects the Program first.</p>
  </section>

  <section>
    <h2>What you need</h2>
    <div class="grid">
      <div class="card"><h3>OBS Studio 28+</h3><p>Free, on both computers.</p></div>
      <div class="card"><h3>Two computers</h3><p>Windows 10 or 11 (macOS and Linux versions available). A graphics card with hardware video support helps at the studio.</p></div>
      <div class="card"><h3>Internet at both ends</h3><p>Starlink, fibre, 4G or 5G. About 8 Mbps upload in the van for 1080p.</p></div>
    </div>
  </section>
</main>
<?php page_foot();
