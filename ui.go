package main

import (
	"bufio"
	"fmt"
	"html/template"
	"net/http"
	"os"
	"sort"
	"strings"
	"sync"
	"time"

	"marketplace/apis"
)

var runMu sync.Mutex
var frontendLogs = &logStore{lines: make([]string, 0, 300)}

type logStore struct {
	mu    sync.Mutex
	lines []string
}

func (l *logStore) Write(p []byte) (int, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	for _, line := range strings.Split(strings.TrimRight(string(p), "\n"), "\n") {
		if strings.TrimSpace(line) == "" {
			continue
		}
		l.lines = append(l.lines, stripANSI(line))
	}
	if len(l.lines) > 300 {
		l.lines = l.lines[len(l.lines)-300:]
	}
	return len(p), nil
}

func (l *logStore) Text() string {
	l.mu.Lock()
	defer l.mu.Unlock()
	return strings.Join(l.lines, "\n")
}

func stripANSI(s string) string {
	var b strings.Builder
	for i := 0; i < len(s); i++ {
		if s[i] == 0x1b && i+1 < len(s) && s[i+1] == '[' {
			i += 2
			for i < len(s) && s[i] != 'm' {
				i++
			}
			continue
		}
		b.WriteByte(s[i])
	}
	return b.String()
}

func startUI(aiClient *apis.OpenAI, fbPoster *apis.FacebookPoster, baseDescription string, tags []string) {
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		data := map[string]string{
			"Headless":      envValue("HEADLESS", "true"),
			"DryRun":        envValue("DRY_RUN", "false"),
			"GroupKeywords": groupKeywordValue(),
			"OpenAIKey":     os.Getenv("OPENAI_API_KEY"),
			"CUser":         os.Getenv("FB_C_USER"),
			"Datr":          os.Getenv("FB_DATR"),
			"Fr":            os.Getenv("FB_FR"),
			"Presence":      os.Getenv("FB_PRESENCE"),
			"Sb":            os.Getenv("FB_SB"),
			"Wd":            os.Getenv("FB_WD"),
			"Xs":            os.Getenv("FB_XS"),
			"PriceRange":    "400-600",
			"Schedule":      "07:00, 12:00, 17:00",
			"Now":           time.Now().Format("15:04:05"),
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_ = uiTemplate.Execute(w, data)
	})

	http.HandleFunc("/logs", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		_, _ = w.Write([]byte(frontendLogs.Text()))
	})

	http.HandleFunc("/settings", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		updates := map[string]string{
			"OPENAI_API_KEY": r.FormValue("OPENAI_API_KEY"),
			"HEADLESS":       boolValue(r.FormValue("HEADLESS")),
			"DRY_RUN":        boolValue(r.FormValue("DRY_RUN")),
			"GROUP_KEYWORDS": r.FormValue("GROUP_KEYWORDS"),
			"FB_C_USER":      r.FormValue("FB_C_USER"),
			"FB_DATR":        r.FormValue("FB_DATR"),
			"FB_FR":          r.FormValue("FB_FR"),
			"FB_PRESENCE":    r.FormValue("FB_PRESENCE"),
			"FB_SB":          r.FormValue("FB_SB"),
			"FB_WD":          r.FormValue("FB_WD"),
			"FB_XS":          r.FormValue("FB_XS"),
		}
		if err := saveEnv(updates); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		for key, value := range updates {
			_ = os.Setenv(key, value)
		}
		syncPosterFromEnv(fbPoster)
		cliLog("UI", "32", "Settings saved to .env")
		http.Redirect(w, r, "/", http.StatusSeeOther)
	})

	http.HandleFunc("/run", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		go runFromUI(aiClient, fbPoster, baseDescription, tags, false)
		http.Redirect(w, r, "/", http.StatusSeeOther)
	})

	http.HandleFunc("/dry-run", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		go runFromUI(aiClient, fbPoster, baseDescription, tags, true)
		http.Redirect(w, r, "/", http.StatusSeeOther)
	})

	go func() {
		cliLog("UI", "36", "Dashboard running at http://localhost:8080")
		if err := http.ListenAndServe(":8080", nil); err != nil {
			cliLog("UI", "31", "Server stopped: %v", err)
		}
	}()
}

func runFromUI(aiClient *apis.OpenAI, fbPoster *apis.FacebookPoster, baseDescription string, tags []string, dryRun bool) {
	runMu.Lock()
	defer runMu.Unlock()
	syncPosterFromEnv(fbPoster)
	oldDryRun := os.Getenv("DRY_RUN")
	if dryRun {
		_ = os.Setenv("DRY_RUN", "true")
	} else {
		_ = os.Unsetenv("DRY_RUN")
	}
	runPosting(aiClient, fbPoster, baseDescription, tags)
	if oldDryRun == "" {
		_ = os.Unsetenv("DRY_RUN")
	} else {
		_ = os.Setenv("DRY_RUN", oldDryRun)
	}
}

func syncPosterFromEnv(fbPoster *apis.FacebookPoster) {
	fbPoster.CUser = os.Getenv("FB_C_USER")
	fbPoster.Datr = os.Getenv("FB_DATR")
	fbPoster.Fr = os.Getenv("FB_FR")
	fbPoster.Presence = os.Getenv("FB_PRESENCE")
	fbPoster.Sb = os.Getenv("FB_SB")
	fbPoster.Wd = os.Getenv("FB_WD")
	fbPoster.Xs = os.Getenv("FB_XS")
	fbPoster.Headless = os.Getenv("HEADLESS") != "false"
}

func envValue(key, fallback string) string {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	return value
}

func boolValue(value string) string {
	if value == "true" || value == "on" || value == "1" {
		return "true"
	}
	return "false"
}

func groupKeywordValue() string {
	value := strings.TrimSpace(os.Getenv("GROUP_KEYWORDS"))
	if value == "" {
		return "jual,beli,jogja"
	}
	return value
}

func saveEnv(updates map[string]string) error {
	env := map[string]string{}
	file, err := os.Open(".env")
	if err == nil {
		scanner := bufio.NewScanner(file)
		for scanner.Scan() {
			line := strings.TrimSpace(scanner.Text())
			if line == "" || strings.HasPrefix(line, "#") {
				continue
			}
			parts := strings.SplitN(line, "=", 2)
			if len(parts) == 2 {
				env[strings.TrimSpace(parts[0])] = strings.TrimSpace(parts[1])
			}
		}
		if scanErr := scanner.Err(); scanErr != nil {
			_ = file.Close()
			return scanErr
		}
		_ = file.Close()
	} else if !os.IsNotExist(err) {
		return err
	}
	for key, value := range updates {
		env[key] = strings.TrimSpace(value)
	}
	keys := make([]string, 0, len(env))
	for key := range env {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	var sb strings.Builder
	for _, key := range keys {
		sb.WriteString(fmt.Sprintf("%s=%s\n", key, env[key]))
	}
	return os.WriteFile(".env", []byte(sb.String()), 0600)
}

var uiTemplate = template.Must(template.New("ui").Parse(`<!doctype html>
<html lang="id">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>Siswanto Aki Dashboard</title>
<style>
:root{color-scheme:dark;--bg:#050505;--panel:rgba(255,255,255,.055);--panel2:rgba(255,255,255,.085);--ink:#f7f2e8;--muted:#9c9589;--line:rgba(255,255,255,.11);--amber:#e9b86f;--green:#9ad6a5;--blue:#a7c7ff;--red:#ff9c9c}*{box-sizing:border-box}body{margin:0;min-height:100dvh;background:radial-gradient(760px 460px at 8% 2%,rgba(233,184,111,.16),transparent 58%),radial-gradient(720px 420px at 92% 4%,rgba(167,199,255,.12),transparent 60%),var(--bg);color:var(--ink);font-family:Geist,"Plus Jakarta Sans",system-ui,sans-serif}body:before{content:"";position:fixed;inset:0;pointer-events:none;background:linear-gradient(rgba(255,255,255,.022) 1px,transparent 1px),linear-gradient(90deg,rgba(255,255,255,.022) 1px,transparent 1px);background-size:44px 44px;mask-image:linear-gradient(to bottom,black,transparent 88%)}main{width:min(1440px,100%);margin:0 auto;padding:22px;display:grid;grid-template-columns:260px 1fr;gap:18px;overflow-x:hidden}.sidebar{position:sticky;top:22px;height:calc(100dvh - 44px);border:1px solid var(--line);border-radius:34px;background:rgba(255,255,255,.05);backdrop-filter:blur(24px);padding:18px;display:flex;flex-direction:column;justify-content:space-between}.brand{border-radius:24px;background:rgba(0,0,0,.35);padding:18px}.brand b{display:block;font-size:15px;letter-spacing:.08em;text-transform:uppercase}.brand span{display:block;margin-top:8px;color:var(--muted);font-size:13px;line-height:1.5}.nav{display:grid;gap:8px;margin-top:24px}.nav a{color:var(--muted);text-decoration:none;padding:12px 14px;border-radius:18px}.nav a:first-child,.nav a:hover{background:rgba(255,255,255,.08);color:var(--ink)}.sidefoot{color:var(--muted);font-size:12px;line-height:1.6}.content{display:grid;gap:18px}.topbar{border:1px solid var(--line);border-radius:34px;background:rgba(255,255,255,.045);padding:18px;display:flex;align-items:center;justify-content:space-between;gap:16px}.topbar h1{margin:0;font-size:clamp(28px,4vw,52px);line-height:.95;letter-spacing:-.06em}.topbar p{margin:8px 0 0;color:var(--muted)}.actions{display:flex;gap:10px;flex-wrap:wrap}.btn{border:0;border-radius:999px;min-height:48px;padding:8px 8px 8px 18px;display:inline-flex;align-items:center;gap:14px;font-weight:750;cursor:pointer;transition:transform .7s cubic-bezier(.32,.72,0,1),background .7s cubic-bezier(.32,.72,0,1)}.btn:active{transform:scale(.98)}.btn strong{width:32px;height:32px;border-radius:999px;display:grid;place-items:center;transition:transform .7s cubic-bezier(.32,.72,0,1)}.btn:hover strong{transform:translateX(3px) translateY(-1px) scale(1.05)}.primary{background:var(--ink);color:#070707}.primary strong{background:#111;color:var(--ink)}.secondary{background:rgba(255,255,255,.08);color:var(--ink);border:1px solid var(--line)}.secondary strong{background:rgba(255,255,255,.12)}.grid{display:grid;grid-template-columns:repeat(12,1fr);grid-auto-flow:dense;gap:18px}.shell{border:1px solid var(--line);border-radius:32px;background:rgba(255,255,255,.042);padding:7px}.card{min-height:100%;border-radius:25px;background:linear-gradient(145deg,var(--panel),var(--panel2));box-shadow:inset 0 1px 1px rgba(255,255,255,.13);padding:24px;position:relative;overflow:hidden}.span3{grid-column:span 3}.span4{grid-column:span 4}.span5{grid-column:span 5}.span7{grid-column:span 7}.span8{grid-column:span 8}.span12{grid-column:span 12}.label{color:var(--muted);font-size:12px;text-transform:uppercase;letter-spacing:.16em}.value{font-size:34px;letter-spacing:-.05em;margin-top:10px}.ok{color:var(--green)}.warn{color:var(--amber)}.muted{color:var(--muted);line-height:1.6}.rows{display:grid;gap:10px}.row{display:flex;justify-content:space-between;gap:12px;border-bottom:1px solid var(--line);padding:12px 0;color:var(--muted)}.row b{color:var(--ink);text-align:right}.settings{display:grid;grid-template-columns:repeat(2,1fr);gap:14px}.field{display:grid;gap:8px}.field.full{grid-column:1/-1}label{font-size:12px;text-transform:uppercase;letter-spacing:.14em;color:var(--muted)}input,textarea,select{width:100%;border:1px solid var(--line);outline:none;border-radius:18px;background:rgba(0,0,0,.3);color:var(--ink);padding:13px 14px;font:inherit;transition:border-color .7s cubic-bezier(.32,.72,0,1),background .7s cubic-bezier(.32,.72,0,1)}textarea{min-height:90px;resize:vertical}input:focus,textarea:focus,select:focus{border-color:rgba(233,184,111,.55);background:rgba(0,0,0,.45)}.savebar{display:flex;justify-content:flex-end;margin-top:16px}.small{font-size:12px;color:var(--muted);line-height:1.55}.split{display:grid;grid-template-columns:1fr 1fr;gap:14px}.code{font-family:"SF Mono",ui-monospace,monospace;font-size:12px;color:var(--blue);word-break:break-all}.logbox{height:360px;overflow:auto;border:1px solid var(--line);border-radius:22px;background:rgba(0,0,0,.36);padding:16px;font-family:"SF Mono",ui-monospace,monospace;font-size:12px;line-height:1.65;color:#d9d3c8;white-space:pre-wrap}.logbox .ai{color:#d9b4ff}.logbox .form{color:#a7c7ff}.logbox .group{color:#9ad6a5}.logbox .publish{color:#e9b86f}.logbox .err{color:#ff9c9c}@media(max-width:980px){main{grid-template-columns:1fr}.sidebar{position:relative;height:auto}.grid{grid-template-columns:1fr}.span3,.span4,.span5,.span7,.span8,.span12{grid-column:span 1}.settings,.split{grid-template-columns:1fr}.topbar{align-items:flex-start;flex-direction:column}}@media(max-width:640px){main{padding:14px}.topbar,.sidebar,.shell{border-radius:24px}.card{padding:18px}.actions{width:100%}.btn{width:100%;justify-content:space-between}}</style>
</head>
<body>
<main>
  <aside class="sidebar">
    <div>
      <div class="brand"><b>Siswanto Aki</b><span>Facebook Marketplace automation dashboard</span></div>
      <nav class="nav"><a href="#overview">Overview</a><a href="#logs">Logs</a><a href="#settings">Settings</a><a href="#debug">Debug</a></nav>
    </div>
    <div class="sidefoot">Local only<br>http://localhost:8080<br>{{.Now}}</div>
  </aside>
  <section class="content">
    <header class="topbar" id="overview">
      <div><h1>Marketplace Control Center</h1><p>Run posting, test dry-run, and update environment settings from one local dashboard.</p></div>
      <div class="actions"><form method="post" action="/run"><button class="btn primary" type="submit">Run Post <strong>↗</strong></button></form><form method="post" action="/dry-run"><button class="btn secondary" type="submit">Dry Run <strong>→</strong></button></form></div>
    </header>
    <section class="grid">
      <div class="shell span3"><div class="card"><div class="label">Price</div><div class="value">{{.PriceRange}}</div><p class="small">Random per post</p></div></div>
      <div class="shell span3"><div class="card"><div class="label">Schedule</div><div class="value">3x</div><p class="small">{{.Schedule}}</p></div></div>
      <div class="shell span3"><div class="card"><div class="label">Headless</div><div class="value {{if eq .Headless "false"}}warn{{else}}ok{{end}}">{{.Headless}}</div><p class="small">Browser visibility</p></div></div>
      <div class="shell span3"><div class="card"><div class="label">Dry Run Env</div><div class="value {{if eq .DryRun "true"}}warn{{else}}ok{{end}}">{{.DryRun}}</div><p class="small">Global safety flag</p></div></div>
      <div class="shell span5"><div class="card"><h2>Runtime</h2><div class="rows"><div class="row"><span>Group keywords</span><b>{{.GroupKeywords}}</b></div><div class="row"><span>OpenAI key</span><b>{{if .OpenAIKey}}set{{else}}empty{{end}}</b></div><div class="row"><span>FB c_user</span><b>{{if .CUser}}set{{else}}empty{{end}}</b></div></div></div></div>
      <div class="shell span7" id="debug"><div class="card"><h2>Debug Files</h2><div class="split"><div><div class="label">Form snapshot</div><p class="code">debug_filled.png</p></div><div><div class="label">Group snapshot</div><p class="code">debug_groups_selected.png</p></div></div><p class="muted">Dry-run clicks Next, selects matching groups, saves screenshots, then stops before Publish.</p></div></div>
      <div class="shell span12" id="logs"><div class="card"><h2>Live Logs</h2><p class="muted">Same runtime logs as terminal, refreshed automatically.</p><pre class="logbox" id="logbox">Waiting for logs...</pre></div></div>
      <div class="shell span12" id="settings"><div class="card"><h2>Settings</h2><p class="muted">Save injects values into .env and current runtime environment. OpenAI client still needs app restart if API key changes.</p><form method="post" action="/settings"><div class="settings"><div class="field"><label>OPENAI_API_KEY</label><input name="OPENAI_API_KEY" value="{{.OpenAIKey}}" autocomplete="off"></div><div class="field"><label>GROUP_KEYWORDS</label><input name="GROUP_KEYWORDS" value="{{.GroupKeywords}}"></div><div class="field"><label>HEADLESS</label><select name="HEADLESS"><option value="true" {{if eq .Headless "true"}}selected{{end}}>true</option><option value="false" {{if eq .Headless "false"}}selected{{end}}>false</option></select></div><div class="field"><label>DRY_RUN</label><select name="DRY_RUN"><option value="false" {{if eq .DryRun "false"}}selected{{end}}>false</option><option value="true" {{if eq .DryRun "true"}}selected{{end}}>true</option></select></div><div class="field"><label>FB_C_USER</label><input name="FB_C_USER" value="{{.CUser}}" autocomplete="off"></div><div class="field"><label>FB_XS</label><input name="FB_XS" value="{{.Xs}}" autocomplete="off"></div><div class="field"><label>FB_DATR</label><input name="FB_DATR" value="{{.Datr}}" autocomplete="off"></div><div class="field"><label>FB_FR</label><input name="FB_FR" value="{{.Fr}}" autocomplete="off"></div><div class="field"><label>FB_PRESENCE</label><input name="FB_PRESENCE" value="{{.Presence}}" autocomplete="off"></div><div class="field"><label>FB_SB</label><input name="FB_SB" value="{{.Sb}}" autocomplete="off"></div><div class="field"><label>FB_WD</label><input name="FB_WD" value="{{.Wd}}" autocomplete="off"></div></div><div class="savebar"><button class="btn primary" type="submit">Save Settings <strong>✓</strong></button></div></form></div></div>
    </section>
  </section>
</main>
<script>
const logbox=document.getElementById('logbox');
function paint(line){return line.replaceAll('&','&amp;').replaceAll('<','&lt;').replaceAll('>','&gt;').replace(/\[(AI)\]/g,'<span class="ai">[$1]</span>').replace(/\[(FORM|MEDIA)\]/g,'<span class="form">[$1]</span>').replace(/\[(GROUP)\]/g,'<span class="group">[$1]</span>').replace(/\[(PUBLISH|SCHEDULE|UI|COMMAND)\]/g,'<span class="publish">[$1]</span>').replace(/(Failed|Error|invalid|expired)/gi,'<span class="err">$1</span>')}
async function loadLogs(){const res=await fetch('/logs',{cache:'no-store'});const text=await res.text();logbox.innerHTML=text?text.split('\n').map(paint).join('\n'):'Waiting for logs...';logbox.scrollTop=logbox.scrollHeight}
loadLogs();setInterval(loadLogs,1200);
</script>
</body>
</html>`))
