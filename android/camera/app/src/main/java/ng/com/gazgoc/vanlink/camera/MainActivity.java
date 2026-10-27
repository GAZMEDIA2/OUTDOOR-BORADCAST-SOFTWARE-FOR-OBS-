package ng.com.gazgoc.vanlink.camera;

import android.Manifest;
import android.app.Activity;
import android.content.Intent;
import android.content.pm.PackageManager;
import android.graphics.Color;
import android.net.Uri;
import android.os.Bundle;
import android.view.WindowManager;
import android.webkit.PermissionRequest;
import android.webkit.WebChromeClient;
import android.webkit.WebResourceRequest;
import android.webkit.WebSettings;
import android.webkit.WebView;
import android.webkit.WebViewClient;

// Van Link Camera: the website's camera page (camera.php) in its own app, with
// the camera and microphone allowed, the screen kept on and no browser bars.
public class MainActivity extends Activity {
    static final String SITE = "https://vanlink.gazgoc.com.ng";
    static final String HOME = SITE + "/camera.php";
    static final int ASK = 1;

    private WebView web;
    private PermissionRequest pending; // the page's camera request, waiting for Android's permission

    @Override
    protected void onCreate(Bundle saved) {
        super.onCreate(saved);
        getWindow().addFlags(WindowManager.LayoutParams.FLAG_KEEP_SCREEN_ON);
        web = new WebView(this);
        web.setBackgroundColor(Color.rgb(11, 16, 32));
        setContentView(web);

        WebSettings s = web.getSettings();
        s.setJavaScriptEnabled(true);
        s.setDomStorageEnabled(true);
        s.setMediaPlaybackRequiresUserGesture(false);
        s.setUserAgentString(s.getUserAgentString() + " VanLinkCamera/1.0");

        web.setWebViewClient(new WebViewClient() {
            @Override
            public boolean shouldOverrideUrlLoading(WebView v, WebResourceRequest r) {
                Uri u = r.getUrl();
                if (SITE.equals(u.getScheme() + "://" + u.getHost())) return false;
                startActivity(new Intent(Intent.ACTION_VIEW, u)); // other sites open in the browser
                return true;
            }
        });
        web.setWebChromeClient(new WebChromeClient() {
            @Override
            public void onPermissionRequest(PermissionRequest r) {
                if (!r.getOrigin().toString().startsWith(SITE)) { r.deny(); return; }
                if (haveMedia()) { r.grant(r.getResources()); return; }
                pending = r;
                requestPermissions(new String[]{Manifest.permission.CAMERA, Manifest.permission.RECORD_AUDIO}, ASK);
            }
        });

        if (saved != null) web.restoreState(saved);
        else web.loadUrl(startUrl(getIntent()));
        if (!haveMedia()) requestPermissions(new String[]{Manifest.permission.CAMERA, Manifest.permission.RECORD_AUDIO}, ASK);
    }

    private boolean haveMedia() {
        return checkSelfPermission(Manifest.permission.CAMERA) == PackageManager.PERMISSION_GRANTED
            && checkSelfPermission(Manifest.permission.RECORD_AUDIO) == PackageManager.PERMISSION_GRANTED;
    }

    // A camera link (…/camera.php?c=CODE) opens with its code filled in.
    private static String startUrl(Intent i) {
        Uri u = i == null ? null : i.getData();
        return u != null && u.toString().startsWith(HOME) ? u.toString() : HOME;
    }

    @Override
    protected void onNewIntent(Intent i) {
        super.onNewIntent(i);
        if (i.getData() != null) web.loadUrl(startUrl(i));
    }

    @Override
    public void onRequestPermissionsResult(int code, String[] perms, int[] results) {
        if (code != ASK || pending == null) return;
        if (haveMedia()) pending.grant(pending.getResources());
        else pending.deny();
        pending = null;
    }

    @Override
    protected void onSaveInstanceState(Bundle out) {
        super.onSaveInstanceState(out);
        web.saveState(out);
    }

    @Override
    public void onBackPressed() {
        if (web.canGoBack()) web.goBack();
        else super.onBackPressed();
    }
}
