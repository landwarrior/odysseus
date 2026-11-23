// Command genicon は winres/icon.png から favicon.ico を作ります。
//
// リポジトリの根で次を実行します。
//
//	go run ./tools/genicon
//
// Windows の exe 用リソースは別途、cmd/odysseus で次を実行します。
//
//	go-winres make --in winres/winres.json --arch amd64,arm64
package main

import (
	"fmt"
	"image"
	_ "image/png"
	"os"
	"path/filepath"

	"github.com/tc-hib/winres"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "genicon: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	root, err := os.Getwd()
	if err != nil {
		return err
	}
	src := filepath.Join(root, "cmd", "odysseus", "winres", "icon.png")
	dst := filepath.Join(root, "internal", "web", "static", "favicon.ico")

	f, err := os.Open(src)
	if err != nil {
		return err
	}
	defer f.Close()

	img, _, err := image.Decode(f)
	if err != nil {
		return fmt.Errorf("decode %s: %w", src, err)
	}

	icon, err := winres.NewIconFromResizedImage(img, []int{16, 32, 48})
	if err != nil {
		return err
	}

	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}
	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()

	if err := icon.SaveICO(out); err != nil {
		return err
	}
	fmt.Println("wrote", dst)
	return nil
}
