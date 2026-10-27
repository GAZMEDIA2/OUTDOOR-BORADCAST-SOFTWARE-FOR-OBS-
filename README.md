# Van Link

**Live video from your outside broadcast van straight to your studio over the internet, with OBS Studio. No satellite dish, no server in between.**

A product of **Gaz Media Productions**.

## How it works

1. Install Van Link on the **studio** computer and the **van** computer (both also run OBS Studio 28+).
2. The studio clicks **Add a van** and sends the pairing code to the van (WhatsApp, SMS…).
3. The van pastes it, sends back its reply code, and the two connect **directly** to each other.
4. The van clicks **GO LIVE**. OBS's Program output appears at the studio as its own OBS source, encrypted end to end.

Both ends see the same monitoring screen: live status, signal bars, delay, data repaired, route, and a history of drops and reconnections.

## Phone cameras (iPhone and Android, no app to install)

At the studio, click **Add a phone camera**. Van Link creates an OBS Browser Source for it in the "Van Link" scene and shows a code (e.g. `ABCD-EFGH`) and a link to send to the phone. On the phone, open the link (or `vanlink.gazgoc.com.ng/camera.php` and type the code), allow the camera, and the picture and sound appear in OBS.

- The video goes directly from the phone to the studio's OBS (WebRTC, encrypted). The website only passes the connection details (`website/api/cam.php`).
- The phone shows **Live at the studio**, and a red **ON AIR** light when the source is on OBS's Program. It can flip cameras, mute the mic and choose 480p/720p/1080p.
- The camera page can be added to the phone's home screen to open like an app.
- Without Van Link, open `studio-camera.php` in a browser: it makes a camera code and the address to paste into an OBS Browser Source by hand.
- Strict networks at both ends may need a TURN relay: add it under `cam_ice_servers` in the website's `config.php`.

## Connecting directly without a server

Van Link tries every direct route automatically and uses the best one:

| Route | When it works |
|---|---|
| IPv6 direct | Both ends have IPv6 (e.g. Starlink at both ends) |
| Same local network | Testing in the same building |
| IPv4 direct (simultaneous dialling) | Most home, fibre, Starlink and many mobile networks |
| Studio reachable | The studio has a public IP or open port: the van connects with the studio code alone |

If both ends are on the strictest kind of network, no direct route exists. The network test in Settings shows this in advance.

The only outside service Van Link may contact is a public "what is my address?" check (STUN), a few bytes when pairing. Video never passes through it, and it can be switched off in Settings.

## Under the hood

- OBS sends its normal SRT stream to Van Link on the same computer (`127.0.0.1:9710`). Van Link carries the SRT packets unchanged through the direct path, and hands them to the studio's OBS media source (`127.0.0.1:9711+`). SRT's own encryption (AES-128, passphrase from the pairing code) and loss repair stay end to end.
- Van Link reads SRT's acknowledgement packets to measure delay, link capacity and repaired data without decrypting anything.
- Status is exchanged every second over the same path, authenticated with a key from the pairing code.
- OBS is controlled through its built-in remote control (obs-websocket, local to the computer only). Van Link switches it on and reads its password by itself, so operators set nothing up in OBS.
- The screen is a local web page on `http://127.0.0.1:8765/`. It can be docked in OBS via View › Docks › Custom Browser Docks, at `http://127.0.0.1:8765/?dock=1`.

## Building

```
./scripts/build.sh   # tests + dist/VanLink-Setup-<version>.exe, macOS and Linux builds
```

Needs Go 1.22+, NSIS (`makensis`) and `binutils-mingw-w64` for the Windows installer and icon. Pushing a tag such as `v0.1.0` builds everything on GitHub and attaches it to a release.

## Status

Early access, version 0.1. Working: pairing, direct connection, video forwarding, two-way monitoring, OBS control, network test, Windows installer.

Planned: licensing (14-day trial, Standard, Pro), extra camera feeds with Program always primary (Pro), backup over a second internet connection (Pro), automatic bitrate control, optional relay for strict network pairs.
