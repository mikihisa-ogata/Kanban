# Copilot instructions

## プロジェクト概要
本プロジェクトは、Goで実装されたREST APIとVueフロントエンドから構成されるカンバンボードでTODOリストを管理するWebアプリケーションである。
開発および実行環境はDockerおよびDocker Composeを利用する。

---

## 技術スタック
- Backend: Go
- Frontend: Vue 3
- Database: MySQL（未実装）
- Container: Docker / Docker Compose（未実装）

---

## バックエンドアーキテクチャ
バックエンドはクリーンアーキテクチャを採用する。

構成：
Handler -> Service -> Repository -> Database

### 各層の責務
- Handler: HTTPリクエストおよびレスポンス処理のみ担当する
- Service: ビジネスロジックを担当する
- Repository: データベースアクセスのみ担当する

### ルール
- Handlerにビジネスロジックを書かない
- Serviceから直接DB操作を行わない
- RepositoryのみDBアクセスを行う

---

## フロントエンド構成ルール
Vueプロジェクトでは以下の構成を基本とする。

- `/api` : API通信処理
- `/components` : 再利用可能なUI部品
- `/pages` : ページ単位のコンポーネント
- `/stores` : 状態管理

### フロントエンドルール
- UIとAPI通信処理を分離する
- コンポーネントは小さく保つ
- Composition APIを優先して使用する

---

## コーディング規約

### Go
- 変数名はcamelCaseを使用する
- 小さな関数に分割する
- エラーは必ず呼び出し元へ返却する
- グローバル変数は使用しない

### Vue
- コンポーネントの責務を明確に分ける
- API通信は直接コンポーネントに書かない
- 再利用可能な処理は共通化する

---

## Docker運用ルール
- サービスごとにコンテナを分離する
- ローカル開発はdocker-composeを使用する
- 環境変数は.envファイルで管理する

---

## ディレクトリ構成方針
既存ディレクトリ構造を優先し、新しい構造を勝手に追加しないこと。

---

## Copilotへの指示
Copilotは以下を守ること。

- 既存コードスタイルに合わせる
- 新しいフレームワークを導入しない
- 既存のServiceやRepositoryを再利用する
- 不必要に複雑なコードを生成しない
- 可読性を優先する

---

## 禁止事
