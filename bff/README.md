# BFF（Backend For Frontend）

Go言語で実装されたTodo APIのバックエンドです。

## 概要

このバックエンドは、フロントエンドからのリクエストを処理し、TODOタスクの管理機能を提供します。CSVファイルを使用してデータを永続化します。

## プロジェクト構成

```
bff/
├── cmd/
│   └── main.go              # アプリケーション エントリーポイント
├── internal/
│   ├── database/
│   │   └── database.go      # MySQL への接続とテーブル作成
│   ├── domain/
│   │   └── todo.go          # TODOドメインモデル
│   ├── handler/
│   │   └── todo.go          # HTTPハンドラー
│   ├── repository/
│   │   ├── todo_repository.go       # リポジトリインターフェース
│   │   └── todo_repository_impl.go  # リポジトリ実装
│   └── service/
│       └── toodo.go         # ビジネスロジック
├── go.mod
├── todos.example.csv        # TODOデータの雛形（todos.csv はgit管理外）
├── epics.example.csv        # エピックデータの雛形（epics.csv はgit管理外）
├── spaces.example.csv       # スペースデータの雛形（spaces.csv はgit管理外）
├── Dockerfile               # Dockerビルド設定
├── docker-compose.yml       # Docker Compose設定
├── .dockerignore            # CSV をイメージに含めない設定
└── README.md
```

## 起動方法

### 前提条件
- [Docker](https://www.docker.com/)
- [Docker Compose](https://docs.docker.com/compose/)

### Dockerを使用した起動

Dockerを使用することで、環境構築なしでサーバーを起動できます。データは `todos.csv`・`epics.csv`・`spaces.csv` に永続化されます。

初回のみ、雛形からデータファイルを作成してください（ファイルがないとDockerがディレクトリを作成してしまうため）。

```bash
cp -n todos.example.csv todos.csv
cp -n epics.example.csv epics.csv
cp -n spaces.example.csv spaces.csv
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

#### サーバーの起動

```bash
go run ./cmd/main.go
```

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

タスクを `BeforeID` のタスクの直前へ移動します。`BeforeID` が 0 または省略の場合は末尾へ移動します。`GET /todos` はこの並び順（`todos.csv` の行の順番）で返します。

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

TODOデータは `bff/todos.csv`、エピックデータは `bff/epics.csv`、スペースデータは `bff/spaces.csv` に保存されます。いずれも個人のチケットデータのため git 管理外で、リポジトリには雛形の `*.example.csv` のみを置いています。`go run` で起動する場合はファイルがなくても自動で作成されます。Docker環境ではボリュームマウントされており、ホスト側のファイルを直接更新・参照可能です。

`todos.csv` の列は `ID,Title,Done,Deadline,Status,EpicID,Description,SpaceID` で、行の順番がタスクの表示順になります。`epics.csv` の列は `ID,Title,SpaceID`、`spaces.csv` の列は `ID,Title` です。`Description` や `SpaceID` の列がない旧形式のファイルもそのまま読み込め（`SpaceID` は 0 = 未割り当て）、次に書き込んだときに列が追加されます。

Docker で起動している場合、`spaces.csv` がないと Docker がディレクトリを作成してしまうため、更新後は `cp -n spaces.example.csv spaces.csv` を実行してから起動し直してください。

## MySQL（CSV からの移行中）

データの保存先を CSV から MySQL へ移行している途中です（issue #4）。Repository の MySQL 実装（`internal/repository/*_mysql.go`）はできていますが、CSV からのデータ移行が済むまでは API は CSV を使っており、MySQL には接続しません。

- 接続先は環境変数 `DB_DSN` で指定します（例: `kanban:kanban@tcp(localhost:3306)/kanban`）。未設定の場合はこの例の値を使います。
- `docker compose up -d` を実行すると `mysql`（MySQL 8.4、ポート 3306）も起動します。データは名前付きボリューム `mysql-data` に保存されます。
- テーブル（`spaces`・`epics`・`todos`）は `internal/database` の `Migrate` が作成します（`CREATE TABLE IF NOT EXISTS`）。`todos.position` はタスクの表示順です。

### テスト

MySQL を使うテストは、環境変数 `TEST_DB_DSN` にテスト専用のデータベースを指定したときだけ実行されます（未設定の場合はスキップされます）。テストはテーブルの中身を消すことがあるので、実データのデータベースは指定しないでください。

```bash
TEST_DB_DSN='kanban:kanban@tcp(localhost:3306)/kanban_test' go test ./...
```

## ライセンス

MIT
