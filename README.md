# Todo API

シンプルで実用的なTODO管理アプリケーションです。バックエンド（Go）とフロントエンド（Vue.js）で構成されています。

## プロジェクト構成

```
todo-api/
├── bff/              # バックエンド（Go）
├── frontend/         # フロントエンド（Vue.js）
└── README.md
```

## 機能

- TODOタスクの作成、編集、削除
- カンバンボード形式でのタスク管理
- ドラッグ&ドロップによるステータス変更
- CSV形式でのデータ永続化

## クイックスタート

### 前提条件

- Go 1.16+
- Node.js 14+
- npm

### セットアップ

#### バックエンド（bff）の起動

```bash
cd bff
go run ./cmd/main.go
```

サーバーは `http://localhost:8080` で起動します。

#### フロントエンド（frontend）の起動

```bash
cd frontend
npm install
npm run dev
```

フロントエンドは `http://localhost:5173` で利用可能になります。

## API

バックエンドは以下のエンドポイントを提供します：

- `GET /todos` - 全TODOを取得
- `POST /todos` - 新規TODOを作成
- `PUT /todos/:id` - TODOを更新
- `DELETE /todos/:id` - TODOを削除

詳細は [bff/README.md](bff/README.md) を参照してください。

## フロントエンド

フロントエンドの詳細については [frontend/README.md](frontend/README.md) を参照してください。

## 技術スタック

### バックエンド
- Go
- CSV形式でのデータ永続化

### フロントエンド
- Vue 3
- Vite
- CSS

## ライセンス

MIT
