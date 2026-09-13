// worker は公開されないバックグラウンド処理。定期的に状態ファイルを書き、
// 同じタスク内の gateway がそれを読んで「worker が生きているか」を見せる。
// (共有ボリュームは ECS のタスク定義で volumes/mountPoints を使う)
package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"
)

func main() {
	dir := os.Getenv("SHARED_DIR")
	if dir == "" {
		dir = "/shared"
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		log.Fatalf("worker: %v", err)
	}
	path := filepath.Join(dir, "worker.txt")
	ticks := 0
	for {
		ticks++
		line := fmt.Sprintf("tick=%d at=%s env=%s\n", ticks,
			time.Now().UTC().Format(time.RFC3339), os.Getenv("KAGEROU_ENV"))
		if err := os.WriteFile(path, []byte(line), 0o644); err != nil {
			log.Printf("worker: write: %v", err)
		}
		time.Sleep(10 * time.Second)
	}
}
