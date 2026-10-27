package app

import (
	"embed"
	"encoding/json"
	"io/fs"
	"net"
	"net/http"
	"strings"
	"time"
)

//go:embed web
var webFS embed.FS

// Handler serves the screens and the API on this computer only.
func (a *App) Handler() http.Handler {
	mux := http.NewServeMux()
	sub, _ := fs.Sub(webFS, "web")
	mux.Handle("/", http.FileServer(http.FS(sub)))

	mux.HandleFunc("/api/state", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, a.State())
	})
	post := func(path string, f func(r *http.Request) (any, error)) {
		mux.HandleFunc(path, func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodPost || r.Header.Get("X-VanLink") != "1" {
				http.Error(w, "forbidden", http.StatusForbidden)
				return
			}
			out, err := f(r)
			if err != nil {
				w.WriteHeader(http.StatusBadRequest)
				writeJSON(w, map[string]string{"error": err.Error()})
				return
			}
			if out == nil {
				out = map[string]bool{"ok": true}
			}
			writeJSON(w, out)
		})
	}
	type body struct {
		Email    string `json:"email"`
		Password string `json:"password"`
		Role     string `json:"role"`
		Name     string `json:"name"`
		ID       string `json:"id"`
		Code     string `json:"code"`
	}
	read := func(r *http.Request) body {
		var b body
		_ = json.NewDecoder(http.MaxBytesReader(nil, r.Body, 1<<16)).Decode(&b)
		return b
	}
	post("/api/role", func(r *http.Request) (any, error) { return nil, a.SetRole(read(r).Role) })
	post("/api/settings", func(r *http.Request) (any, error) {
		var u SettingsUpdate
		if err := json.NewDecoder(http.MaxBytesReader(nil, r.Body, 1<<16)).Decode(&u); err != nil {
			return nil, err
		}
		return nil, a.UpdateSettings(u)
	})
	post("/api/studio/add", func(r *http.Request) (any, error) {
		code, err := a.AddVan(strings.TrimSpace(read(r).Name))
		return map[string]string{"code": code}, err
	})
	post("/api/studio/reply", func(r *http.Request) (any, error) {
		b := read(r)
		return nil, a.StudioReply(b.ID, b.Code)
	})
	post("/api/studio/jump", func(r *http.Request) (any, error) { return nil, a.JumpToLive(read(r).ID) })
	post("/api/studio/remove", func(r *http.Request) (any, error) { return nil, a.RemoveVan(read(r).ID) })
	post("/api/phone/add", func(r *http.Request) (any, error) { return a.AddPhone(strings.TrimSpace(read(r).Name)) })
	post("/api/phone/remove", func(r *http.Request) (any, error) { return nil, a.RemovePhone(read(r).ID) })
	post("/api/van/pair", func(r *http.Request) (any, error) {
		reply, err := a.PairVan(read(r).Code)
		return map[string]string{"reply": reply}, err
	})
	post("/api/van/unpair", func(r *http.Request) (any, error) { a.UnpairVan(); return nil, nil })
	post("/api/van/golive", func(r *http.Request) (any, error) { return nil, a.GoLive() })
	post("/api/van/stop", func(r *http.Request) (any, error) { return nil, a.StopLive() })
	post("/api/nettest", func(r *http.Request) (any, error) { a.RunNetTest(); return nil, nil })
	post("/api/setup", func(r *http.Request) (any, error) { return a.Setup(), nil })
	post("/api/restore", func(r *http.Request) (any, error) { return a.Restore(), nil })
	post("/api/signin", func(r *http.Request) (any, error) { b := read(r); return nil, a.SignIn(b.Email, b.Password) })
	post("/api/signout", func(r *http.Request) (any, error) { a.SignOut(); return nil, nil })
	post("/api/redeem", func(r *http.Request) (any, error) { return nil, a.Redeem(read(r).Code) })
	post("/api/refresh", func(r *http.Request) (any, error) { return nil, a.RefreshLicence() })
	post("/api/openweb", func(r *http.Request) (any, error) { return nil, a.OpenWeb(read(r).Name) })
	post("/api/show", func(r *http.Request) (any, error) { a.RequestShow(); return nil, nil })
	post("/api/obspanel", func(r *http.Request) (any, error) { return nil, a.InstallOBSPanel() })
	post("/api/quit", func(r *http.Request) (any, error) {
		go func() { time.Sleep(300 * time.Millisecond); a.RequestQuit() }()
		return nil, nil
	})

	return localOnly(mux)
}

// localOnly refuses requests that did not come from this computer (and
// pages that try to reach Van Link through a different host name).
func localOnly(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		host, _, err := net.SplitHostPort(r.Host)
		if err != nil {
			host = r.Host
		}
		if host != "127.0.0.1" && host != "localhost" && host != "::1" && host != "[::1]" {
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}
		w.Header().Set("Cache-Control", "no-store")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		h.ServeHTTP(w, r)
	})
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(v)
}
