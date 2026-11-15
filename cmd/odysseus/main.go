// Command odysseus は人材予実管理を、この PC のブラウザから使うサーバとして起動します。
package main

import (
	"bytes"
	"context"
	"crypto/rand"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
	"time"

	"odysseus/internal/config"
	"odysseus/internal/db"
	"odysseus/internal/store"
	"odysseus/internal/web"
)

// main は設定を読み、データベースを用意し、空いているポートで待受を始めます。
func main() {
	cfg, err := config.Load(appDir())
	if err != nil {
		log.Fatal(err)
	}
	if err := os.MkdirAll(cfg.DataDir, 0o700); err != nil {
		log.Fatal(err)
	}
	database, err := db.Open(cfg.Driver, cfg.DSN)
	if err != nil {
		log.Fatal(err)
	}
	defer database.SQL.Close()
	ctx := context.Background()
	if err := db.Migrate(ctx, database); err != nil {
		log.Fatal(err)
	}
	st := store.New(database)
	password, created, err := st.Seed(ctx)
	if err != nil {
		log.Fatal(err)
	}
	if created {
		path := filepath.Join(cfg.DataDir, "initial-admin.txt")
		if err := writeInitialPassword(path, password); err != nil {
			log.Fatal(err)
		}
		log.Printf("初期管理者を作成しました。ユーザーコード: admin")
		log.Printf("初期パスワード: %s", password)
		log.Printf("同じ内容を %s に書きました。変更するとこのファイルは削除されます。", path)
	}
	key, err := loadOrCreateKey(filepath.Join(cfg.DataDir, "session.key"))
	if err != nil {
		log.Fatal(err)
	}
	cfg.SessionKey = key
	srv, err := web.NewServer(st, cfg)
	if err != nil {
		log.Fatal(err)
	}
	ln, err := net.Listen("tcp", cfg.HTTPAddr)
	if err != nil {
		log.Fatalf("待受を開始できません (%s): %v", cfg.HTTPAddr, err)
	}
	cfg.HTTPAddr = ln.Addr().String()
	httpServer := &http.Server{
		Handler:           srv.Handler(),
		ReadHeaderTimeout: 10 * time.Second,
	}
	log.Printf("Odysseus を起動しました")
	log.Printf("  開く URL: %s", cfg.BrowserURL())
	log.Printf("  データ: %s", cfg.DataDir)
	log.Printf("  データベース: %s", cfg.Driver)
	log.Printf("停止するには、この画面で Ctrl+C を押してください。")
	go func() {
		if err := httpServer.Serve(ln); err != nil && err != http.ErrServerClosed {
			log.Fatal(err)
		}
	}()
	if cfg.OpenBrowser {
		openBrowser(cfg.BrowserURL())
	}
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	<-stop
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		log.Printf("停止: %v", err)
	}
}

// appDir は設定とデータの基準フォルダです。
// go run の一時フォルダで動いているときは、今の作業フォルダを返します。
func appDir() string {
	exe, err := os.Executable()
	if err != nil {
		wd, _ := os.Getwd()
		return wd
	}
	dir := filepath.Dir(exe)
	if strings.Contains(filepath.ToSlash(dir), "go-build") {
		if wd, err := os.Getwd(); err == nil {
			return wd
		}
	}
	return dir
}

// loadOrCreateKey は path にある署名鍵を読みます。無ければ 32 バイトを作って保存します。
func loadOrCreateKey(path string) ([]byte, error) {
	b, err := os.ReadFile(path)
	if err == nil && len(b) >= 32 {
		return b[:32], nil
	}
	if err != nil && !os.IsNotExist(err) {
		return nil, err
	}
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return nil, err
	}
	if err := os.WriteFile(path, buf, 0o600); err != nil {
		return nil, err
	}
	return buf, nil
}

// writeInitialPassword は path に、初期管理者の password をメモ帳で読める UTF-8 で書きます。
func writeInitialPassword(path, password string) error {
	var buf bytes.Buffer
	buf.Write([]byte{0xEF, 0xBB, 0xBF})
	fmt.Fprintf(&buf, "初期管理者を作成しました。\r\nユーザーコード: admin\r\n初期パスワード: %s\r\n\r\nログイン後、パスワードを変更してください。\r\n変更するとこのファイルは削除されます。\r\n", password)
	return os.WriteFile(path, buf.Bytes(), 0o600)
}

// openBrowser は rawURL を、OS の既定ブラウザで開きます。開けなくても起動は続けます。
func openBrowser(rawURL string) {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", rawURL)
	case "darwin":
		cmd = exec.Command("open", rawURL)
	default:
		cmd = exec.Command("xdg-open", rawURL)
	}
	if err := cmd.Start(); err != nil {
		log.Printf("ブラウザを開けませんでした: %v", err)
	}
}
