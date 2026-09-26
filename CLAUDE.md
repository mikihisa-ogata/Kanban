# Kanban

カンバンボードでタスクを管理する個人用 Web アプリ。Go の API（`bff/`）と Nuxt 3 のフロントエンド（`frontend/`）で構成される。

開発は AI 駆動で進め、オーナーはコードレビューをしない。issue 対応は `/issue <番号>` の手順に従い、**検証を自分で完結させてから** main に push する。

## 構成

```
bff/        Go 1.23 + gin。データは MySQL 8.4 に保存（接続先は環境変数 DB_DSN。既定は localhost:3306 の kanban）
  cmd/main.go            ルーティングと依存の組み立て（ポート 8080 固定）
  cmd/import-csv/        旧 CSV データを MySQL に取り込むコマンド
  internal/database/     MySQL 接続とテーブル定義（Migrate）。dbtest/ はテスト用の DB 準備
  internal/domain/       エンティティ（Todo, Epic）
  internal/handler/      HTTP の入出力のみ
  internal/service/      ビジネスロジック
  internal/repository/   DB アクセス（interface と *_mysql.go）
frontend/   Nuxt 3（srcDir: src）+ Vue 3 + Tailwind CSS v4
  src/api/               API 通信（API_BASE_URL は http://localhost:8080 固定）
  src/components/        UI 部品
  src/pages/             ページ
  src/layouts/           レイアウト
```

## コマンド

| 目的 | コマンド |
|---|---|
| bff ビルド・静的解析 | `cd bff && go build ./... && go vet ./...` |
| bff テスト | `cd bff && TEST_DB_DSN='root:root@tcp(127.0.0.1:3307)/kanban_test' go test ./...`（検証用 MySQL は下記。TEST_DB_DSN が未設定だと MySQL を使うテストはスキップされ、検証にならない） |
| 検証用 MySQL の起動 | `docker run -d --rm --name kanban-verify-mysql -p 3307:3306 -e MYSQL_ROOT_PASSWORD=root mysql:8.4`（終わったら `docker stop kanban-verify-mysql`） |
| bff 起動 | `cd bff && docker compose up -d mysql && go run ./cmd/main.go`（または `docker compose up -d` で bff ごと起動） |
| フロント起動 | `cd frontend && pnpm dev`（http://localhost:3000。bff の CORS 許可に合わせる） |
| フロントビルド確認 | `cd frontend && pnpm build:verify`（出力先は `.nuxt-verify/`） |

- ビルド確認には必ず `pnpm build:verify` を使う。`pnpm build` は起動中の `nuxt dev` と同じ `.nuxt/` に本番ビルドを上書きし、dev の画面を `#internal/nuxt/paths` エラーで壊す。
- フロントにはテストランナーがない。検証はビルドと画面確認で行う。

## 設計ルール

- 層の依存は Handler → Service → Repository の一方向。Handler にロジックを書かない。DB やファイルにアクセスするのは Repository だけ。
- Go: エラーは握りつぶさずに呼び出し元へ返す。グローバル変数は使わない。
- Vue: Composition API を使う。API 通信は `src/api/` に置き、コンポーネントから直接 `fetch` しない。
- 新しいフレームワークやライブラリは、issue で求められない限り導入しない。既存のディレクトリ構成と書き方に合わせる。
- API のレスポンスやテーブル定義を変えたときは、`bff/README.md` も更新する。テーブル定義は `internal/database` の `Migrate` に置き、既存のデータベースでも動くようにする（列の追加は `ALTER TABLE` を冪等に書くなど）。

## データの扱い（重要）

- オーナーの実データは docker compose の MySQL（localhost:3306 のデータベース `kanban`、ボリューム `mysql-data`）にある。**このデータベースのデータを書き換えない・削除しない。** `docker compose down -v` やボリュームの削除もしない。
- `bff/todos.csv`・`bff/epics.csv`・`bff/spaces.csv` は MySQL 移行前の実データ（バックアップとして残している）。git 管理外。**コミットしない・書き換えない・削除しない。**
- 実データに触れずに検証するときは、上の「検証用 MySQL」（ポート 3307）を使う。テストは `TEST_DB_DSN` のデータベース名に `_repository` などを付けたデータベースを作って中身を消すので、`TEST_DB_DSN` に実データの DB を指定しない。
- オーナーが bff やフロントを起動していることがある（8080 / 3000）。検証中はそのプロセスを止めない。起動中の画面では、表示の確認だけにとどめる。
- 例外として、bff（8080）は bff の変更を push したあとに**自分で再起動して新しいコードを反映する**。オーナーに再起動を頼まない。手順は `/issue` スキルの「起動中の bff を再起動する」に従う。フロント（3000）の `nuxt dev` はホットリロードされるので止めない。タスクの作成・変更・削除は実データを書き換えるので行わない。

## Git

- ブランチも PR も作らず、main に直接コミットして `git push origin main` する。
- コミットメッセージは gitmoji + 日本語の要約（例: `:sparkles:タスクに期限を追加する`、`:bug:…を修正`、`:lipstick:…`）。issue 対応では本文に `Closes #<番号>` を書く。
- force push、`git reset --hard`、履歴の書き換えはしない。
