// gateway は ALB から見える唯一のコンテナ。同じタスク内の api を localhost で
// 呼び、worker が書いた状態を読む。「3 コンテナが本当に動いているか」を 1 画面で示す。
package main

import (
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type view struct {
	Env       string
	Commit    string
	BuiltAt   string
	APIStatus string
	APIBody   string
	Worker    string
}

var page = template.Must(template.New("page").Parse(`<!doctype html>
<meta charset="utf-8"><title>kagerou ECS demo</title>
<style>
body{font-family:system-ui,sans-serif;max-width:42rem;margin:0 auto;padding:2rem 1.5rem;line-height:1.7}
h1{font-size:1.5rem} h2{font-size:1.05rem;margin-top:2rem;border-top:1px solid #8884;padding-top:1.2rem}
code{background:#8882;padding:.1rem .35rem;border-radius:4px} .bad{color:#c44} .good{color:#2a7}
th{text-align:left;padding-right:1rem;white-space:nowrap}
</style>
<h1>kagerou ECS demo</h1>
<p>1 つのタスクに 3 コンテナ(gateway / api / worker)。ALB から見えるのは gateway だけ。</p>

<h2>この環境</h2>
<table>
<tr><th>環境名</th><td><code>{{.Env}}</code></td></tr>
<tr><th>commit</th><td><code>{{.Commit}}</code></td></tr>
<tr><th>built at</th><td><code>{{.BuiltAt}}</code></td></tr>
</table>

<h2>api コンテナ(localhost 通信)</h2>
<p class="{{if eq .APIStatus "ok"}}good{{else}}bad{{end}}">{{.APIStatus}}</p>
<pre><code>{{.APIBody}}</code></pre>

<h2>worker コンテナ(共有ボリューム)</h2>
<pre><code>{{.Worker}}</code></pre>
`))

func main() {
	port := env("PORT", "8080")
	apiURL := env("API_URL", "http://127.0.0.1:8081")
	sharedDir := env("SHARED_DIR", "/shared")

	mux := http.NewServeMux()
	// liveness: 自分が生きているかだけ。api / worker は見ない
	// (依存の障害でタスクが作り直されるのを避ける)
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		v := view{
			Env:     env("KAGEROU_ENV", "local"),
			Commit:  env("GIT_COMMIT", "dev"),
			BuiltAt: env("BUILT_AT", "dev"),
		}
		v.APIStatus, v.APIBody = callAPI(apiURL + "/items")
		v.Worker = readWorker(filepath.Join(sharedDir, "worker.txt"))
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_ = page.Execute(w, v)
	})

	log.Printf("gateway listening on :%s (api=%s)", port, apiURL)
	log.Fatal(http.ListenAndServe(":"+port, mux))
}

func callAPI(url string) (status, body string) {
	client := &http.Client{Timeout: 3 * time.Second}
	res, err := client.Get(url)
	if err != nil {
		return "api に届きません: " + err.Error(), ""
	}
	defer res.Body.Close()
	b, _ := io.ReadAll(io.LimitReader(res.Body, 4096))
	var pretty any
	if json.Unmarshal(b, &pretty) == nil {
		if out, err := json.MarshalIndent(pretty, "", "  "); err == nil {
			b = out
		}
	}
	if res.StatusCode != http.StatusOK {
		return fmt.Sprintf("HTTP %d", res.StatusCode), string(b)
	}
	return "ok", string(b)
}

func readWorker(path string) string {
	b, err := os.ReadFile(path)
	if err != nil {
		return "worker の出力がまだありません(起動直後か、共有ボリューム未設定): " + err.Error()
	}
	return strings.TrimSpace(string(b))
}

func env(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}
