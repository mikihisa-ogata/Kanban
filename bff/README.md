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
├── Dockerfile               # Dockerビルド設定
├── docker-compose.yml       # Docker Compose設定
└── README.md
```

## 起動方法

### 前提条件
- [Docker](https://www.docker.com/)
- [Docker Compose](https://docs.docker.com/compose/)

### Dockerを使用した起動

Dockerを使用することで、環境構築なしでサーバーを起動できます。データは `todos.csv` と `epics.csv` に永続化されます。

初回のみ、雛形からデータファイルを作成してください（ファイルがないとDockerがディレクトリを作成してしまうため）。

```bash
cp -n todos.example.csv todos.csv
cp -n epics.example.csv epics.csv
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
    "Deadline": "2026-03-31",
    "Status": "Waiting",
    "EpicID": 0,
    "Description": "タスクの説明（任意。改行を含められる）"
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
  "Deadline": "2026-03-31",
  "Status": "Open",
  "Description": "タスクの説明（省略可）"
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
  "Deadline": "2026-03-31",
  "Status": "InProgress",
  "Description": "タスクの説明（省略すると空になる）"
}
```

### TODOを削除

```
DELETE /todos/:id
```

## アーキテクチャ

本プロジェクトはクリーンアーキテクチャを採用しており、以下のレイヤーで構成されています：

- **Domain Layer**: ビジネスロジックと関連エンティティ
- **Service Layer**: ビジネスロジックの実装
- **Repository Layer**: データアクセス層
- **Handler Layer**: HTTPリクエスト/レスポンスの処理

## データ永続化

TODOデータは `bff/todos.csv`、エピックデータは `bff/epics.csv` に保存されます。どちらも個人のチケットデータのため git 管理外で、リポジトリには雛形の `*.example.csv` のみを置いています。`go run` で起動する場合はファイルがなくても自動で作成されます。Docker環境ではボリュームマウントされており、ホスト側のファイルを直接更新・参照可能です。

`todos.csv` の列は `ID,Title,Done,Deadline,Status,EpicID,Description` です。`Description` 列がない旧形式のファイルもそのまま読み込め、次に書き込んだときに列が追加されます。

## ライセンス

MIT
