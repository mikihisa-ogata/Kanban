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
├── todos.csv                # TODOデータ
└── README.md
```

## 起動方法

### 前提条件
- Go 1.16 以上

### サーバーの起動

```bash
go run ./cmd/main.go
```

サーバーは `http://localhost:8080` で起動します。

## API エンドポイント

### 全TODOを取得

```
GET /todos
```

**レスポンス例:**
```json
[
  {
    "id": "1",
    "title": "タスク1",
    "description": "説明",
    "status": "todo",
    "createdAt": "2026-02-11T00:00:00Z"
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
  "title": "新しいタスク",
  "description": "タスクの説明",
  "status": "todo"
}
```

### TODOを更新

```
PUT /todos/:id
```

**リクエスト:**
```json
{
  "title": "更新されたタスク",
  "description": "更新された説明",
  "status": "in_progress"
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

TODOデータはプロジェクトルート直下の `todos.csv` ファイルに保存されます。

## 開発

### 依存関係

```bash
go mod download
```

### テスト（実装予定）

```bash
go test ./...
```

## ライセンス

MIT
