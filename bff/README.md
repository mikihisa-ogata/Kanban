# BFF（Backend For Frontend）

Go言語で実装されたTodo APIのバックエンドです。

## 概要

このバックエンドは、フロントエンドからのリクエストを処理し、TODOタスクの管理機能を提供します。データは MySQL に保存します。

## プロジェクト構成

```
bff/
├── cmd/
│   ├── main.go              # アプリケーション エントリーポイント
│   └── import-csv/          # CSV 保存時代のデータを MySQL に取り込むコマンド
├── internal/
│   ├── database/
│   │   ├── database.go      # MySQL への接続とテーブル作成
│   │   └── dbtest/          # MySQL を使うテストの準備
│   ├── domain/
│   │   └── todo.go          # TODOドメインモデル
│   ├── handler/
│   │   └── todo.go          # HTTPハンドラー
│   ├── repository/
│   │   ├── todo_repository.go        # リポジトリインターフェース
│   │   ├── todo_repository_mysql.go  # リポジトリ実装（MySQL）
│   │   └── csv_import.go             # CSV からの取り込み
│   └── service/
│       └── toodo.go         # ビジネスロジック
├── go.mod
├── Dockerfile               # Dockerビルド設定
├── docker-compose.yml       # Docker Compose設定
├── .dockerignore            # CSV（旧データ）をイメージに含めない設定
└── README.md
```

## 起動方法

### 前提条件
- [Docker](https://www.docker.com/)
- [Docker Compose](https://docs.docker.com/compose/)

### Dockerを使用した起動

Dockerを使用することで、環境構築なしでサーバーを起動できます。`bff` と `mysql`（MySQL 8.4、ポート 3306）が起動し、データは名前付きボリューム `mysql-data` に保存されます。

```bash
docker compose up -d
```

サーバーは `http://localhost:8080` で起動します。

ログを確認する場合：
```bash
docker compose logs -f
```

停止する場合：
```bash
docker compose down
```

### ローカルでの起動（Dockerを使用しない場合）

#### 前提条件
- Go 1.23 以上
- MySQL（Docker で MySQL だけを起動する場合は `docker compose up -d mysql`）

#### サーバーの起動

```bash
go run ./cmd/main.go
```

接続先は環境変数 `DB_DSN` で指定します（未設定の場合は `kanban:kanban@tcp(localhost:3306)/kanban`）。起動時にテーブルがなければ作成します。

## API エンドポイント

### 全TODOを取得

```
GET /todos
```

**レスポンス例:**
```json
[
  {
    "ID": 1,
    "Title": "タスク1",
    "Done": false,
    "Deadline": "2026-03-31（省略可。省略すると期限なし）",
    "Status": "Waiting",
    "EpicID": 0,
    "Description": "タスクの説明（任意。改行を含められる）",
    "SpaceID": 0
  }
]
```

### 新規TODOを作成

```
POST /todos
```

**リクエスト:**
```json
{
  "Title": "新しいタスク",
  "Deadline": "2026-03-31（省略可。省略すると期限なし）",
  "Status": "Open",
  "Description": "タスクの説明（省略可）",
  "EpicID": 0,
  "SpaceID": 0
}
```

### TODOを更新

```
PUT /todos/:id
```

**リクエスト:**
```json
{
  "Title": "更新されたタスク",
  "Done": true,
  "Deadline": "2026-03-31（省略すると期限なしになる）",
  "Status": "InProgress",
  "Description": "タスクの説明（省略すると空になる）",
  "EpicID": 0,
  "SpaceID": 0
}
```

### TODOを削除

```
DELETE /todos/:id
```

### TODOを並び替え

```
POST /todos/:id/move
```

**リクエスト:**
```json
{
  "BeforeID": 3
}
```

タスクを `BeforeID` のタスクの直前へ移動します。`BeforeID` が 0 または省略の場合は末尾へ移動します。`GET /todos` はこの並び順（`todos.position`）で返します。

`EpicID` / `SpaceID` は省略すると 0（未割り当て）になります。エピックを指定する場合、タスクの `SpaceID` はそのエピックの `SpaceID` と同じでなければなりません（異なると 400）。

### エピック

```
GET /epics
POST /epics        {"Title": "エピック名", "SpaceID": 0}
PUT /epics/:id     {"Title": "エピック名", "SpaceID": 2}
DELETE /epics/:id
```

レスポンスは `{"ID": 1, "Title": "エピック名", "SpaceID": 0}` の配列です。`PUT` でスペースを変えると、子タスクも同じスペースへ移動します。削除すると子タスクはエピック未割り当てになります。

### スペース

エピックの上位の単位です（プロダクトごとのTODOリストなど）。

```
GET /spaces
POST /spaces       {"Title": "スペース名"}
DELETE /spaces/:id
```

レスポンスは `{"ID": 1, "Title": "スペース名"}` の配列です。削除すると、属していたエピックとタスクはスペース未割り当て（`SpaceID` 0）になります。エピックとタスクの紐付けは残ります。

## アーキテクチャ

本プロジェクトはクリーンアーキテクチャを採用しており、以下のレイヤーで構成されています：

- **Domain Layer**: ビジネスロジックと関連エンティティ
- **Service Layer**: ビジネスロジックの実装
- **Repository Layer**: データアクセス層
- **Handler Layer**: HTTPリクエスト/レスポンスの処理

## データ永続化

データは MySQL の `spaces`・`epics`・`todos` テーブルに保存します。テーブルは起動時に `internal/database` の `Migrate` が作成します（`CREATE TABLE IF NOT EXISTS`）。

- `todos.position` がタスクの表示順です。
- `EpicID`・`SpaceID` の 0 は「未割り当て」を表すため、外部キー制約は付けていません。
- `Deadline` は `DATE` 型で、期限なし（空文字）は `NULL` として保存します。

### CSV からの移行

以前はデータを `todos.csv`・`epics.csv`・`spaces.csv` に保存していました。これらのファイルを MySQL に取り込むには、`bff/` で次を実行します（`DB_DSN` の MySQL に取り込みます）。ID とタスクの並び順はそのまま引き継ぎます。CSV は読むだけで変更しません。取り込み先のテーブルが空でない場合は何もしません。

```bash
go run ./cmd/import-csv            # カレントディレクトリの CSV を取り込む
go run ./cmd/import-csv -dir 別の場所  # 別のディレクトリの CSV を取り込む
```

## テスト

MySQL を使うテストは、環境変数 `TEST_DB_DSN` を指定したときだけ実行されます（未設定の場合はスキップされます）。テストは `TEST_DB_DSN` のデータベース名に `_repository`・`_handler` などを付けたデータベースを作り、中身を消しながら使います。そのため、DSN のユーザーにはデータベースを作成する権限が必要です。実データのデータベースは指定しないでください。

```bash
TEST_DB_DSN='root:root@tcp(localhost:3306)/kanban_test' go test ./...
```

## ライセンス

MIT
