---
name: issue
description: GitHub issue を番号指定で受け取り、調査・実装・検証・セルフレビュー・main への push・issue への報告までを人のレビューなしで完結させる。「issue #7 対応して」「#12 やって」「/issue 5」のように issue 番号付きで対応を頼まれたら使う。
argument-hint: <issue番号>
---

# issue 対応

対象: issue #$ARGUMENTS

オーナーはコードをレビューしない。**あなたの検証が最後の品質ゲートになる。** 検証が通るまでは完了扱いにしない。途中で「進めてよいですか」と確認を求めない。

## 1. 要件を読む

- `gh issue view $ARGUMENTS --comments` で本文とコメントを読む。
- issue が閉じている、または番号が存在しない場合は、何もせずにその旨を報告して終える。
- 関連するコードを読み、変更範囲を特定する。`CLAUDE.md` の設計ルールとデータの扱いに従う。

## 2. 判断する

- 要件の解釈が複数あり、どれを選ぶかでユーザー体験やデータが大きく変わる場合（例: データ形式の破壊的変更、機能の削除範囲が読み取れない）に限り、AskUserQuestion で質問する。
- それ以外は、既存の挙動や UI に一番自然に合う案を自分で選んで進める。選んだ内容と理由は最後の報告に書く。
- issue が大きく、1 回で終えるには無理がある場合（例: #4 DB 移行、#6 ユーザー管理）は、着手前に分割案を報告して指示を待つ。

## 3. 実装する

- 既存の書き方に合わせ、issue の範囲外の変更（ついでのリファクタリングなど）はしない。
- bff のロジックやハンドラーを変更したら、`go test` で確かめられるテストを `_test.go` に追加する。DB を使うテストでは `internal/database/dbtest` の `dbtest.Open(t, "<パッケージ名>")` で空のテスト用データベースを用意し、実データに触れないようにする。
- フロントにはテストランナーがないので、新しく導入しない。

## 4. 自分で検証する

次の中から、変更した側に当てはまるものをすべて実行し、すべて通るまで直す。

- bff: 検証用 MySQL（`CLAUDE.md` のコマンド表）を起動し、`cd bff && go build ./... && go vet ./... && TEST_DB_DSN='root:root@tcp(127.0.0.1:3307)/kanban_test' go test ./...`。MySQL を使うテストが SKIP ではなく PASS になっていることを確かめる。
- frontend: `cd frontend && pnpm build:verify`（`pnpm build` は起動中の dev サーバーを壊すので使わない）
- 画面や API の挙動が変わる場合は、実際に動かして確認する。
  - API: `go build -o <scratch>/bff ./cmd` でビルドしたバイナリを `DB_DSN='root:root@tcp(127.0.0.1:3307)/kanban_verify'` を付けて起動し、`curl` で確認する（データベース `kanban_verify` は事前に作成しておく）。ポート 8080 が使用中なら、検証のためにオーナーのプロセスは止めない。代わりに別ポートで起動するか、httptest のテストで確認する。
  - 画面: claude-in-chrome スキルで http://localhost:3000 を開き、変更箇所を確認する。オーナーが起動中のサーバーでは表示確認のみ行い、データを変更する操作はしない。サーバーが起動していない場合は、上のスクラッチ用の bff と `pnpm dev` を自分で起動して操作を確認し、終わったら自分で起動したプロセスを止める。
  - Chrome 連携が使えないなど、確認できなかったことは報告に「未確認」として正直に書く。

## 5. セルフレビュー

- `git diff` 全体を読み直し、バグ、issue の要件漏れ、`CLAUDE.md` のルール違反、不要な変更やデバッグ出力が残っていないかを確認する。
- 見つけた問題は直し、手順 4 の検証を再実行する。

## 6. コミットして push する

- `git status` を確認し、`bff/todos.csv`・`bff/epics.csv`・`bff/spaces.csv`・ビルド成果物・スクラッチのファイルを含めない。変更したファイルだけを名前指定で `git add` する。
- コミットメッセージは gitmoji + 日本語の要約。本文に `Closes #$ARGUMENTS` を書く。
- `git push origin main` する。push が拒否されたら `git pull --rebase origin main` してから検証を再実行し、もう一度 push する。force push はしない。

## 7. 起動中の bff を再起動する

bff を変更して push した場合は、オーナーが起動中の bff に新しいコードを自分で反映する。オーナーに再起動を頼まない（フロントの `nuxt dev` はホットリロードされるので触らない）。

- `lsof -iTCP:8080 -sTCP:LISTEN` で起動中か確認する。起動していなければ何もしない。
- `ps -o pid,ppid,command` と `lsof -a -p <pid> -d cwd` で起動方法を確かめる。
  - `go run ./cmd/main.go`（カレントが `bff/`）の場合: 先に `lsof -iTCP:3306 -sTCP:LISTEN` で MySQL が動いていることを確かめ（動いていなければ `cd bff && docker compose up -d mysql`）、子プロセス（`go-build/.../main`）と親の `go run` を `kill` し、8080 が空いたら `cd bff && nohup go run ./cmd/main.go >> ~/Library/Logs/kanban-bff.log 2>&1 < /dev/null & disown` で起動し直す。セッション終了後も動き続けるよう、バックグラウンドタスク（run_in_background）では起動しない。
  - Docker（`docker compose`）の場合: `cd bff && docker compose up -d --build` で作り直す（`down -v` はしない。実データのボリュームが消える）。
  - それ以外の起動方法の場合は止めずに、報告に「再起動が必要」と書く。
- 起動後に `curl` で GET の API が 200 を返すことを確かめる（POST・PUT・DELETE は実データを書き換えるので実行しない）。
- 再起動できたかどうかを報告に書く。

## 8. 報告する

`gh issue comment $ARGUMENTS` で issue に次をコメントし、同じ内容をユーザーにも返す。

- 何を変えたか（ユーザーから見た変化）
- 自分で判断したことと、その理由
- 実行した検証とその結果
- 起動中の bff を再起動したか
- 未確認の点や、残っている懸念（なければ「なし」）
- コミットのハッシュ

コミットの `Closes` で issue が閉じなかった場合は、`gh issue close $ARGUMENTS` で閉じる。
