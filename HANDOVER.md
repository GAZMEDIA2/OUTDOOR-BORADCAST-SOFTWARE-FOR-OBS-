# Van Link – Project handover (25 Oct 2026)

**Product:** Van Link, by Gaz Media Productions (owner: Engr. Ngwana Joshua, admin email `ioghenevwogaga@gmail.com`).
**What it does:** sends live video from an outside-broadcast (OB) van to the studio over the internet (Starlink, 4G or 5G) with OBS Studio, replacing a satellite dish. Direct van-to-studio, encrypted, with no server in the middle and monitoring at both ends.
**Code:** GitHub `GAZMEDIA2/OUTDOOR-BORADCAST-SOFTWARE-FOR-OBS-` (branch `main`).
**Website:** `vanlink.gazgoc.com.ng` (Namecheap shared hosting, cPanel, server premium52.web-hosting.com).

To continue in a new Claude session, share this file and say: *"Continue the Van Link project from HANDOVER.md, starting with 'Next steps'."* Connect the GitHub repo in that session too.

---

## 1. Decisions made (keep these)

| Topic | Decision |
|---|---|
| Name and brand | **Van Link**, a Gaz Media Productions product. Icon = Gaz Media camera logo + **dark round badge with a red signal symbol** (distinct from Gaz Presenter). Always ask the owner for logos before creating branding. |
| Architecture | Separate **Van Link program** (Go) + **Van Link panel docked in OBS**. Video: OBS → SRT → Van Link → direct internet path → Van Link → OBS Media Source. OBS controlled via its built-in obs-websocket (local only), switched on **automatically** by Van Link. |
| Video sent | OBS **Program output** (Program is always primary). Extra camera feeds = Pro feature, later. |
| Networking | **No relay/server.** Studio computer is the "server". Pairing by codes swapped over WhatsApp/SMS; direct dialling over IPv6, IPv4 hole punching, LAN, reverse direction. Public STUN (address lookup only) allowed, can be switched off. Optional relay can be added later for strict ↔ strict networks. |
| Window | Own app window (WebView2), not a browser tab. Closing the window keeps Van Link running; Settings › Quit stops it. Starts with Windows (background). |
| Plans | **Free trial:** 14 days, **30 min broadcasting per 24 h**, 1 studio + 1 van (2 computers). **Standard:** **$1,100/year**, 1 van, 2 computers. **Pro:** **$1,900/year**, up to 10 vans, 12 computers, extra feeds and backup connection (rolling out). **No monthly fees.** Owner account = unlimited forever. |
| Licensing | Customers sign in in the app with their website email + password. Licence is **signed with Ed25519** by the website; the app verifies it with the built-in public key. **Locks itself when the trial or subscription ends, online or offline**; catches the clock being set back; licence tied to one computer; works offline for 7 days between check-ins. |
| Payments | **Paystack** (owner's existing account, business "Gaz Presenter", live, **NGN**; dollar prices converted at `usd_rate`) + **manual** mobile money/bank (owner approves in admin). **Do NOT change the Paystack webhook**: it serves Gaz Presenter (`gazgoc.com.ng/wp-json/gazpresenter/v1/paystack`). Van Link verifies payments directly instead. |
| Registration codes | Every confirmed payment creates a code `VLK-XXXX-XXXX-XXXX`, emailed and shown on the account page. The customer enters it in the app (or website) to activate. The owner can create codes in admin (cash sales, gifts). Codes are single-use; can be tied to one account. |

---

## 2. What is DONE and working

### The app (Go, `cmd/`, `internal/`)
- **Connection engine** (`internal/link`): pairing codes, direct dialling, SRT forwarding, 1-second two-way status, signal bars, delay, repaired-data %, route, history. **Tested with real encrypted 720p SRT video end to end.**
- **OBS control** (`internal/obsws`): sets the van's stream destination, starts/stops streaming, creates the studio's "Van Link – <van>" Media Source in a "Van Link" scene with low-delay settings, turns off OBS **Stream Delay** automatically, "Jump to live".
- **Automatic OBS connection** (`internal/obsdock/websocket.go`): reads/enables OBS's WebSocket settings itself; no passwords for operators.
- **OBS panel** (`internal/obsdock`): adds the "Van Link" dock to OBS (OBS must be closed). Own installer: `VanLink-OBS-Panel-Setup`.
- **Screens** (`internal/app/web/index.html`): role choice, studio (add van, pairing codes, WhatsApp share, monitor per van, Jump to live), van (pair, GO LIVE/STOP, LIVE AT STUDIO), settings, network test, one-click setup, Quit.
- **Windows installers** (NSIS, `packaging/windows/`), branded; `scripts/build.sh` builds everything; GitHub Actions workflow builds on push/tag.
- **Last installer the owner tested: v0.1.3.** Results: link works; first test had ~1 minute delay, and 0.1.3 contains the fixes (Stream Delay auto-off, low-delay source, Jump to live). **Owner has not yet reported the delay result for 0.1.3.**

### Licensing in the app (Go), partly done
- `internal/licence`: verifies signed licences and decides locked/unlocked (tested: expiry offline, clock rollback, copied or forged licence, owner unlimited).
- `internal/app/account.go`: sign-in, sign-out, redeem registration code, periodic check-in, trial-minute counting (van side), enforcement (stops GO LIVE, stops the stream, pauses forwarding when locked, limits number of vans).
- **NOT DONE: the sign-in / lock / registration-code screens in `web/index.html`.** The API routes exist: `/api/signin`, `/api/signout`, `/api/redeem`, `/api/refresh`, and `/api/state` returns `licence` (ok, reason, message, plan, planName, email, expires, maxVans, trialLeft) and `server`.
- ⚠️ **Do not ship a new installer until those screens exist.** The current code requires a licence, so without the sign-in screen the app would lock with no way to sign in. v0.1.3 (no licensing) is the last safe installer.

### The website (`website/`, PHP 8 + MySQL), code complete, NOT uploaded yet
- Pages:
  - `index.php`: product page, with pricing and images
  - `signup.php`: starts the trial
  - `login.php`, `logout.php`
  - `account.php`: licence, registration codes, downloads, computers, payments, "Check my payment", change password
  - `buy.php`: Paystack or manual payment
  - `paystack-callback.php`
  - `download.php`: login required
  - `admin.php`: owner only. Overview, Customers (with search; full details, licence, computers, payments, trial usage, block/unblock, reset password, set/end licence), Payments (approve/reject/check with Paystack), **Registration codes** (create/cancel), Computers
  - `install.php`: one-time setup; creates tables and the owner account, then **must be deleted**
- App API: `api/login.php`, `api/check.php`, `api/redeem.php`, `api/logout.php`.
- Security: hashed passwords (nobody can see passwords, only reset them), CSRF, login rate limits, the owner email can't be taken by sign-up, secrets blocked from the web, and admin rights only via install.php.
- **Test:** `website/tests/flow.py` with 31 checks; **all pass** on a local SQLite copy.

### Hosting, done in cPanel
- Subdomain **vanlink.gazgoc.com.ng** → folder `/home/gazgdxcg/vanlink.gazgoc.com.ng` (live, https works, currently empty).
- MySQL database **gazgdxcg_vanlink**, user **gazgdxcg_gazmedia** (ALL PRIVILEGES). The password is known only to the owner.

### Documents
- "Van Link – Product Overview" (Claude Docs + Word file) with 4 pictures, pricing, FAQ and roadmap. Labelled early access. Company: Gaz Media Productions.

---

## 3. Secrets: where they are (never put them in GitHub)
- **Licence secret key:** inside the private `config.php` sent to the owner in chat (25 Oct). **Keep it safe.** If it's lost, a new key pair must be made and the app rebuilt with the new public key (`internal/licence/licence.go`, `PublicKey`).
- Public key (safe to share, already in the app): `YoapNMRwLFAJmRy0FuNK0RFheASh2ZjnFgvsRNoddlU=`
- Database password: owner only. Paystack secret key: owner pastes it into `config.php` (`paystack_secret_key`).

---

## 4. Next steps (in order)

1. **Upload the website**
   - In cPanel File Manager, upload the contents of `website/` (the zip sent in chat) into `/home/gazgdxcg/vanlink.gazgoc.com.ng/`.
   - Upload the private `config.php` next to `index.php`, and edit in it:
     - `db_pass` (database password)
     - `paystack_secret_key` (from the Paystack dashboard › API Keys)
     - `usd_rate`
     - `manual_payment_text` (MoMo/bank details)
     - `support_email`
   - Open `https://vanlink.gazgoc.com.ng/install.php`, create the owner account with `ioghenevwogaga@gmail.com`, then **delete install.php**.
   - Check sign-up, log in, and the admin page. Make a small Paystack test, or use `sk_test_` keys first.
2. **Finish the app's sign-in screens** (`internal/app/web/index.html`):
   - **Signed out:** sign-in form (email, password) + "Create free trial account" link (opens the website).
   - **Locked** (expired, offline, clock, invalid, device): message + registration-code box + "Check again" (`/api/refresh`) + "Buy or renew" (opens `account.php` in the browser) + "Sign out".
   - Header pill showing the plan; trial minutes left; Settings › Account (email, plan, expiry, code box, sign out).
   - Add an `/api/openweb` route so website links open in the normal browser from the app window.
   - The OBS panel uses the same page, so it locks too (as the owner wants).
3. Rebuild (`scripts/build.sh`, bump to v0.2.0), put the installers in the website's `downloads/` folder, test sign-in against the live site, and send them to the owner.
4. **Owner's open test:** the delay with v0.1.3 (clap test, the Delay value on screen).
5. Later (Pro): extra camera feeds (OBS helper plugin + per-feed outputs, Program always primary), backup second internet connection (SRT bonding), automatic bitrate control, optional relay, code signing certificate (to remove the Windows SmartScreen warning), password-reset emails.

## 5. Useful facts
- Owner's Namecheap account: gazmediaproductions; hosting plan Stellar Plus; other sites on the same hosting: gazgoc.com.ng (WordPress, Gaz Presenter), overlays.gazgoc.com.ng, demideadeenterprise.com.ng, dijecare.org.ng. **Don't touch them.**
- Local ports: Van Link screen `127.0.0.1:8765` (dock URL `/?dock=1`); van OBS → `127.0.0.1:9710`; studio OBS sources `127.0.0.1:9711+`; studio internet ports `47100+` (one per van).
- Build needs: Go 1.22+ (deps vendored), NSIS, mingw binutils + gcc-mingw (for the icon). Tests need ffmpeg.
