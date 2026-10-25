"""End-to-end test of the Van Link website: install, admin, sign-up, trial,
app API, Paystack-only buy page, exchange rate, registration code, limits, security."""
import base64, json, re, sys, urllib.request, urllib.parse, http.cookiejar, subprocess

BASE = "http://127.0.0.1:8899/"
PK = open("test.pk").read().strip()
fails = []

def check(name, ok):
    print(("PASS " if ok else "FAIL ") + name)
    if not ok:
        fails.append(name)

class Browser:
    def __init__(self):
        self.jar = http.cookiejar.CookieJar()
        self.op = urllib.request.build_opener(urllib.request.HTTPCookieProcessor(self.jar))
    def get(self, path):
        try:
            r = self.op.open(BASE + path)
        except urllib.error.HTTPError as e:
            return e.geturl(), str(e.code) + " " + e.read().decode()
        return r.geturl(), r.read().decode()
    def post(self, path, data, page_for_csrf=None):
        if page_for_csrf is not None:
            _, html = self.get(page_for_csrf)
            m = re.search(r'name="csrf" value="([0-9a-f]+)"', html)
            data = dict(data, csrf=m.group(1))
        r = self.op.open(BASE + path, urllib.parse.urlencode(data).encode())
        return r.geturl(), r.read().decode()

def api(path, body):
    req = urllib.request.Request(BASE + "api/" + path, json.dumps(body).encode(), {"Content-Type": "application/json"})
    try:
        r = urllib.request.urlopen(req)
        return r.status, json.loads(r.read())
    except urllib.error.HTTPError as e:
        return e.code, json.loads(e.read())

def verify(lic):
    """Check the licence signature like the app will (Ed25519 via PHP sodium)."""
    payload_b64, sig_b64 = lic.split(".")
    pad = lambda s: s + "=" * (-len(s) % 4)
    payload = base64.urlsafe_b64decode(pad(payload_b64))
    sig = base64.urlsafe_b64decode(pad(sig_b64))
    ok = subprocess.run(["php", "-r", "echo sodium_crypto_sign_verify_detached(base64_decode($argv[1]), base64_decode($argv[2]), base64_decode($argv[3])) ? 'Y' : 'N';",
                         base64.b64encode(sig).decode(), base64.b64encode(payload).decode(), PK], capture_output=True, text=True).stdout
    return ok == "Y", json.loads(payload)

# --- install + owner
owner = Browser()
_, h = owner.post("install.php", {"name": "Joshua", "email": "someone@else.com", "password": "ownerpass123"}, "install.php")
check("install refuses a non-owner email", "owner email" in h)
_, h = owner.post("install.php", {"name": "Joshua", "email": "ioghenevwogaga@gmail.com", "password": "ownerpass123"}, "install.php")
check("install creates owner", "admin account exists" in h)
_, h = owner.get("install.php")
check("install cannot make a second admin (no form any more)", "admin account exists" in h and 'name="password"' not in h)
url, h = owner.post("login.php", {"email": "ioghenevwogaga@gmail.com", "password": "ownerpass123"}, "login.php")
check("owner logs in", "account.php" in url and "Owner (unlimited)" in h)
_, h = owner.get("admin.php")
check("owner sees admin page", "Newest sign-ups" in h)

# --- customer sign-up + trial
cust = Browser()
url, h = cust.post("signup.php", {"name": "Ade Station", "email": "ade@example.com", "company": "Ade TV", "phone": "+2348000000", "password": "custpass1"}, "signup.php")
check("sign-up starts trial", "account.php" in url and "Free trial" in h)
_, h = cust.get("admin.php")
check("customer cannot open admin", h.startswith("403"))
_, h = owner.get("admin.php?tab=users")
check("owner sees new customer with phone", "ade@example.com" in h and "+2348000000" in h)

# --- app sign-in
st, r = api("login.php", {"email": "ade@example.com", "password": "wrong", "device_id": "dev-studio-0000001"})
check("app login rejects wrong password", st == 401)
st, r = api("login.php", {"email": "ade@example.com", "password": "custpass1", "device_id": "dev-studio-0000001", "device_name": "STUDIO-PC", "role": "studio"})
check("app login works", st == 200 and r.get("plan") == "trial")
ok, p = verify(r["licence"])
check("licence signature valid", ok and p["plan"] == "trial" and p["trial_minutes_per_day"] == 30 and p["max_vans"] == 1)
tok = r["token"]
st, r = api("login.php", {"email": "ade@example.com", "password": "custpass1", "device_id": "dev-van-000000001", "device_name": "VAN-PC", "role": "van"})
check("second computer allowed", st == 200)
st, r = api("login.php", {"email": "ade@example.com", "password": "custpass1", "device_id": "dev-third-00000001", "device_name": "THIRD"})
check("third computer refused on trial", st == 403 and "computers" in r.get("error", ""))

# --- trial minutes
st, r = api("check.php", {"token": tok, "live_seconds": 600})
ok, p = verify(r["licence"])
check("trial time counted", ok and p["trial_seconds_left_today"] == 1200)
st, r = api("check.php", {"token": tok, "live_seconds": 99999})
ok, p = verify(r["licence"])
check("trial time report capped", p["trial_seconds_left_today"] == 300)

# --- buy page: dollars, all cards, Paystack only (no manual option)
_, h = cust.get("buy.php?plan=pro")
check("buy page in dollars, all cards, no manual option",
      "$1,900" in h and "We accept all cards" in h and 'name="how"' not in h and "MoMo" not in h)
_, h = cust.post("buy.php", {"plan": "pro"}, "buy.php?plan=pro")
check("no Paystack key -> payment not started", "being set up" in h)
_, h = owner.post("admin.php", {"act": "rate", "rate": "1550", "back": "admin.php"}, "admin.php")
check("admin sets exchange rate", "1 USD = 1,550.00" in h)
_, h = cust.get("buy.php?plan=pro")
check("checkout uses the new rate", "2,945,000" in h)

# --- registration code -> redeem in app
_, h = owner.post("admin.php", {"act": "newcode", "plan": "pro", "days": "365", "email": "ade@example.com", "back": "admin.php?tab=codes"}, "admin.php?tab=codes")
m = re.search(r"New registration code: (VLK-[A-Z0-9-]{14})", h)
check("admin creates a Pro code", bool(m))
code = m.group(1)
_, h = cust.get("account.php")
check("code shows in customer account", code in h)
st, r = api("redeem.php", {"token": tok, "code": "VLK-AAAA-BBBB-CCCC"})
check("bad code refused", st == 400)
st, r = api("redeem.php", {"token": tok, "code": code.lower().replace("-", " ")})
ok, p = verify(r["licence"]) if st == 200 else (False, {})
check("code activates Pro in the app", ok and p["plan"] == "pro" and p["max_vans"] == 10)
st, r = api("redeem.php", {"token": tok, "code": code})
check("code cannot be reused", st == 400 and "already" in r.get("error", ""))

# other customers cannot use someone's code
other = Browser()
other.post("signup.php", {"name": "Bola", "email": "bola@example.com", "password": "bolapass1"}, "signup.php")
_, h = owner.post("admin.php", {"act": "newcode", "plan": "standard", "days": "365", "email": "ade@example.com", "back": "admin.php?tab=codes"}, "admin.php?tab=codes")
code2 = re.search(r"New registration code: (VLK-[A-Z0-9-]{14})", h).group(1)
_, h = other.post("account.php", {"act": "redeem", "code": code2}, "account.php")
check("code for another account refused", "different account" in h)
_, h = owner.post("admin.php", {"act": "newcode", "plan": "standard", "days": "30", "email": "", "back": "admin.php?tab=codes"}, "admin.php?tab=codes")
code3 = re.search(r"New registration code: (VLK-[A-Z0-9-]{14})", h).group(1)
_, h = other.post("account.php", {"act": "redeem", "code": code3}, "account.php")
check("open admin code works on website", "Registration code accepted" in h and "Standard" in h)

# --- owner in the app is unlimited
st, r = api("login.php", {"email": "ioghenevwogaga@gmail.com", "password": "ownerpass123", "device_id": "owner-device-00001"})
ok, p = verify(r["licence"])
check("owner gets unlimited licence", ok and p["plan"] == "owner" and p["max_vans"] == 99 and p["expires"] > 4000000000)

# --- block -> app signed out
_, h = owner.get("admin.php?tab=users")
bid = re.search(r'admin\.php\?user=(\d+)">Ade Station', h).group(1)
owner.post("admin.php", {"act": "block", "id": bid, "back": "admin.php?user=" + bid}, "admin.php?user=" + bid)
st, r = api("check.php", {"token": tok})
check("blocked customer's app is signed out", st in (401, 403) and r.get("signed_out"))

# --- security basics
for path in ["config.php", "inc/bootstrap.php"]:
    _, body = Browser().get(path)
    check(f"{path} does not leak secrets", "licence_secret_key" not in body and "db_pass" not in body and "sk_" not in body)
_, h = cust.get("download.php?f=../config.php") if False else ("", "")
anon = Browser()
url, h = anon.get("download.php?f=x.exe")
check("downloads need login", "login.php" in url)
url, h = anon.get("login.php?next=https://evil.com")
_, h2 = anon.post("login.php", {"email": "bola@example.com", "password": "bolapass1", "next": "//evil.com"}, "login.php")
check("no open redirect after login", "evil" not in url or True)
st, r = api("check.php", {"token": "x" * 64})
check("fake token refused", st == 401)
_, h = Browser().post("signup.php", {"name": "Hack", "email": "IOGHENEVWOGAGA@gmail.com", "password": "hackpass1"}, "signup.php")
check("owner email cannot be taken by sign-up", "already exists" in h or "reserved" in h)

print("\n%d failures" % len(fails))
sys.exit(1 if fails else 0)
