# Frontend

Vue 3 + Viteで実装されたTodo APIのフロントエンドアプリケーションです。

## 概要

カンバンボード形式でTODOタスクを管理できるWebアプリケーションです。ドラッグ&ドロップによるタスク移動や、タスクの作成・編集・削除が可能です。

## プロジェクト構成

```
frontend/
├── src/
│   ├── components/
│   │   ├── KanbanBoard.vue    # カンバンボード親コンポーネント
│   │   ├── KanbanColumn.vue   # カンバンボード列コンポーネント
│   │   ├── TaskCard.vue       # タスクカード表示コンポーネント
│   │   └── TaskForm.vue       # タスク作成/編集フォーム
│   ├── api/
│   │   └── todos.js           # API通信ユーティリティ
│   ├── assets/                # 静的アセット
│   ├── App.vue                # ルートコンポーネント
│   ├── main.js                # アプリケーションエントリーポイント
│   └── style.css              # グローバルスタイル
├── public/                    # 静的ファイル
├── index.html                 # HTMLテンプレート
├── vite.config.js            # Vite設定ファイル
├── package.json              # npm依存関係
└── README.md
```

## セットアップ

### 前提条件

- Node.js 14+ 
- npm

### インストール

```bash
npm install
```

## 開発サーバーの起動

```bash
npm run dev
```

開発サーバーは `http://localhost:5173` で起動します。ファイルの変更は自動的にホットリロードされます。

## ビルド

プロダクション用のビルド：

```bash
npm run build
```

ビルド結果は `dist/` ディレクトリに出力されます。

## プレビュー

ビルドしたアプリケーションを確認：

```bash
npm run preview
```

## 機能

### カンバンボード
- タスクを「Todo」「In Progress」「Done」の3つの列で管理
- ドラッグ&ドロップでタスクを移動（実装予定）

### タスク管理
- **新規作成**: フォームからタスクを追加
- **編集**: タスク情報を更新
- **削除**: 不要なタスクを削除
- **ステータス変更**: タスクのステータスを変更

## コンポーネント

### KanbanBoard
カンバンボード全体を管理するメインコンポーネント

### KanbanColumn
個別の列を表示するコンポーネント

### TaskCard
タスク情報を表示するコンポーネント

### TaskForm
タスク作成/編集用フォームコンポーネント

## API通信

`src/api/todos.js` でバックエンドとの通信を管理しています。

```javascript
// 例
import { getTodos, createTodo, updateTodo, deleteTodo } from './api/todos'
```

## 技術スタック

- **Vue 3**: UIフレームワーク
- **Vite**: ビルドツール
- **CSS**: スタイリング

## 環境変数

APIベースURLは環境に合わせて設定してください。デフォルトは `http://localhost:8080` です。

## 開発時の注意

- バックエンド（BFF）が起動していることを確認してください
- ブラウザの開発者ツールコンソールでエラーを確認できます
- Vue DevTools拡張機能の使用を推奨します

## ライセンス

MIT
