# Kanban

カンバンボードでタスクを管理する個人用 Web アプリ。Go の API（`bff/`）と Nuxt 3 のフロントエンド（`frontend/`）で構成される。

開発は AI 駆動で進め、オーナーはコードレビューをしない。issue 対応は `/issue <番号>` の手順に従い、**検証を自分で完結させてから** main に push する。

## 構成

```
bff/        Go 1.23 + gin。データは CSV（todos.csv / epics.csv）に保存
  cmd/main.go            ルーティングと依存の組み立て（ポート 8080 固定）
  internal/domain/       エンティティ（Todo, Epic）
  internal/handler/      HTTP の入出力のみ
  internal/service/      ビジネスロジック
  internal/repository/   CSV アクセス（interface と *_impl.go）
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
| bff テスト | `cd bff && go test ./...` |
| bff 起動 | `cd bff && go run ./cmd/main.go`（または `docker compose up -d`） |
| フロント起動 | `cd frontend && npm run dev -- --port 5173`（http://localhost:5173。bff の CORS 許可に合わせる） |
| フロントビルド確認 | `cd frontend && npm run build:verify`（出力先は `.nuxt-verify/`） |

- ビルド確認には必ず `npm run build:verify` を使う。`npm run build` は起動中の `nuxt dev` と同じ `.nuxt/` に本番ビルドを上書きし、dev の画面を `#internal/nuxt/paths` エラーで壊す。
- フロントにはテストランナーがない。検証はビルドと画面確認で行う。

## 設計ルール

- 層の依存は Handler → Service → Repository の一方向。Handler にロジックを書かない。ファイルにアクセスするのは Repository だけ。
- Go: エラーは握りつぶさずに呼び出し元へ返す。グローバル変数は使わない。
- Vue: Composition API を使う。API 通信は `src/api/` に置き、コンポーネントから直接 `fetch` しない。
- 新しいフレームワークやライブラリは、issue で求められない限り導入しない。既存のディレクトリ構成と書き方に合わせる。
- API のレスポンスや CSV の形式を変えたときは、`bff/*.example.csv` のヘッダーと `README.md` / `bff/README.md` も更新する。

## データの扱い（重要）

- `bff/todos.csv` と `bff/epics.csv` はオーナーの実データで、git 管理外。**コミットしない・書き換えない・削除しない。**
- CSV のパスは実行時のカレントディレクトリからの相対パス。実データに触れずに検証するときは、スクラッチ用のディレクトリに `*.example.csv` をコピーし、そのディレクトリをカレントにしてビルド済みのバイナリを起動する。
- オーナーが bff やフロントを起動していることがある（8080 / 5173。3000 番は別プロジェクトが使っていることがある）。そのプロセスを止めない。起動中の画面では、表示の確認だけにとどめる。タスクの作成・変更・削除は実データを書き換えるので行わない。

## Git

- ブランチも PR も作らず、main に直接コミットして `git push origin main` する。
- コミットメッセージは gitmoji + 日本語の要約（例: `:sparkles:タスクに期限を追加する`、`:bug:…を修正`、`:lipstick:…`）。issue 対応では本文に `Closes #<番号>` を書く。
- force push、`git reset --hard`、履歴の書き換えはしない。
