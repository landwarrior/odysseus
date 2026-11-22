// Package config は起動時の環境変数を読みます。Docker でも同じ名前を使います。
package config

import (
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// Config は起動時に決まる設定です。
// HTTPAddr は待受アドレス、Driver は sqlite か postgres、DSN は接続文字列です。
// DataDir は SQLite ファイルとセッション鍵の置き場です。
// OpenBrowser が真なら起動時にブラウザを開きます。
// StandardMinutes は 1 人日の分数、SessionKey はログインクッキーの署名鍵です。
// Dev が真なら HTML/CSS を埋め込みではなく WebDir のファイルから都度読みます。
type Config struct {
	HTTPAddr        string
	Driver          string
	DSN             string
	DataDir         string
	WebDir          string
	OpenBrowser     bool
	Dev             bool
	StandardMinutes int64
	SessionKey      []byte
}

// Load は baseDir（実行ファイルのあるフォルダ）を基準に環境変数を読みます。
// 空の項目は既定値になります。値が不正なときは error を返します。
func Load(baseDir string) (Config, error) {
	dataDir := env("ODYSSEUS_DATA_DIR", filepath.Join(baseDir, "data"))
	driver := strings.ToLower(env("ODYSSEUS_DB_DRIVER", "sqlite"))
	if driver != "sqlite" && driver != "postgres" {
		return Config{}, fmt.Errorf("ODYSSEUS_DB_DRIVER は sqlite または postgres です: %s", driver)
	}
	dsn := os.Getenv("ODYSSEUS_DB_DSN")
	if dsn == "" {
		if driver == "postgres" {
			return Config{}, fmt.Errorf("PostgreSQL を使うときは ODYSSEUS_DB_DSN が必要です")
		}
		dsn = filepath.Join(dataDir, "odysseus.db")
	}
	stdMin := int64(480)
	if v := strings.TrimSpace(os.Getenv("ODYSSEUS_STANDARD_MINUTES")); v != "" {
		n, err := strconv.ParseInt(v, 10, 64)
		if err != nil || n <= 0 {
			return Config{}, fmt.Errorf("ODYSSEUS_STANDARD_MINUTES は正の整数です")
		}
		stdMin = n
	}
	return Config{
		HTTPAddr:        env("ODYSSEUS_HTTP_ADDR", "127.0.0.1:0"),
		Driver:          driver,
		DSN:             dsn,
		DataDir:         dataDir,
		WebDir:          filepath.Join(baseDir, "internal", "web"),
		OpenBrowser:     env("ODYSSEUS_OPEN_BROWSER", "1") != "0",
		Dev:             env("ODYSSEUS_DEV", "") == "1",
		StandardMinutes: stdMin,
	}, nil
}

// BrowserURL は c.HTTPAddr から、ブラウザで開く URL を作ります。
// 待受が全インターフェースのときは、開く先を 127.0.0.1 にします。
func (c Config) BrowserURL() string {
	host, port, err := net.SplitHostPort(c.HTTPAddr)
	if err != nil {
		return "http://" + c.HTTPAddr
	}
	if host == "" || host == "0.0.0.0" || host == "::" {
		host = "127.0.0.1"
	}
	return "http://" + net.JoinHostPort(host, port)
}

// env は key の環境変数を返します。未設定か空白だけなら fallback を返します。
func env(key, fallback string) string {
	v := strings.TrimSpace(os.Getenv(key))
	if v == "" {
		return fallback
	}
	return v
}
