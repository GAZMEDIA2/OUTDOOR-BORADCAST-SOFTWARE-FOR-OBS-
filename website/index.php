<?php
require __DIR__ . '/inc/bootstrap.php';
require __DIR__ . '/inc/layout.php';
$user = current_user();
page_head('Live video from your OB van to your studio', $user);
$P = $CFG['plans'];
$start = $user ? 'account.php' : 'signup.php';
?>
<main class="home">

  <!-- ============ HERO ============ -->
  <div class="hero2">
    <div>
      <span class="tag">Early access · Windows · Works with OBS Studio</span>
      <h1>Send live pictures from your outside broadcast van to your studio. <span class="hl">No satellite dish.</span></h1>
      <p class="lead">Van Link uses the internet your van already has (Starlink, 4G or 5G) to carry your live programme straight to your studio computer. The studio sees it as a normal camera in OBS, ready to put on air.</p>
      <div class="cta">
        <a class="btn big" href="<?= $start ?>">Start your free 14-day trial</a>
        <a class="btn ghost big" href="#how">See how it works</a>
      </div>
      <ul class="ticks">
        <li>No card needed for the trial</li>
        <li>No monthly fees</li>
        <li>No networking skills needed</li>
      </ul>
    </div>
    <figure class="art" aria-label="Illustration: a van sends live video over the internet to a studio, which confirms it is live">
      <svg viewBox="0 0 560 400" role="img" aria-label="Van sends video through the internet to the studio">
        <defs>
          <linearGradient id="sky" x1="0" y1="0" x2="0" y2="1"><stop offset="0" stop-color="#1E1B4B"/><stop offset="1" stop-color="#0F172A"/></linearGradient>
          <marker id="ah" viewBox="0 0 10 10" refX="9" refY="5" markerWidth="7" markerHeight="7" orient="auto"><path d="M0 0L10 5L0 10z" fill="#22C55E"/></marker>
        </defs>
        <rect x="0" y="0" width="560" height="400" rx="24" fill="url(#sky)"/>
        <g fill="#E0E7FF" opacity=".7"><circle cx="40" cy="40" r="1.6"/><circle cx="120" cy="80" r="1.2"/><circle cx="470" cy="50" r="1.8"/><circle cx="520" cy="110" r="1.2"/><circle cx="410" cy="30" r="1.2"/><circle cx="180" cy="30" r="1.4"/></g>
        <!-- road -->
        <rect x="0" y="318" width="560" height="82" rx="0" fill="#1E293B"/>
        <rect x="0" y="318" width="560" height="4" fill="#F59E0B"/>
        <!-- van -->
        <g transform="translate(28 210)">
          <rect x="0" y="20" width="150" height="86" rx="10" fill="#2563EB"/>
          <path d="M150 44h38l26 30v32h-64z" fill="#1D4ED8"/>
          <rect x="160" y="50" width="30" height="22" rx="3" fill="#9CC5E8"/>
          <rect x="14" y="36" width="56" height="30" rx="4" fill="#22C55E"/>
          <text x="42" y="56" text-anchor="middle" font-size="12" font-weight="700" fill="#fff" font-family="system-ui">OB</text>
          <circle cx="40" cy="108" r="16" fill="#111827"/><circle cx="40" cy="108" r="6" fill="#9CA3AF"/>
          <circle cx="172" cy="108" r="16" fill="#111827"/><circle cx="172" cy="108" r="6" fill="#9CA3AF"/>
          <!-- starlink dish on roof -->
          <rect x="94" y="6" width="44" height="8" rx="2" fill="#E5E7EB" transform="rotate(-12 116 10)"/>
          <rect x="112" y="12" width="4" height="10" fill="#9CA3AF"/>
          <!-- camera person -->
          <circle cx="-2" cy="44" r="0" fill="none"/>
        </g>
        <!-- signal waves from van -->
        <g fill="none" stroke="#22C55E" stroke-width="3" stroke-linecap="round" opacity=".8">
          <path d="M150 196a20 20 0 0 1 22-14"/><path d="M146 186a34 34 0 0 1 36-22"/>
        </g>
        <!-- cloud internet -->
        <g transform="translate(200 54)">
          <path d="M40 70a34 34 0 0 1 8-66 44 44 0 0 1 82 6 30 30 0 0 1 30 54z" fill="#1E293B" stroke="#38BDF8" stroke-width="2"/>
          <text x="86" y="54" text-anchor="middle" font-size="15" font-weight="700" fill="#F8FAFC" font-family="system-ui">Internet</text>
          <text x="86" y="72" text-anchor="middle" font-size="11" fill="#94A3B8" font-family="system-ui">Starlink · 4G · 5G · fibre</text>
        </g>
        <!-- arrows -->
        <path d="M170 170 C 190 120, 200 110, 222 104" fill="none" stroke="#22C55E" stroke-width="3" stroke-dasharray="7 6" marker-end="url(#ah)"/>
        <path d="M362 104 C 392 110, 404 128, 412 160" fill="none" stroke="#22C55E" stroke-width="3" stroke-dasharray="7 6" marker-end="url(#ah)"/>
        <!-- lock -->
        <g transform="translate(276 152)">
          <rect x="0" y="12" width="28" height="22" rx="4" fill="#22C55E"/>
          <path d="M6 12v-5a8 8 0 0 1 16 0v5" fill="none" stroke="#22C55E" stroke-width="3"/>
          <text x="14" y="58" text-anchor="middle" font-size="11" fill="#86EFAC" font-family="system-ui">encrypted</text>
        </g>
        <!-- studio building -->
        <g transform="translate(372 168)">
          <rect x="0" y="22" width="160" height="128" rx="8" fill="#1E293B" stroke="#475569" stroke-width="2"/>
          <path d="M-8 26 L80 -6 L168 26z" fill="#F97316"/>
          <text x="80" y="18" text-anchor="middle" font-size="12" font-weight="700" fill="#fff" font-family="system-ui">STUDIO</text>
          <!-- monitor -->
          <rect x="18" y="40" width="124" height="74" rx="6" fill="#1F2937"/>
          <rect x="26" y="48" width="108" height="46" rx="3" fill="#374151"/>
          <rect x="40" y="56" width="80" height="16" rx="4" fill="#DC2626"/>
          <text x="80" y="68" text-anchor="middle" font-size="9" font-weight="700" fill="#fff" font-family="system-ui">● LIVE FROM VAN</text>
          <rect x="34" y="79" width="40" height="8" rx="2" fill="#4B5563"/>
          <rect x="100" y="80" width="5" height="7" fill="#22C55E"/><rect x="107" y="77" width="5" height="10" fill="#22C55E"/><rect x="114" y="74" width="5" height="13" fill="#22C55E"/><rect x="121" y="71" width="5" height="16" fill="#4B5563"/>
          <rect x="70" y="114" width="20" height="10" fill="#1F2937"/>
          <rect x="54" y="124" width="52" height="5" rx="2" fill="#1F2937"/>
        </g>
        <!-- confirmation bubble -->
        <g transform="translate(26 112)">
          <rect x="0" y="0" width="146" height="46" rx="12" fill="#1E293B" stroke="#F472B6" stroke-width="2"/>
          <path d="M70 46 l10 12 l6-12z" fill="#1E293B" stroke="#F472B6" stroke-width="2"/>
          <rect x="72" y="44" width="16" height="4" fill="#1E293B"/>
          <circle cx="20" cy="23" r="7" fill="#DC2626"/>
          <text x="34" y="20" font-size="11" font-weight="700" fill="#F8FAFC" font-family="system-ui">LIVE AT STUDIO</text>
          <text x="34" y="35" font-size="10" fill="#94A3B8" font-family="system-ui">confirmed by the studio</text>
        </g>
      </svg>
    </figure>
  </div>

  <!-- ============ IN SIMPLE WORDS ============ -->
  <section class="simple">
    <h2>What is Van Link, in simple words?</h2>
    <div class="two">
      <div>
        <p class="big">Think of it as a <b>private, broadcast-quality video line</b> between your van and your studio, carried over the internet instead of a satellite.</p>
        <p>Today, many stations send pictures from an event back to the studio with a satellite dish on a truck. That works, but the dish is heavy, the equipment is expensive, satellite time must be booked, and someone has to point the dish carefully at every location.</p>
        <p>Van Link replaces that with <b>a small program on two computers</b>: one in the van and one in the studio. Your van already has internet (Starlink, a 4G/5G router or MiFi). Van Link uses it to send your live programme straight to the studio, and the studio puts it on air like any other camera.</p>
      </div>
      <div class="explain">
        <div class="ex"><span class="num">1</span><div><b>In the van</b><p>Your cameras and mixer feed OBS as usual. OBS hands the finished programme to Van Link.</p></div></div>
        <div class="ex"><span class="num">2</span><div><b>Over the internet</b><p>Van Link sends it, locked with encryption, directly to your studio computer. No middle server.</p></div></div>
        <div class="ex"><span class="num">3</span><div><b>At the studio</b><p>The picture appears in OBS as a source called "Van Link – your van name". Put it in any scene and go on air.</p></div></div>
      </div>
    </div>
  </section>

  <!-- ============ OLD WAY vs VAN LINK ============ -->
  <section>
    <h2>The old way and the Van Link way</h2>
    <p class="muted intro">The same job, getting live pictures from the field to the studio, done two very different ways.</p>
    <div class="versus">
      <div class="card old">
        <svg viewBox="0 0 300 150" aria-hidden="true">
          <rect width="300" height="150" rx="12" fill="#1F2937"/>
          <rect x="40" y="70" width="150" height="50" rx="6" fill="#4B5563"/><path d="M190 82h34l22 22v16h-56z" fill="#9CA3AF"/>
          <circle cx="70" cy="122" r="12" fill="#374151"/><circle cx="210" cy="122" r="12" fill="#374151"/>
          <ellipse cx="110" cy="40" rx="42" ry="22" fill="#D1D5DB" stroke="#9CA3AF" stroke-width="3" transform="rotate(-25 110 40)"/>
          <rect x="106" y="50" width="8" height="22" fill="#9CA3AF"/>
          <path d="M150 20 l40 -14" stroke="#9CA3AF" stroke-width="3" stroke-dasharray="5 5"/>
          <g transform="translate(236 18)"><rect x="8" y="6" width="12" height="12" rx="2" fill="#9CA3AF"/><rect x="-6" y="9" width="12" height="6" fill="#D1D5DB"/><rect x="22" y="9" width="12" height="6" fill="#D1D5DB"/></g>
        </svg>
        <h3>Satellite uplink truck</h3>
        <ul class="cross">
          <li>Heavy dish and uplink equipment on every trip</li>
          <li>Satellite time booked and paid for each event</li>
          <li>A trained engineer must point the dish</li>
          <li>Takes time to set up before you can go live</li>
          <li>The crew often can't tell if the studio is receiving them</li>
        </ul>
      </div>
      <div class="card new">
        <svg viewBox="0 0 300 150" aria-hidden="true">
          <rect width="300" height="150" rx="12" fill="#0B2E1A"/>
          <rect x="40" y="70" width="150" height="50" rx="6" fill="#2563EB"/><path d="M190 82h34l22 22v16h-56z" fill="#1D4ED8"/>
          <circle cx="70" cy="122" r="12" fill="#111827"/><circle cx="210" cy="122" r="12" fill="#111827"/>
          <rect x="110" y="58" width="44" height="8" rx="2" fill="#E5E7EB" transform="rotate(-10 132 62)"/>
          <rect x="54" y="80" width="46" height="26" rx="3" fill="#2E7D32"/>
          <text x="77" y="98" font-size="11" text-anchor="middle" fill="#fff" font-weight="700" font-family="system-ui">LAPTOP</text>
          <g fill="none" stroke="#2E7D32" stroke-width="3" stroke-linecap="round"><path d="M150 46a16 16 0 0 1 18-10"/><path d="M146 36a30 30 0 0 1 32-18"/></g>
          <text x="248" y="40" font-size="12" text-anchor="middle" fill="#86EFAC" font-weight="700" font-family="system-ui">internet</text>
          <text x="248" y="56" font-size="11" text-anchor="middle" fill="#86EFAC" font-family="system-ui">you already have</text>
        </svg>
        <h3>Van Link</h3>
        <ul class="check">
          <li>A laptop or PC in the van, with OBS</li>
          <li>Uses the internet you already pay for</li>
          <li>Pair once by sending a code on WhatsApp</li>
          <li>Press GO LIVE; nothing to point or book</li>
          <li>The van sees <b>LIVE AT STUDIO</b> only when the studio is really receiving</li>
        </ul>
      </div>
    </div>
  </section>

  <!-- ============ STEP BY STEP ============ -->
  <section id="how">
    <h2>How it works, step by step</h2>
    <p class="muted intro">This is everything your team does, from download to being on air. Pairing (steps 2 and 3) is done once; after that you just press GO LIVE.</p>
    <ol class="flow">
      <li>
        <div class="ico"><svg viewBox="0 0 48 48" aria-hidden="true"><rect x="6" y="8" width="36" height="24" rx="3" fill="none" stroke="currentColor" stroke-width="3"/><path d="M16 40h16M24 32v8" stroke="currentColor" stroke-width="3"/><path d="M24 13v12m-5-5 5 5 5-5" fill="none" stroke="currentColor" stroke-width="3"/></svg></div>
        <div><h3>Install Van Link on two computers</h3><p>Download it from your account page and install it on the <b>van computer</b> and the <b>studio computer</b>. On each one, choose "This is the Van" or "This is the Studio". Van Link finds OBS by itself and adds a Van Link panel inside OBS. One click sets up the Windows firewall for you.</p></div>
      </li>
      <li>
        <div class="ico"><svg viewBox="0 0 48 48" aria-hidden="true"><path d="M24 6a17 17 0 0 0-14.7 25.5L7 41l9.8-2.3A17 17 0 1 0 24 6z" fill="none" stroke="currentColor" stroke-width="3"/><path d="M18 17c0 7 6 13 13 13l2-3-4-2-2 2c-3-1-5-3-6-6l2-2-2-4z" fill="currentColor"/></svg></div>
        <div><h3>The studio sends a pairing code by WhatsApp</h3><p>At the studio, click <b>Add a van</b> and give it a name, for example "Stadium van". Van Link makes a code. Press <b>Share on WhatsApp</b> (or copy it into an SMS or email) and send it to the van crew.</p><img class="stepshot" src="assets/screens/pair-studio-code.jpg" alt="Van Link at the studio after clicking Add a van: the pairing code with Copy code and Share on WhatsApp buttons" loading="lazy"></div>
      </li>
      <li>
        <div class="ico"><svg viewBox="0 0 48 48" aria-hidden="true"><path d="M8 24h24m-8-8 8 8-8 8" fill="none" stroke="currentColor" stroke-width="3"/><path d="M40 10v28" stroke="currentColor" stroke-width="3"/></svg></div>
        <div><h3>The van pastes the code and sends one back</h3><p>In the van, paste the studio's code and click <b>Pair</b>. Van Link shows a reply code; send that back to the studio the same way. The two computers now find each other and connect <b>directly</b>, even when both are on Starlink or mobile data.</p><img class="stepshot" src="assets/screens/pair-van-reply.jpg" alt="Van Link in the van after pairing: Connected to City Studio, ready to go live, with the reply code to send back" loading="lazy"></div>
      </li>
      <li>
        <div class="ico live"><svg viewBox="0 0 48 48" aria-hidden="true"><circle cx="24" cy="24" r="17" fill="none" stroke="currentColor" stroke-width="3"/><circle cx="24" cy="24" r="8" fill="currentColor"/></svg></div>
        <div><h3>Press GO LIVE in the van</h3><p>Van Link starts OBS streaming for you and sends the programme. Within moments the studio screen says <b>Receiving van live</b>, and the van screen turns red: <b>LIVE AT STUDIO</b>.</p><img class="stepshot" src="assets/screens/van-live.jpg" alt="Van Link in the van showing the red LIVE AT STUDIO banner" loading="lazy"></div>
      </li>
      <li>
        <div class="ico"><svg viewBox="0 0 48 48" aria-hidden="true"><rect x="5" y="9" width="38" height="26" rx="3" fill="none" stroke="currentColor" stroke-width="3"/><path d="M12 41h24" stroke="currentColor" stroke-width="3"/><path d="M14 27l7-8 5 5 4-4 6 7" fill="none" stroke="currentColor" stroke-width="3"/></svg></div>
        <div><h3>Put the van on air from OBS at the studio</h3><p>The van appears in the studio's OBS as a source called <b>"Van Link – Stadium van"</b> in a scene called "Van Link". Add it to any scene with your graphics, lower thirds and studio cameras, just like a local camera.</p><img class="stepshot" src="assets/screens/obs-studio-receiving.jpg" alt="OBS Studio at the studio showing the Van Link scene with the source Van Link - Stadium van playing the van's live picture" loading="lazy"></div>
      </li>
    </ol>
    <figure class="wide">
      <img src="assets/how-it-works.png" alt="Diagram: in the van, OBS feeds the Van Link sender; encrypted live video goes to the Van Link receiver at the studio, which feeds an OBS source. Live status, signal and delay flow back to the van.">
      <figcaption><b>The whole picture.</b> Your programme goes one way (van → studio). Status goes both ways every second, so the van always knows whether the studio is receiving it, how strong the signal is, and how much delay there is.</figcaption>
    </figure>
  </section>

  <!-- ============ SCREENS EXPLAINED ============ -->
  <section id="features">
    <h2>What you see on the screens, explained</h2>
    <p class="muted intro">Both ends see the same live information, updated every second, in plain language.</p>
    <div class="shots2">
      <figure><img src="assets/screens/studio-live.jpg" alt="Van Link at the studio: Receiving van live, with signal, upload, delay, missing data and route for both ends" loading="lazy"><figcaption><b>Studio screen.</b> "Receiving van live", plus how the van and the studio are doing.</figcaption></figure>
      <figure><img src="assets/screens/van-live.jpg" alt="Van Link in the van: LIVE AT STUDIO, with signal, sending speed and studio reception" loading="lazy"><figcaption><b>Van screen.</b> Turns red, "LIVE AT STUDIO", only when the studio confirms it is receiving.</figcaption></figure>
    </div>
    <p class="muted small">Real Van Link screens from our test. In that test both computers were side by side, so the delay shows almost zero; on your network the screens show your real figures.</p>
    <div class="glossary">
      <div class="g"><span class="chip red">LIVE AT STUDIO</span><p>Shown on the van screen <b>only</b> when the studio confirms it is receiving your picture. No more phoning the studio to ask "are you getting us?".</p></div>
      <div class="g"><span class="chip"><i class="bars"><i></i><i></i><i></i><i class="off"></i></i> Signal</span><p>Like the bars on a phone: how healthy the connection is right now, for the van and for the studio. Four green bars is excellent.</p></div>
      <div class="g"><span class="chip">Network delay</span><p>How long the picture takes to cross the internet from the van to the studio. On top of that, you choose a small safety buffer in Settings (from 0.3 to 4 seconds); a bigger buffer rides out weak mobile signal better.</p></div>
      <div class="g"><span class="chip">Missing data (repaired)</span><p>The internet sometimes loses small pieces of video. Van Link notices and resends them before they reach the screen, and shows you how much it had to repair.</p></div>
      <div class="g"><span class="chip">Route</span><p>How the two computers are connected, for example "Direct". <b>Encrypted</b> means only your paired studio can open the video.</p></div>
      <div class="g"><span class="chip">Jump to live</span><p>A button at the studio that drops any delay that has built up, so the picture is as fresh as possible.</p></div>
    </div>
  </section>


  <!-- ============ REAL OBS TEST ============ -->
  <section id="obs-test">
    <h2>See it working with OBS Studio</h2>
    <p class="muted intro">These are real screenshots from a test we ran with <b>OBS Studio 30</b> on a van computer and a studio computer, using a test camera picture. Nothing was set up by hand in OBS: Van Link switched on OBS's remote control, filled in the stream settings and created the studio source by itself.</p>
    <div class="obsrun">
      <figure><span class="num">1</span><img src="assets/screens/obs-van-streaming.jpg" alt="OBS Studio in the van while streaming: Stop Streaming button lit, 0 dropped frames, about 2.5 to 2.8 Mbps" loading="lazy">
        <figcaption><b>The van's OBS, streaming.</b> After GO LIVE, Van Link started OBS streaming by itself. The bar at the bottom shows the stream running with <b>0 dropped frames</b>.</figcaption></figure>
      <figure><span class="num">2</span><img src="assets/screens/obs-studio-receiving.jpg" alt="OBS Studio at the studio: scene Van Link with source Van Link - Stadium van showing the same live picture" loading="lazy">
        <figcaption><b>The studio's OBS, receiving.</b> Van Link created the scene <b>"Van Link"</b> with the source <b>"Van Link – Stadium van"</b>. The van's live picture plays there, ready to be put in any scene and sent to air.</figcaption></figure>
    </div>
    <div class="shots2 small3">
      <figure><img src="assets/screens/settings.jpg" alt="Van Link settings: name, OBS connection, delay buffer, account and licence, network test and OBS panel" loading="lazy"><figcaption><b>Settings.</b> Your account and plan, the delay buffer, the network test and the OBS panel, all in one place.</figcaption></figure>
      <figure><img src="assets/screens/obs-panel.jpg" alt="The compact Van Link panel for the studio and the van, as shown docked inside OBS" loading="lazy"><figcaption><b>The Van Link panel for OBS.</b> The same information in a narrow panel that sits inside OBS, for the studio (left) and the van (right).</figcaption></figure>
    </div>
  </section>

  <!-- ============ FEATURES ============ -->
  <section>
    <h2>Why broadcasters choose Van Link</h2>
    <div class="grid feat">
      <div class="card"><div class="fi"><svg viewBox="0 0 48 48" aria-hidden="true" fill="none" stroke="currentColor" stroke-width="3" stroke-linecap="round" stroke-linejoin="round"><path d="M6 16h28m-7-7 7 7-7 7M42 32H14m7-7-7 7 7 7"/></svg></div><h3>Direct van-to-studio</h3><p>Video goes straight from your van to your own studio computer. It does not pass through a cloud or relay server, so nobody else handles your pictures.</p></div>
      <div class="card"><div class="fi"><svg viewBox="0 0 48 48" aria-hidden="true" fill="none" stroke="currentColor" stroke-width="3" stroke-linecap="round" stroke-linejoin="round"><rect x="10" y="20" width="28" height="22" rx="4"/><path d="M16 20v-6a8 8 0 0 1 16 0v6"/></svg></div><h3>Encrypted end to end</h3><p>Every feed is locked with encryption. Only the studio you paired with can open it.</p></div>
      <div class="card"><div class="fi"><svg viewBox="0 0 48 48" aria-hidden="true" fill="none" stroke="currentColor" stroke-width="3" stroke-linecap="round" stroke-linejoin="round"><rect x="6" y="10" width="36" height="24" rx="3"/><path d="M16 40h16"/><circle cx="24" cy="22" r="5"/></svg></div><h3>Inside OBS Studio</h3><p>No new mixing software to learn. The van streams from OBS; the studio receives each van as a normal OBS source, plus a Van Link panel docked in OBS.</p></div>
      <div class="card"><div class="fi"><svg viewBox="0 0 48 48" aria-hidden="true" fill="none" stroke="currentColor" stroke-width="3" stroke-linecap="round" stroke-linejoin="round"><path d="M38 24a14 14 0 1 1-4-10"/><path d="M36 6v8h-8"/></svg></div><h3>Reconnects by itself</h3><p>If the van's internet drops for a moment, Van Link keeps trying and reconnects automatically when the signal returns.</p></div>
      <div class="card"><div class="fi"><svg viewBox="0 0 48 48" aria-hidden="true" fill="none" stroke="currentColor" stroke-width="3" stroke-linecap="round" stroke-linejoin="round"><path d="M10 25l9 9 19-20"/></svg></div><h3>Plain-language messages</h3><p>"Signal is weak, lower the bitrate or move to better coverage" instead of technical error codes. No networking knowledge needed.</p></div>
      <div class="card"><div class="fi"><svg viewBox="0 0 48 48" aria-hidden="true" fill="none" stroke="currentColor" stroke-width="3" stroke-linecap="round" stroke-linejoin="round"><path d="M24 6l5 11 12 1-9 8 3 12-11-6-11 6 3-12-9-8 12-1z"/></svg></div><h3>Program stays first</h3><p>Your main programme feed is always protected first. Extra camera feeds (Pro, rolling out) never push it off air.</p></div>
    </div>
  </section>

  <!-- ============ WHO ============ -->
  <section>
    <h2>Who is it for?</h2>
    <div class="who">
      <div><b>TV stations</b><span>News, events and live programmes from the field</span></div>
      <div><b>Churches and ministries</b><span>Crusades, conventions and outreach broadcasts</span></div>
      <div><b>Sports</b><span>Matches and tournaments from any stadium</span></div>
      <div><b>Event producers</b><span>Concerts, weddings, conferences and launches</span></div>
      <div><b>Schools and campuses</b><span>Graduations, debates and campus TV</span></div>
      <div><b>Government and NGOs</b><span>Town halls, campaigns and live briefings</span></div>
    </div>
  </section>

  <!-- ============ INTERNET ============ -->
  <section>
    <h2>Works on the internet you already have</h2>
    <div class="two">
      <div>
        <p>Van Link works with <b>Starlink</b>, <b>fibre</b>, and <b>4G or 5G</b> routers and MiFi devices (MTN, Airtel, Glo and others). It also works when <b>both</b> the van and the studio are on Starlink or mobile data, which is where many other tools struggle.</p>
        <p>The most important number is the van's <b>upload</b> speed, because the van is sending. As a guide, plan for <b>about 8 Mbps of upload in the van for 1080p</b>. A lower quality setting in OBS needs less.</p>
        <p>Not sure about your connection? Van Link has a built-in <b>Network test</b> (Settings › Network test) that checks your connection and tells you in plain words how well it will work.</p>
      </div>
      <div class="nets">
        <div><b>Starlink</b><span>Satellite internet, in the van or at the studio</span></div>
        <div><b>4G / 5G</b><span>Mobile routers and MiFi</span></div>
        <div><b>Fibre / broadband</b><span>Usually at the studio</span></div>
        <div><b>Tip</b><span>On weak mobile signal, choose a bigger delay buffer and a lower bitrate in OBS. Van Link tells you when to.</span></div>
      </div>
    </div>
  </section>

  <!-- ============ PRICING ============ -->
  <section id="pricing">
    <h2>Simple pricing</h2>
    <p class="muted">Prices in US dollars, paid once a year. No monthly fees.</p>
    <p class="cards-line">We accept all cards: <b>Visa</b>, <b>Mastercard</b>, <b>Verve</b> and <b>American Express</b>, from any country. Secure checkout by Paystack.</p>
    <div class="price">
      <div class="card">
        <h3>Free trial</h3>
        <div class="amount">$0 <small>for <?= (int)$CFG['trial_days'] ?> days</small></div>
        <ul><li><?= (int)$CFG['trial_minutes_per_day'] ?> minutes of broadcasting per 24 hours</li><li>One studio and one van</li><li>All Standard features</li><li>No card needed</li></ul>
        <a class="btn ghost" href="signup.php">Start free trial</a>
      </div>
      <div class="card best">
        <span class="pop">Most popular</span>
        <h3>Standard</h3>
        <div class="amount"><?= money($P['standard']['price']) ?> <small>/ year</small></div>
        <ul><li>Unlimited broadcasting</li><li>One studio and one van</li><li>Two-way monitoring and auto reconnect</li><li>OBS panel and one-click setup</li></ul>
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

  <!-- ============ CHECKLIST ============ -->
  <section>
    <h2>What you need before you start</h2>
    <div class="grid need">
      <div class="card"><h3>1. OBS Studio</h3><p>Free software from obsproject.com, version 28 or newer, on both computers. You probably use it already.</p></div>
      <div class="card"><h3>2. Two Windows computers</h3><p>Windows 10 or 11: one in the van, one in the studio. A graphics card with hardware video support helps at the studio.</p></div>
      <div class="card"><h3>3. Internet at both ends</h3><p>Starlink, fibre, 4G or 5G. About 8 Mbps upload in the van for 1080p.</p></div>
      <div class="card"><h3>4. A phone with WhatsApp</h3><p>Only to send the pairing codes once. SMS or email works too.</p></div>
    </div>
  </section>

  <!-- ============ FAQ ============ -->
  <section id="faq">
    <h2>Questions people ask</h2>
    <div class="faq">
      <details open><summary>Do I need a network engineer to set it up?</summary><p>No. Van Link connects to OBS by itself, sets up the Windows firewall with one click, and finds the way between the van and the studio automatically. If something is wrong, it tells you in plain words what to do.</p></details>
      <details><summary>Does my video go through your servers?</summary><p>No. The video goes directly from your van computer to your studio computer, encrypted. Our website is only used for your account, licence and payments.</p></details>
      <details><summary>What happens if the internet drops during a broadcast?</summary><p>Van Link repairs small losses automatically, and if the connection drops completely it keeps trying and reconnects by itself when the signal comes back. Both screens show exactly what is happening.</p></details>
      <details><summary>How much delay is there?</summary><p>It depends mostly on your internet. You choose a safety buffer in Settings (0.3 to 4 seconds), and both screens show the real network delay live. If delay builds up, the studio can press <b>Jump to live</b>. We recommend testing on your own network during the free trial.</p></details>
      <details><summary>Can I send more than one camera?</summary><p>Van Link sends your OBS Program, the finished mix, which is what most stations put on air. Extra individual camera feeds are coming to the Pro plan, and the Program will always stay the main feed.</p></details>
      <details><summary>Can one studio receive several vans?</summary><p>Yes, with Pro: up to <?= (int)$P['pro']['max_vans'] ?> vans per studio, each arriving as its own OBS source. Standard covers one studio and one van.</p></details>
      <details><summary>How does the free trial work?</summary><p>Create an account, download Van Link, and sign in on both computers. For <?= (int)$CFG['trial_days'] ?> days you can broadcast up to <?= (int)$CFG['trial_minutes_per_day'] ?> minutes in every 24 hours, with all Standard features. No card is needed.</p></details>
      <details><summary>How do I pay, and what happens after?</summary><p>Choose Standard or Pro and pay securely by card through Paystack (prices are in US dollars). As soon as the payment is confirmed you get a <b>registration code</b> by email and on your account page. Enter it in Van Link and your plan is active for one year.</p></details>
      <details><summary>What happens when my plan ends?</summary><p>Van Link stops sending until you renew, whether the computer is online or offline. Renew on the website, enter your new registration code, and you are back on air. Your settings and pairings are kept.</p></details>
      <details><summary>Does it work on Mac?</summary><p>The installers available today are for Windows 10 and 11. Contact us if you need macOS or Linux.</p></details>
    </div>
  </section>

  <!-- ============ FINAL CTA ============ -->
  <section class="final">
    <div>
      <h2>Ready to try it on your own network?</h2>
      <p>Start free for <?= (int)$CFG['trial_days'] ?> days. Install on two computers, pair with a WhatsApp code, and press GO LIVE.</p>
    </div>
    <a class="btn big" href="<?= $start ?>">Start your free trial</a>
  </section>
</main>
<?php page_foot();
