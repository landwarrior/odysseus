// Command genicon は winres/icon.png から favicon.ico とログイン用 logo.png を作ります。
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
	"image/png"
	"os"
	"path/filepath"

	"github.com/tc-hib/winres"
	"golang.org/x/image/draw"
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
	staticDir := filepath.Join(root, "internal", "web", "static")
	favicon := filepath.Join(staticDir, "favicon.ico")
	logo := filepath.Join(staticDir, "logo.png")

	f, err := os.Open(src)
	if err != nil {
		return err
	}
	defer f.Close()

	img, _, err := image.Decode(f)
	if err != nil {
		return fmt.Errorf("decode %s: %w", src, err)
	}

	if err := os.MkdirAll(staticDir, 0o755); err != nil {
		return err
	}

	icon, err := winres.NewIconFromResizedImage(img, []int{16, 32, 48})
	if err != nil {
		return err
	}
	out, err := os.Create(favicon)
	if err != nil {
		return err
	}
	if err := icon.SaveICO(out); err != nil {
		out.Close()
		return err
	}
	if err := out.Close(); err != nil {
		return err
	}
	fmt.Println("wrote", favicon)

	const logoSize = 192
	logoImg := image.NewRGBA(image.Rect(0, 0, logoSize, logoSize))
	draw.CatmullRom.Scale(logoImg, logoImg.Bounds(), img, img.Bounds(), draw.Over, nil)
	logoOut, err := os.Create(logo)
	if err != nil {
		return err
	}
	if err := png.Encode(logoOut, logoImg); err != nil {
		logoOut.Close()
		return err
	}
	if err := logoOut.Close(); err != nil {
		return err
	}
	fmt.Println("wrote", logo)
	return nil
}
