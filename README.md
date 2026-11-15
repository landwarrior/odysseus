# Odysseus

人材の予定と実績を突き合わせるシステムです。実行ファイルを起動すると、その PC のブラウザから使えます。データベースの初期値は SQLite です。

## 起動

開発マシンで実行ファイルを作ります。

```powershell
$env:CGO_ENABLED = "0"
go build -trimpath -ldflags "-s -w" -o odysseus.exe ./cmd/odysseus
.\odysseus.exe
```

起動した画面は開いたままにします。止めるときは Ctrl+C です。ポートは起動のたびに空いている番号を使い、ブラウザはその URL を開きます。実際のアドレスは起動画面に出ます。データは実行ファイルと同じ場所の `data` フォルダにできます。

初めて起動すると、管理者 `admin` が作られます。初期パスワードは起動画面と `data/initial-admin.txt` に出ます。ログインしたら、そのパスワードを変更します。変更するとテキストファイルは削除されます。

## 設定

Docker で動かすときも、同じ環境変数を使います。

| 変数 | 既定 | 意味 |
| --- | --- | --- |
| `ODYSSEUS_HTTP_ADDR` | `127.0.0.1:0`（空いているポート） | 待受。固定したいときだけ指定する。コンテナでは `0.0.0.0:8080` など |
| `ODYSSEUS_DB_DRIVER` | `sqlite` | `sqlite` または `postgres` |
| `ODYSSEUS_DB_DSN` | `data/odysseus.db` | 接続文字列。PostgreSQL では必須 |
| `ODYSSEUS_DATA_DIR` | 実行ファイル横の `data` | SQLite とセッション鍵の置き場 |
| `ODYSSEUS_OPEN_BROWSER` | `1` | `0` でブラウザを開かない |
| `ODYSSEUS_STANDARD_MINUTES` | `480` | 1人日の分数 |

PostgreSQL は接続先を変えるだけで使えます。SQL は共通で、プレースホルダの違いだけを接続層で吸収しています。

## テスト

```powershell
go test ./...
```

PostgreSQL でも同じマイグレーションを確認する場合は、空のデータベースを指す `ODYSSEUS_TEST_POSTGRES_DSN` を置いてからテストします。無ければその確認は飛ばします。
