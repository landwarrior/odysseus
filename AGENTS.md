# Odysseus

人材の予定工数・予定原価と、実績工数・実績原価を突き合わせる。ブランチは `features/new-generation`。修正のたびにこのブランチへコミットする。push は頼まれたときだけ。

## 構成

Go の単一実行ファイル。画面はサーバが返す HTML。PHP と Chef は捨て済み。詳細な合意は `docs/再設計.md`。

- `cmd/odysseus` … 起動。待受の既定は `127.0.0.1:0`（空いているポート）。URL は起動ログに出す。
- `internal/db` … 接続、`?` を PostgreSQL の `$1` に直す処理、マイグレーション。業務 SQL は書かない。
- `internal/store` … テーブルの読み書き。`ctx` は中断用で、各関数では繰り返さない。
- `internal/calc` と `internal/report` … 金額と工数。分と円。データベースを知らない。
- `internal/web` … 画面。ハンドラの `w` は応答、`r` はリクエスト。

DB は `ODYSSEUS_DB_DRIVER=sqlite|postgres`。SQLite が既定。MySQL は未対応。主キーが `TEXT` なので、入れるなら `VARCHAR` に替える必要がある。

Go のドキュメントは `//` を宣言の直前に置く。仮引数は本文で名前を出して書く。`@param` はない。

## 工程画面と役割画面

`main.process-page` と `main.role-page`。ここは崩れやすい。次を守る。行の中身は同じクラス（`process-add`、`process-list` など）を使う。

- 幅は `min(100%, 1100px)`。1100px に達したあとは、ブラウザを広げても幅も中身も変えない。
- 左右余白は `24px` 固定。`vw` や `rem` にしない。根の文字サイズは `clamp(16px, 0.35vw + 14px, 20px)` なので、`rem` はウィンドウが広いほど大きくなり、1100px の内側を削る。
- これらの画面の文字とラベルは `16px` 固定。説明文（`.note`）だけ `0.92rem` のままなので、ウィンドウが広いと大きくなる。行の幅は変えない。
- 登録済みの1行は、次の3塊で折り返す。塊の中では折り返さない。
  1. 番号と「上へ」「下へ」
  2. 名前の入力と「有効」。見える「名前」ラベルは付けない（`aria-label` のみ）。幅は空いている分を使う。下限 225px。
  3. 「保存」と「削除」。幅 126px、間隔 10px。削除は赤。
- 追加欄には「名前」ラベルを残す。新しい工程や役割は一覧の一番下に入る。

## 起動

```powershell
$env:CGO_ENABLED = "0"
go build -trimpath -ldflags "-s -w" -o odysseus.exe ./cmd/odysseus
.\odysseus.exe
```

CSS は実行ファイルに埋め込まれる。画面を変えたらビルドし直す。初期管理者は `admin`。パスワードは起動ログと `data/initial-admin.txt`。
