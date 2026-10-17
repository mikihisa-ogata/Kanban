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

Dockerを使用することで、環境構築なしでサーバーを起動できます。`bff` と `mysql`（MySQL 8.4、ポート 3306）が起動し、データは名前付きボリューム `mysql-data` に、添付ファイルの本体は名前付きボリューム `attachments-data`（コンテナ内の `/app/attachments`）に保存されます。

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

添付ファイルの本体は、環境変数 `ATTACHMENTS_DIR` のディレクトリに保存します（未設定の場合は起動したディレクトリの `attachments/`。`bff/` で起動すれば `bff/attachments/`）。ディレクトリは最初の添付のときに作成します。`bff/attachments/` は `.gitignore` 済みです。データベースだけをバックアップ・移行すると添付ファイルの本体が欠けるので、このディレクトリも一緒に扱ってください。

### 最初のユーザーを作る

API はログインが必要です（ログインしていなければ 401 を返します）。ユーザーを追加する前のデータベースでも、起動すれば `users`・`sessions` テーブルを作るので、次の手順で最初のユーザーを作ります。

1. bff を新しいコードで起動し直す（`go run ./cmd/main.go` を止めて起動し直す。Docker の場合は `docker compose up -d --build`。`docker compose down -v` はデータのボリュームが消えるので使わない）。
2. フロントを起動し直す（`pnpm dev` を止めて起動し直す。ログイン画面などの新しいページを認識させるため）。
3. http://localhost:3000 を開くとログイン画面に移るので、「サインアップ」から表示名・メールアドレス・パスワード（8 文字以上）を入れて登録する。登録するとそのままログインし、既存のボード・課題がそのまま見えます。

既存のデータは、どのユーザーからも同じように見えます（ユーザーごとの権限はありません）。ユーザーの追加も同じくサインアップ画面から行います。

## API エンドポイント

### 認証

```
POST /auth/signup   {"displayName": "山田 太郎", "email": "taro@example.com", "password": "8 文字以上"}
POST /auth/login    {"email": "taro@example.com", "password": "..."}
POST /auth/logout
GET  /auth/me
```

認証はセッションの Cookie（`kanban_session`。HttpOnly・SameSite=Lax・有効期間 30 日）で行います。`/auth/*` 以外の API は、有効なセッションの Cookie がなければ 401（`{"error": "ログインしてください"}`）を返します。ブラウザからは資格情報付き（`fetch` の `credentials: 'include'`）で送ってください。CORS は `http://localhost:3000`・`http://localhost:5173` からの資格情報付きリクエストを許可しています。

- `POST /auth/signup` はユーザーを作成してログインし、201 でユーザーを `user` に入れて返します（Cookie を付けます）。表示名が空・50 文字超、メールアドレスの形式が不正、パスワードが 8 文字未満・72 バイト超なら 400、メールアドレスが登録済みなら 409 を返します。
- `POST /auth/login` はログインし、200 でユーザーを `user` に入れて返します（Cookie を付けます）。メールアドレスが登録されていない場合とパスワードが違う場合は、どちらも 401（`メールアドレスまたはパスワードが正しくありません`）を返します。
- `POST /auth/logout` はセッションを削除して Cookie を消します。ログインしていなくても 200 を返します。
- `GET /auth/me` はログイン中のユーザーを返します。ログインしていなければ 401 を返します。

```json
{
  "ID": 1,
  "DisplayName": "山田 太郎",
  "Email": "taro@example.com",
  "AvatarColor": "#0052CC",
  "CreatedAt": "2026-10-12T01:00:00Z"
}
```

- メールアドレスは前後の空白を除いて小文字にして保存します。大文字・小文字の違いだけのアドレスは同じユーザーです。
- パスワードは bcrypt のハッシュにして保存し、レスポンスには含めません。セッションのトークンも SHA-256 のハッシュにして保存します。
- `AvatarColor` はアバター（表示名の頭文字の丸）の色で、メールアドレスから決まります。

### ユーザー一覧

```
GET /users
```

すべてのユーザーを表示名の順の配列で返します（担当者・報告者を選ぶ候補）。各要素の形式は `GET /auth/me` と同じです（パスワードのハッシュは含みません）。

### ウォッチと通知

```
GET    /todos/:id/watchers
POST   /todos/:id/watch
DELETE /todos/:id/watch
GET    /notifications
POST   /notifications/:id/read
POST   /notifications/read-all
```

- `GET /todos/:id/watchers` は課題（`:id`）のウォッチャーを、表示名の順にユーザーの概要（`ID`・`DisplayName`・`AvatarColor`）の配列で返します。
- `POST /todos/:id/watch` はログイン中のユーザーをウォッチャーにし、`DELETE` は外します。どちらもウォッチャーの配列を返します。課題がなければ 404 です。
- 課題を作成したユーザー・担当者・コメントしたユーザーは、自動でウォッチャーになります。

ウォッチ中の課題が他のユーザーに変更・コメントされると、そのユーザー以外のウォッチャーに通知を作ります。

- 変更は、変更履歴を記録する操作すべてが対象です（`PUT /todos/:id` のほか、ドラッグでの移動、エピックやスペースの削除、作業ログによる残り時間の調整など）。
- 1 回の操作で変わった項目は 1 件の通知にまとめます。
- 新しい担当者には、割り当てられたことも通知します。

#### メンション

コメントと課題の説明（Markdown）には、`@[表示名](user:ユーザー ID)`（例: `@[山田 太郎](user:1)`）の形でユーザーをメンションできます。

- メンションされたユーザーに `mentioned` の通知を作ります（自分へのメンションと、存在しないユーザーは除きます）。
- コメント・説明の編集では、新しくメンションされたユーザーだけに通知します。
- メンションされたウォッチャーには、同じコメントの `commented` の通知を重ねて作りません（説明の変更の `updated` は別に作ります）。
- メンションだけではウォッチャーになりません。

`GET /notifications` はログイン中のユーザーへの通知を、新しい順に最大 50 件と未読の数で返します。

```json
{
  "Unread": 1,
  "Notifications": [
    {
      "ID": 3,
      "UserID": 2,
      "TodoID": 1,
      "IssueKey": "KAN-1",
      "IssueTitle": "ログイン画面を作る",
      "ActorID": 1,
      "Actor": { "ID": 1, "DisplayName": "山田 太郎", "AvatarColor": "#0052CC" },
      "Kind": "updated",
      "Detail": "status,assignee",
      "Read": false,
      "CreatedAt": "2026-10-12T03:00:00Z"
    }
  ]
}
```

- `Kind` は `updated`（変更）・`commented`（コメント）・`mentioned`（コメントか説明でのメンション）のいずれかです。
- `Detail` は、`updated` なら変わった項目名（変更履歴の `Field`）を `,` でつないだものです（`statusCategory`・`key` は含めません）。`commented`・`mentioned` ならコメント・説明の冒頭です（メンションは `@表示名` にし、空白をまとめ、100 文字を超えると `…` で切ります）。
- `IssueKey`・`IssueTitle` は取得した時点の課題キーと要約です。課題を削除すると、その課題の通知とウォッチャーも削除します。
- `POST /notifications/:id/read` は通知を既読にします。ログイン中のユーザーへの通知でなければ 404 です。`POST /notifications/read-all` はすべて既読にします。

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
    "StatusID": 3,
    "StatusName": "待ち状況",
    "StatusCategory": "InProgress",
    "IssueType": "Task",
    "Priority": "Medium",
    "Labels": ["backend", "ui"],
    "StoryPoints": 3,
    "OriginalEstimate": 960,
    "RemainingEstimate": 600,
    "TimeSpent": 360,
    "ParentID": 0,
    "EpicID": 0,
    "Description": "タスクの説明（任意。改行を含められる）",
    "SpaceID": 0,
    "Number": 3,
    "Key": "TASK-3",
    "AssigneeID": 2,
    "ReporterID": 1,
    "SprintID": 0,
    "FixVersionIDs": [2],
    "ComponentIDs": [4],
    "Assignee": { "ID": 2, "DisplayName": "佐藤 花子", "AvatarColor": "#00875A" },
    "Reporter": { "ID": 1, "DisplayName": "山田 太郎", "AvatarColor": "#0052CC" },
    "CommentCount": 2,
    "Links": [
      {
        "ID": 7,
        "Relation": "is blocked by",
        "Issue": { "ID": 4, "Key": "KAN-4", "Title": "先に終わらせるタスク", "Status": "InProgress", "StatusName": "進行中", "StatusCategory": "InProgress", "IssueType": "Task", "Priority": "High" }
      }
    ],
    "CreatedAt": "2026-10-10T12:00:00Z",
    "UpdatedAt": "2026-10-10T12:30:00Z"
  }
]
```

`StatusID` は課題のステータス（ワークフローのステータスの ID。`GET /statuses` を参照）で、`StatusName`・`StatusCategory` はその名前とカテゴリ（`ToDo`・`InProgress`・`Done`）です。`Done` はステータスのカテゴリが `Done` なら `true` になります（リクエストの `done` は使いません）。`Status` はワークフローの導入前の固定のステータス（`Open`・`InProgress`・`Waiting`・`Closed`）で、互換のために返しています。既定のステータスはそれぞれの値、追加したステータスはカテゴリから決まります（To Do は `Open`、In Progress は `InProgress`、Done は `Closed`）。

`IssueType` は課題タイプで、`Task`（タスク）・`Bug`（バグ）・`Story`（ストーリー）・`Subtask`（サブタスク）のいずれかです。課題タイプの追加前に作られたタスクは `Task` になります。

`Priority` は優先度で、`Highest`（最高）・`High`（高）・`Medium`（中）・`Low`（低）・`Lowest`（最低）のいずれかです。既定は `Medium` で、優先度の追加前に作られたタスクも `Medium` になります。

`ParentID` はサブタスクの親課題の ID で、サブタスクでなければ `0` です。サブタスク（`IssueType` が `Subtask`）は必ず親課題を持ち、親課題と同じスペース・エピックに置かれます。サブタスクの下にサブタスクは作れません。

`StoryPoints` はストーリーポイント（見積もり）で、0 以上 999999.99 以下の数値（小数は第 2 位まで）です。未設定は `null` で、`0` とは区別します。

`Labels` はラベル（Jira の自由入力のタグ）の名前順の配列で、ラベルがなければ `[]` です。ラベル名は空白を含まない 255 文字以内で、前後の空白は除きます。大文字・小文字の違いだけのラベルは同じラベルとして扱い、最初に作られた名前を使います（例: `UI` を付けても、既に `ui` があれば `ui` になります）。

`Key` は課題キー（`<スペースのキー>-<Number>`。例: `KAN-12`）です。`Number` はスペース内の連番で、サーバーが採番します（リクエストに含めても無視します）。

- 番号はスペースごとに 1 から振ります。タスクを削除しても、その番号は再利用しません。
- スペース未割り当てのタスクは、接頭辞 `TASK` の番号を共有します（例: `TASK-3`）。`TASK` はスペースのキーには使えません。
- `PUT` でスペースを変えると、Jira で課題を別のプロジェクトへ移したときと同じく、移動先のスペースの次の番号になります（キーが変わります）。エピックのスペース変更やスペースの削除で子タスクのスペースが変わった場合も同じです。古いキーは残りません。

`OriginalEstimate`（初期見積もり）と `RemainingEstimate`（残り時間）は分の整数で、未設定は `null` です。`TimeSpent` は作業ログの合計（分）です（リクエストに含めても無視します）。

`SprintID` は課題のスプリント（`GET /sprints` を参照）で、`0` はバックログ（どのスプリントにも入っていない）です。変えるには `PUT /todos/:id/sprint` を使います（`PUT /todos/:id` では変わりません）。

`FixVersionIDs` は課題の修正バージョン（`GET /versions` を参照）の ID 順の配列で、なければ `[]` です。変えるには `PUT /todos/:id/fix-versions` を使います（`PUT /todos/:id` では変わりません）。

`ComponentIDs` は課題のコンポーネント（`GET /components` を参照）の ID 順の配列で、なければ `[]` です。変えるには `PUT /todos/:id/components` を使います（`PUT /todos/:id` では変わりません）。

`AssigneeID` は担当者、`ReporterID` は報告者のユーザー ID で、`0` はなし（未割り当て）です。`Assignee`・`Reporter` はそのユーザーの概要（表示名とアバターの色）で、なしなら `null` です。報告者は課題を作成したユーザーになります。ユーザー機能の追加前に作られた課題は、どちらもなしです。

`CommentCount` はその課題のコメント数です（リクエストに含めても無視します）。

`Links` は課題リンクの配列で、リンクがなければ `[]` です。`Relation` はこの課題から見たリンクの名前、`Issue` はリンク相手の課題の概要です。リンクは片方の課題に作ると、相手の課題にも逆向きの名前で返ります。

| 種類 | リンク元から見た名前 | リンク先から見た名前 |
|---|---|---|
| ブロック | `blocks` | `is blocked by` |
| 関連 | `relates to` | `relates to` |
| 重複 | `duplicates` | `is duplicated by` |
| 複製 | `clones` | `is cloned by` |

並びは上の表の順（`blocks` → `is blocked by` → `relates to` → …）で、同じ名前のリンクは作成順です。`POST`・`PUT` のレスポンスの `todo` と `GET /issues/:key` にも含まれます。

`CreatedAt`（作成日時）と `UpdatedAt`（最終更新日時）は UTC の RFC 3339 形式で、サーバーが記録します（リクエストに含めても無視します）。`UpdatedAt` は `PUT` で変わり、並び替え（`move`）では変わりません。

### 新規TODOを作成

```
POST /todos
```

**リクエスト:**
```json
{
  "Title": "新しいタスク",
  "Deadline": "2026-03-31（省略可。省略すると期限なし）",
  "Status": "Open（StatusID か Status。省略するとスペースの To Do の先頭）",
  "IssueType": "Bug（省略可。省略すると Task）",
  "Priority": "High（省略可。省略すると Medium）",
  "Labels": ["ui", "不具合"],
  "StoryPoints": 0.5,
  "ParentID": 0,
  "Description": "タスクの説明（省略可）",
  "EpicID": 0,
  "SpaceID": 0,
  "AssigneeID": 2
}
```

`AssigneeID`（省略可）は担当者のユーザー ID で、省略か `0` は未割り当てです。存在しないユーザーなら 400 を返します。報告者はリクエストしたユーザー（ログイン中のユーザー）になります。

**レスポンス（201）:** 作成したタスクを `todo` に入れて返します（形式は `GET /todos` の要素と同じ。採番した課題キー `Key` を含む）。

```json
{
  "message": "Todoの作成に成功しました",
  "todo": { "ID": 12, "Title": "新しいタスク", "Key": "KAN-12", "...": "..." }
}
```

`ParentID`（省略可）に課題の ID を指定すると、その課題のサブタスクを作ります。課題タイプは `Subtask` になり（`IssueType` は省略するか `Subtask` にする）、`SpaceID`・`EpicID` は送った値にかかわらず親課題と同じになります。親課題が存在しない、または親課題がサブタスクの場合は 400 を返します。`ParentID` なしで `IssueType` に `Subtask` を指定した場合も 400 です。

### TODOを更新

```
PUT /todos/:id
```

**リクエスト:**
```json
{
  "Title": "更新されたタスク",
  "Deadline": "2026-03-31（省略すると期限なしになる）",
  "StatusID": 2,
  "Status": "InProgress（StatusID の代わりに導入前の値でも指定できる。どちらも省略すると今のステータスのまま）",
  "IssueType": "Story（省略すると今の課題タイプのまま）",
  "Priority": "Low（省略すると今の優先度のまま）",
  "Labels": ["ui"],
  "StoryPoints": null,
  "Description": "タスクの説明（省略すると空になる）",
  "EpicID": 0,
  "SpaceID": 0,
  "AssigneeID": 2,
  "ReporterID": 1
}
```

`AssigneeID`・`ReporterID` は担当者・報告者のユーザー ID です。省略すると今のまま、`null` か `0` でなし（未割り当て）にします。存在しないユーザーや、ID でない値は 400 を返します。

**レスポンス（200）:** 更新後のタスクを `todo` に入れて返します（形式は `GET /todos` の要素と同じ）。スペースを変えた場合は、新しい課題キーになっています。

```json
{
  "message": "Todoの更新に成功しました",
  "todo": { "ID": 12, "Title": "更新されたタスク", "Key": "KAN-12", "...": "..." }
}
```

### 課題キーでTODOを取得

```
GET /issues/:key
```

課題キー（例: `KAN-12`、スペース未割り当てなら `TASK-3`）のタスクを 1 件返します（形式は `GET /todos` の要素と同じ）。英小文字のキー（`kan-12`）も同じ課題として扱います。存在しないキーや形式が正しくないキーには 404 を返します。

```json
{ "error": "課題が見つかりません" }
```

### 課題を検索（課題一覧）

```
GET /issues/search?space=DEV&space=OPS&type=Bug&status=進行中&assignee=2&assignee=unassigned&label=ui&text=ログイン&sort=priority&order=asc&page=1&perPage=50
```

Jira の課題ナビゲーターの一覧です。全スペースを横断して、絞り込み・並べ替え・ページングを DB で行い、1 ページ分を返します。

```json
{ "Issues": [ { "ID": 12, "Key": "DEV-12", "Title": "...", "...": "GET /todos と同じ形（Links は null）" } ], "Total": 120, "Page": 1, "PerPage": 50 }
```

- 条件はどれも省略でき、同じ名前を繰り返すとそのいずれか（OR）、種類どうしはすべて（AND）を満たす課題を返します。サブタスクも含みます。
  - `space`: スペースのキー（大文字・小文字は区別しない）。`TASK` はスペース未割り当て。存在しないキーはどの課題にも合いません。
  - `type`: 課題タイプ（`Task`・`Bug`・`Story`・`Subtask`）。
  - `status`: ステータスの名前（スペースをまたいで同じ名前のステータスをまとめて選びます）。
  - `assignee`: 担当者のユーザー ID か `unassigned`（未割り当て）。
  - `label`: ラベルの名前（大文字・小文字は区別しない）。
  - `text`: 要約・説明に含まれる文字列か、課題キーそのもの（例: `dev-12`）。
- `sort` は `key`・`type`・`summary`・`status`・`assignee`・`reporter`・`priority`・`created`・`updated`・`due`（期限）、`order` は `asc`・`desc` です。`sort` を省略すると作成日の新しい順です。優先度は高い順が `asc` で、担当者・報告者・期限のない課題は `asc`・`desc` とも末尾に並びます。
- `page` は 1 から、`perPage` は 1〜100（既定 50）です。誤った値は 400 です。

### 課題を JQL で検索（詳細検索）

```
GET /issues/jql?jql=project = DEV AND status != Done AND assignee = currentUser() ORDER BY priority DESC&page=1&perPage=50
```

Jira の JQL の簡易版で課題を検索します。レスポンスは `GET /issues/search` と同じ形です。`jql` が空なら全課題を作成日の新しい順に返します。

- 項目: `project`（スペースのキーか名前。`TASK` はスペース未割り当て）・`type`（`issuetype`）・`status`・`statusCategory`（`ToDo`・`"In Progress"`・`Done`）・`priority`・`assignee`・`reporter`（ユーザー ID・メール・表示名）・`labels`・`epic`（`"epic link"`。ID か名前）・`sprint`（ID か名前）・`fixVersion`（ID か名前）・`created`・`updated`・`due`（`duedate`）・`text`・`issuekey`（`key`・`issue`。課題キー）。項目名・キーワード・名前は大文字・小文字を区別しません。
- 演算子: `=`・`!=`・`IN (…)`・`NOT IN (…)`・`~`（`text` だけ。要約・説明・課題キー）・`>`・`>=`・`<`・`<=`（`priority`・`created`・`updated`・`due`。優先度は高いほど大きい）・`IS EMPTY`・`IS NOT EMPTY`（`assignee`・`reporter`・`labels`・`epic`・`sprint`・`fixVersion`・`due`）。条件は `AND`・`OR`・`NOT` と括弧で組み合わせます（`AND` が `OR` より先）。`NOT IN` と `!=` は値のない課題には合いません（Jira と同じ）。
- 値に空白や記号を含むときは `"…"` か `'…'` で囲みます。関数: `currentUser()`・`issueHistory()`（`issuekey` に使う。自分が詳細を表示した課題）・`openSprints()`・`closedSprints()`・`futureSprints()`・`releasedVersions()`・`unreleasedVersions()`・`now()`・`startOfDay()`。日付は `2026-10-01`・`"2026-10-01 09:00"`、相対日付は `-7d`・`2w`・`-1h`・`-30m`（現在からの差）。日付だけの値はその日全体を表します（`created = 2026-10-01` はその日に作成）。
- `IN` の後の関数は括弧なしでも書けます（例: `issuekey IN issueHistory()`、`sprint IN openSprints()`）。
- `ORDER BY 項目 [ASC|DESC], …` で並べ替えます（項目は `GET /issues/search` の `sort` と同じ。`priority DESC` が高い順）。`lastViewed` は自分が最後に表示した日時の順です（表示していない課題は末尾）。
- 値はすべてプレースホルダで SQL に渡します。
- JQL の誤り（構文・存在しない項目や値）は 400 で、位置（先頭からの文字数。0 から）を返します。

```json
{ "error": "14 文字目: 対応する「(」がない「)」があります", "position": 13 }
```

### ラベル一覧を取得

```
GET /labels
```

いずれかのタスクに付いているラベルの名前を、名前順の配列で返します（入力欄の補完候補に使います）。どのタスクにも付いていないラベルは返しません。

```json
["backend", "ui", "不具合"]
```

### TODOを削除

```
DELETE /todos/:id
```

サブタスクを持つ課題を削除すると、サブタスクも一緒に削除します。削除した課題（とサブタスク）の課題リンク・コメント・変更履歴・添付ファイル・作業ログも削除します。

### 課題を一括変更・一括削除

```
POST /issues/bulk-update   {"ids": [1, 2, 3], "statusId": 5, "assigneeId": 2, "priority": "High", "addLabels": ["ui"], "removeLabels": ["old"], "sprintId": 4, "epicId": 0, "addFixVersionIds": [7], "removeFixVersionIds": [6]}
POST /issues/bulk-delete   {"ids": [1, 2, 3]}
```

- `bulk-update` は `ids` の課題をまとめて変更し、200 で `{"message": "...", "count": 2}`（内容が変わった課題の数）を返します。省略した項目は今のままです。
  - `statusId` はワークフローのステータスの ID です。別のスペースの課題には、名前（なければカテゴリ）が同じステータスを当てます。
  - `assigneeId`・`sprintId`・`epicId` は `null` か `0` で「なし」（スプリントはバックログ）にします。スプリントは完了していない、課題と同じスペースのもの、エピック・修正バージョンも課題と同じスペースのものを選びます。アーカイブ済みのバージョンは付けられません。
  - `addLabels`・`removeLabels` はラベルを付ける・外す（外すほうは大文字・小文字を区別しない）、`addFixVersionIds`・`removeFixVersionIds` は修正バージョンを付ける・外す指定です。
  - サブタスクのスプリント・エピックは親課題に合わせます（サブタスクに指定しても変えず、親課題を変えるとサブタスクも移します）。
  - 検証はすべて書き込む前に行い、1 件でも当てられない変更があれば、どの課題も変えずに 400（存在しない課題があれば 404）を返します。書き込みは変更履歴（`sprint`・`fixVersions` を含む）と新しい担当者のウォッチャー追加を含めて 1 つのトランザクションで行います。変わった課題のウォッチャーには通知を作ります。
- `bulk-delete` は `ids` の課題（とそのサブタスク）を 1 つのトランザクションでまとめて削除し、200 で `{"message": "...", "count": 3}`（サブタスクを含めた削除した課題の数）を返します。存在しない課題が混じっていれば、どれも削除せずに 404 を返します。削除するものは `DELETE /todos/:id` と同じです。
- どちらも `ids` が空なら 400、`bulk-update` で変更する項目がなければ 400 です。

### 課題リンクを作成

```
POST /todos/:id/links
```

**リクエスト:**
```json
{
  "relation": "is blocked by",
  "targetId": 4
}
```

`:id` の課題から `targetId` の課題へのリンクを作ります。`relation` は `:id` の課題から見た名前（`GET /todos` の `Links` の表のいずれか）です。上の例は「4 の課題が `:id` の課題をブロックする」リンクになり、4 の課題からは `blocks` として見えます。

**レスポンス（201）:** リンクを含む `:id` の課題を `todo` に入れて返します（形式は `GET /todos` の要素と同じ）。

- `:id` の課題が存在しなければ 404 を返します。
- `relation` が表にない名前、`targetId` の課題が存在しない、`targetId` が `:id` と同じ場合は 400 を返します。
- 同じリンクが既にあれば 409 を返します。`relates to` は向きを区別しません（A relates to B があれば、B relates to A は作れません）。`blocks` などは向きが違えば別のリンクです。

### 課題リンクを削除

```
DELETE /links/:id
```

`:id` は `Links` の `ID` です。リンクは両方の課題から消えます。存在しなければ 404 を返します。

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

この並び順は、ボードの列の中の並びと、バックログの順位（Jira のランク）を兼ねます。別々に持つと、バックログで上げた課題がボードでは下にあるといった食い違いが起きるため、1 つにしています。バックログで順位を変えるとボードの並びも変わり、その逆も同じです。並び順は全スペースで 1 つなので、スペースの間や完了した課題との前後も保ちます。

`EpicID` / `SpaceID` は省略すると 0（未割り当て）になります。`IssueType` に `Task`・`Bug`・`Story` 以外、`Priority` に上の 5 つ以外、`Labels` に空や空白を含むラベルを指定すると 400 を返します。`Labels` は、作成で省略するとラベルなし、更新で省略すると今のラベルのまま、`[]` を送るとすべて外します。`StoryPoints` は、作成で省略または `null` にすると未設定、更新で省略すると今の値のまま、`null` を送ると未設定に戻します。範囲外や小数第 3 位以下を含む値は 400 を返します。

ステータスは `statusId`（ワークフローのステータスの ID）か、導入前の `status`（`Open`・`InProgress`・`Waiting`・`Closed`）で指定します。`statusId` を優先します。

- 作成で指定がなければ、課題のスペースのワークフローのカテゴリ To Do の先頭になります。更新で指定がなければ今のままです。
- `status` で指定すると、スペースの同じ導入前のステータス（なければ同じカテゴリの先頭）になります。ただし今のステータスの `Status` と同じ値を送った場合は、今のステータスのままです。
- 存在しない `statusId`、知らない `status` は 400 を返します。
- スペースを変えると（エピックのスペース変更やスペースの削除で変わった場合も）、移動先のワークフローの同じ名前のステータス、なければ同じカテゴリの先頭のステータスになります。別のスペースのステータスの `statusId` を指定した場合も同じように対応付けます。

`PUT` の `originalEstimate`・`remainingEstimate` は「`1w 2d 3h 30m`」形式の文字列で指定します（w=週・d=日・h=時間・m=分。1d=8h、1w=5d。順番・空白の有無は問わず、`1.5h` のような小数も使えます）。省略すると今の値のまま、`null` か空文字で未設定に戻します。単位のない数値など読めない値は 400 を返します。残り時間が未設定のまま初期見積もりを設定すると、残り時間は「初期見積もり − 記録済みの作業時間」になります。

`PUT` では親課題（`ParentID`）は変えられません（リクエストに含めても無視します）。サブタスクの `SpaceID`・`EpicID` は親課題と同じままになり、サブタスクと他の課題タイプを相互に変えようとすると 400 を返します。親課題の `SpaceID`・`EpicID` を変えると、そのサブタスクも同じスペース・エピックに移ります（スペースが変わると課題キーも変わります）。エピックを指定する場合、タスクの `SpaceID` はそのエピックの `SpaceID` と同じでなければなりません（異なると 400）。

### 作業ログ

```
GET    /todos/:id/worklogs
POST   /todos/:id/worklogs   {"timeSpent": "2h 30m", "startedAt": "2026-10-11T10:00:00+09:00", "note": "調査"}
PUT    /worklogs/:id         （POST と同じ形）
DELETE /worklogs/:id
```

`GET` は課題（`:id`）の作業ログを開始日時の順（古い順）の配列で返します。

```json
[
  {
    "ID": 1,
    "TodoID": 3,
    "Minutes": 150,
    "StartedAt": "2026-10-11T01:00:00Z",
    "Note": "調査",
    "CreatedAt": "2026-10-11T02:30:00Z",
    "UpdatedAt": null
  }
]
```

- `timeSpent` は「`1w 2d 3h 30m`」形式で 1 分以上、`startedAt` は RFC 3339 形式で必須、`note` は任意（10000 文字まで）です。不正な値は 400 を返します。
- 残り時間は自動で調整します。記録すると作業時間の分だけ減らし（0 未満にはしません）、作業時間を直すと増減の分だけ、削除するとその分だけ戻します。残り時間が未設定で初期見積もりがあれば「初期見積もり − 記録済み」にし、どちらも未設定なら変えません。
- `POST`・`PUT` に `remainingEstimate`（`1w 2d` 形式。`null` か空文字は未設定）を付けると、自動の調整の代わりにその値にします。
- 残り時間の変化は変更履歴（`remainingEstimate`）に残ります。課題を削除すると作業ログも削除します。課題・作業ログが存在しなければ 404 を返します。

### 添付ファイル

```
GET    /todos/:id/attachments
POST   /todos/:id/attachments     multipart/form-data の file フィールドにファイル
GET    /attachments/:id/file      ファイル本体（?inline=1 で画像をブラウザに表示）
DELETE /attachments/:id
```

`GET /todos/:id/attachments` は課題の添付ファイルの情報を添付順（古い順）の配列で返します。

```json
[
  {
    "ID": 1,
    "TodoID": 3,
    "FileName": "画面.png",
    "ContentType": "image/png",
    "Size": 48213,
    "CreatedAt": "2026-10-11T01:00:00Z"
  }
]
```

- 1 ファイルの上限は 10MB です。超えると 413、空のファイルは 400 を返します。
- `POST` は 201 で添付したファイルの情報を `attachment` に入れて返します。ファイル名はディレクトリ部分を除いて保存します。
- `ContentType` はアップロード時の申告ではなく、ファイルの中身から判定します。
- `GET /attachments/:id/file` は通常ダウンロードさせます（`Content-Disposition: attachment`）。`?inline=1` を付けると、PNG・JPEG・GIF・WebP・BMP の画像に限りブラウザでそのまま表示します（サムネイル用）。SVG や HTML などは `inline=1` でもダウンロードになります。
- 課題・添付ファイルが存在しなければ 404 を返します。課題を削除すると、その課題（とサブタスク）の添付ファイルも本体ごと削除します。

### 変更履歴

```
GET /todos/:id/history
```

課題（`:id`）の変更履歴を記録順（古い順）の配列で返します。課題が存在しなければ 404 を返します。

```json
[
  { "ID": 1, "TodoID": 3, "Field": "created", "OldValue": null, "NewValue": null, "UserID": 1, "User": { "ID": 1, "DisplayName": "山田 太郎", "AvatarColor": "#0052CC" }, "CreatedAt": "2026-10-11T01:00:00Z" },
  { "ID": 2, "TodoID": 3, "Field": "status", "OldValue": "オープン", "NewValue": "進行中", "UserID": 1, "User": { "ID": 1, "DisplayName": "山田 太郎", "AvatarColor": "#0052CC" }, "CreatedAt": "2026-10-11T01:05:00Z" },
  { "ID": 3, "TodoID": 3, "Field": "storyPoints", "OldValue": "3", "NewValue": "5", "UserID": 0, "User": null, "CreatedAt": "2026-10-11T01:05:00Z" },
  { "ID": 4, "TodoID": 3, "Field": "sprint", "OldValue": null, "NewValue": "DEV スプリント 1", "OldRefID": 0, "NewRefID": 3, "UserID": 1, "User": { "ID": 1, "DisplayName": "山田 太郎", "AvatarColor": "#0052CC" }, "CreatedAt": "2026-10-12T02:00:00Z" }
]
```

`UserID` は変更したユーザーの ID、`User` はその概要です。`OldRefID`・`NewRefID` は項目が指すものの変更前・変更後の ID で、`sprint` だけが持ちます（スプリントの ID。`0` はバックログ）。それ以外の項目と、ID を残す前の `sprint` の記録は `null` です。ユーザー機能の追加前の記録は `UserID` が `0`、`User` が `null`（変更者なし）です。

課題の作成（`Field` が `created`）と、項目の変更を 1 項目 1 件で記録します。値は変更時点の文字列で、`null` は「なし」です。同じ内容のままの更新は記録しません。

| Field | 値 |
|---|---|
| `title` / `description` | 要約・説明 |
| `status` | ステータスの名前（ワークフローの導入前の記録は `Open`・`InProgress`・`Waiting`・`Closed`） |
| `statusCategory` | `ToDo`・`InProgress`・`Done`（カテゴリが変わったときだけ） |
| `done` | `true`・`false`（ワークフローの導入前の記録だけ。今は完了はカテゴリで決まる） |
| `issueType` / `priority` | `IssueType`・`Priority` の値 |
| `labels` | ラベル名を `, ` でつないだもの |
| `storyPoints` | 数値（例: `0.5`） |
| `originalEstimate` / `remainingEstimate` | 分の整数（例: `90`） |
| `deadline` | `2026-03-31` 形式 |
| `epic` / `space` | 変更時点のエピック・スペースの名前 |
| `key` | 課題キー（スペースの変更で変わる） |
| `assignee` / `reporter` | 変更時点の担当者・報告者の表示名（`null` はなし） |
| `sprint` | 変更時点のスプリントの名前（`null` はバックログ）。`PUT /todos/:id/sprint`・一括変更・スプリントの完了（未完了の課題の移動）・削除で記録 |
| `fixVersions` | 変更時点の修正バージョンの名前を名前順に `, ` でつないだもの（一括変更でだけ記録） |

`PUT /todos/:id`・一括変更（`POST /issues/bulk-update`）のほか、エピックのスペース変更・エピックやスペースの削除・親課題の移動で課題が変わった場合も記録します。ステータス・ストーリーポイント・スプリントは値と日時を残すので、ある時点のステータスとポイントとスプリントの課題を後から再計算できます（バーンダウンで使います）。

### コメント

```
GET    /todos/:id/comments
POST   /todos/:id/comments   {"body": "Markdown の本文"}
PUT    /comments/:id         {"body": "編集後の本文"}
DELETE /comments/:id
```

`GET` は課題（`:id`）のコメントを投稿順（古い順）の配列で返します。

```json
[
  {
    "ID": 1,
    "TodoID": 3,
    "Body": "**確認しました**",
    "AuthorID": 2,
    "Author": { "ID": 2, "DisplayName": "佐藤 花子", "AvatarColor": "#00875A" },
    "CreatedAt": "2026-10-11T01:00:00Z",
    "UpdatedAt": "2026-10-11T01:05:00Z"
  }
]
```

- `Body` は Markdown の本文です。空や空白だけの本文、10000 文字を超える本文は 400 を返します。
- `CreatedAt` は投稿日時、`UpdatedAt` は最後に編集した日時です（UTC の RFC 3339 形式）。編集していないコメントの `UpdatedAt` は `null` です。
- `POST` は 201 で作成したコメントを、`PUT` は 200 で編集後のコメントを `comment` に入れて返します。
- 課題やコメントが存在しなければ 404 を返します。
- `AuthorID` は投稿者（投稿したときにログインしていたユーザー）の ID、`Author` はその概要です。ユーザー機能の追加前のコメントは `AuthorID` が `0`、`Author` が `null`（投稿者なし）です。

### ワークフローのステータス

```
GET /statuses
```

すべてのワークフローのステータス（ボードの列）を、スペース・並び順の順に返します。`SpaceID` が `0` のものは、スペース未割り当ての課題のワークフローです。`WIPLimit` は列の最大件数（WIP 制限）で、`null` は制限なしです。スペースを作ると、そのスペースに既定の 4 ステータスを作ります。スペースを削除すると、そのスペースのステータスも削除します（課題は未割り当てのワークフローへ移ります）。

```json
[
  { "ID": 1, "SpaceID": 0, "Name": "オープン", "Category": "ToDo", "Position": 1, "WIPLimit": null },
  { "ID": 2, "SpaceID": 0, "Name": "進行中", "Category": "InProgress", "Position": 2, "WIPLimit": 3 },
  { "ID": 3, "SpaceID": 0, "Name": "待ち状況", "Category": "InProgress", "Position": 3, "WIPLimit": null },
  { "ID": 4, "SpaceID": 0, "Name": "クローズ", "Category": "Done", "Position": 4, "WIPLimit": null }
]
```

### ワークフローの編集

```
POST   /spaces/:id/statuses          {"name": "レビュー", "category": "InProgress"}
PUT    /spaces/:id/statuses/order    {"statusIds": [5, 6, 13, 7, 8]}
PUT    /statuses/:id                 {"name": "コードレビュー"}
PUT    /statuses/:id/wip-limit       {"wipLimit": 3}
DELETE /statuses/:id?moveTo=8
```

スペース（`:id`）のワークフローのステータスを追加・並べ替え・名前変更・最大件数の設定・削除します。スペース未割り当てのワークフロー（`SpaceID` 0）は変更できません（400）。

- `POST` はスペースの末尾にステータスを追加し、201 で追加したステータスを `status` に入れて返します。`category` は `ToDo`・`InProgress`・`Done` のいずれかです。
- 名前は前後の空白を除いて 1〜50 文字で、同じスペースに同じ名前（大文字・小文字は区別しない）があれば 409 を返します。`PUT /statuses/:id` で変えられるのは名前だけです（カテゴリは課題の完了の扱いが変わるので変えられません）。
- `PUT /statuses/:id/wip-limit` は列の最大件数（Jira のカンバンの列制約）を設定し、200 で変更後のステータスを `status` に入れて返します。`wipLimit` は 1〜999 の整数で、`null` か `0` で制限なしに戻します。範囲外や整数でない値は 400 です。最大件数を超えてもボードの表示が変わるだけで、課題の移動は止めません。
- `PUT /spaces/:id/statuses/order` はボードの列の順です。スペースのステータスの ID をすべて 1 回ずつ指定します（過不足があれば 400）。
- `DELETE` は、そのステータスの課題があれば `moveTo`（同じスペースの別のステータスの ID）へ移してから削除します。課題があるのに `moveTo` がなければ 409、`moveTo` が別のスペースのステータスや自分自身なら 400 を返します。移した課題のステータスの変化は変更履歴に残ります。ワークフローの最後のステータスは削除できません（400）。

### スプリント

```
GET    /sprints
POST   /spaces/:id/sprints     {"name": "DEV スプリント 1"}
PUT    /sprints/:id            {"name": "DEV スプリント 1", "goal": "ログインを作る", "startDate": "2026-10-12", "endDate": "2026-10-26"}
POST   /sprints/:id/start      {"name": "DEV スプリント 1", "goal": "ログインを作る", "startDate": "2026-10-12", "endDate": "2026-10-26"}
POST   /sprints/:id/complete   {"moveTo": 5}
DELETE /sprints/:id
GET    /sprints/:id/issues
GET    /sprints/:id/burndown?tz=Asia/Tokyo
GET    /spaces/:id/velocity?tz=Asia/Tokyo
PUT    /todos/:id/sprint       {"sprintId": 3}
```

Jira のスクラムのスプリントです。スプリントはスペースに属し、状態 `State` は `future`（未開始）→ `active`（開始中）→ `closed`（完了）と進みます。

```json
[
  {
    "ID": 3, "SpaceID": 1, "Name": "DEV スプリント 1", "Goal": "ログインを作る", "State": "closed",
    "StartDate": "2026-10-12", "EndDate": "2026-10-26", "StartedAt": "2026-10-12T00:30:00Z", "CompletedAt": "2026-10-26T09:00:00Z", "CreatedAt": "2026-10-10T12:00:00Z",
    "CompletedIssues": 4, "IncompleteIssues": 1, "CompletedPoints": 13
  }
]
```

- `GET /sprints` はすべてのスプリントを、スペース・作成順に返します。`StartDate`・`EndDate` は期間（未設定は空文字）、`StartedAt` は開始した日時（未開始と、開始した日時を記録する前に開始したスプリントは `null`）、`CompletedAt` は完了した日時（未完了は `null`）です。`CompletedIssues`・`IncompleteIssues`・`CompletedPoints` は完了したときに記録した、完了した課題・完了していない課題の件数と、完了した課題のストーリーポイントの合計です（完了していないスプリントは 0）。
- `POST /spaces/:id/sprints` は未開始のスプリントを作り、201 で `sprint` に入れて返します。`name` を省略すると「`<スペースのキー>` スプリント `<スペースのスプリントの通し番号>`」になります。スペース未割り当て（`:id` が 0）は 400、存在しないスペースは 404 です。
- `PUT /sprints/:id` は名前・ゴール・期間を変えます（完了したスプリントは 400）。`POST /sprints/:id/start` は未開始のスプリントを、名前・ゴール・期間を決めて開始します。同じスペースに開始中のスプリントがあれば 409、未開始でなければ 400 です。名前は前後の空白を除いて 1〜100 文字、ゴールは 1000 文字以内、期間は `YYYY-MM-DD` で終了日を開始日以降にします。開始中・開始するスプリントでは期間が必須です（未開始なら両方空にできます）。
- `POST /sprints/:id/complete` は開始中のスプリントを完了します。そのときのスプリントの課題（サブタスクを除く）を、ストーリーポイントと完了していたか（ステータスのカテゴリが `Done`）とあわせて記録し、完了していない課題（とそのサブタスク）を `moveTo`（同じスペースの未開始のスプリントの ID。`0` か省略はバックログ）へ移します。完了した課題は完了したスプリントに残ります。開始中でなければ 400、`moveTo` が別のスペースのスプリントや未開始でないスプリントなら 400 です。
- `DELETE /sprints/:id` は未開始のスプリントを削除し、その課題をバックログに戻します（開始中・完了したスプリントは 400）。
- `GET /sprints/:id/issues` は完了したときに記録した課題 `[{"SprintID": 3, "TodoID": 12, "StoryPoints": 5, "Completed": true}]` を課題の ID 順に返します（`StoryPoints` は未設定なら `null`）。
- `GET /sprints/:id/burndown` は開始中・完了したスプリントのバーンダウン（Jira のスプリントバーンダウン）を、変更履歴から再計算して返します。未開始のスプリントは 400、存在しなければ 404 です。日付はクエリ `tz`（IANA のタイムゾーン名。省略するとサーバーのタイムゾーン、正しくなければ 400）で区切ります。

  ```json
  {
    "SprintID": 3, "Start": "2026-10-12T00:30:00Z", "End": "2026-10-26T15:00:00Z", "Until": "2026-10-20T03:00:00Z", "StartPoints": 13,
    "Events": [
      { "Time": "2026-10-13T02:00:00Z", "Date": "2026-10-13", "TodoID": 12, "Key": "DEV-12", "Title": "ログイン画面", "Kind": "completed", "Points": 5, "Change": -5, "Remaining": 8 }
    ],
    "Days": [
      { "Date": "2026-10-12", "Added": 0, "Removed": 0, "Estimate": 0, "Completed": 0, "Remaining": 13 },
      { "Date": "2026-10-13", "Added": 0, "Removed": 0, "Estimate": 0, "Completed": -5, "Remaining": 8 }
    ]
  }
  ```

  - 残りポイントは、スプリントに入っている未完了（ステータスのカテゴリが `Done` でない）の課題（サブタスクを除く）のストーリーポイントの合計です（未設定は 0）。`StartPoints` は起点 `Start`（開始した日時。記録がなければ開始日の 0 時）の残りポイントで、理想線は `Start` の `StartPoints` から計画の終わり `End`（終了日の翌日の 0 時）の 0 を結びます。`Until` は実績の終わり（完了した日時。開始中なら現在）です。
  - `Events` は起点から `Until` までに残りポイントを変えた出来事を起きた順に並べます。`Kind` は `added`（追加。スプリント中に作成した課題を含む）・`removed`（除外。スペースの移動を含む）・`completed`（完了）・`reopened`（再開）・`estimate`（未完了の課題のポイント変更）で、`Change` は残りポイントの増減、`Remaining` はその後の残りポイント、`Points` はその後の課題のポイント（未設定は `null`）です。追加・除外・完了・再開はポイントが 0 でも含めます。完了したスプリントでは、完了したときの未完了の課題の移動は含めません。
  - `Days` は開始日から終了日（実績が終了日より後まであればその日）までの日ごとの増減（`Completed` は完了で減った分と再開で増えた分の合計）と、その日の終わりの残りポイントです。まだ来ていない日の `Remaining` は `null` です。
  - 履歴のない課題は、今（完了したスプリントでは完了したときに記録した課題も）スプリントに入っていれば最初から入っていたものとし、履歴のない項目は今の値のままだったものとして数えます。
- `GET /spaces/:id/velocity` はスペースのベロシティ（Jira のベロシティチャート）を返します。完了した直近 7 つのスプリントを完了した順（古い順）に並べ、`Average` は `Completed` の平均です（完了したスプリントがなければ `Sprints` は空、`Average` は 0）。存在しないスペースは 404、`tz` はバーンダウンと同じです（開始日しか記録がないスプリントの起点を、そのタイムゾーンの 0 時として読みます）。

  ```json
  {
    "Sprints": [
      { "SprintID": 3, "Name": "DEV スプリント 1", "StartDate": "2026-10-12", "EndDate": "2026-10-26", "CompletedAt": "2026-10-26T09:00:00Z", "Commitment": 18, "Completed": 13 }
    ],
    "Average": 13
  }
  ```

  - `Commitment` は開始したときにスプリントに入っていた課題（サブタスクを除く。開始前に完了していた課題も含む）のストーリーポイントの合計で、バーンダウンと同じく変更履歴から再計算します。`Completed` は完了したときに記録した、完了していた課題のストーリーポイントの合計です（どちらも未設定は 0）。
- `PUT /todos/:id/sprint` は課題とそのサブタスクを `sprintId` のスプリント（`0` はバックログ）に入れます。スプリントは課題と同じスペースで、完了していないものに限ります（違えば 400）。サブタスクのスプリントは親課題と同じになるので、サブタスクを指定すると 400 です。課題・スプリントが存在しなければ 404 です。
- サブタスクを作ると、親課題と同じスプリントに入ります。課題のスペースが変わると（エピックのスペース変更やスペースの削除を含む）、スプリントはスペースに属するのでバックログに戻ります。スペースを削除すると、そのスプリントも削除します。

### バージョン（リリース）

```
GET    /versions
POST   /spaces/:id/versions    {"name": "v1.0", "description": "最初のリリース", "startDate": "2026-10-01", "releaseDate": "2026-11-01"}
PUT    /versions/:id           {"name": "v1.0", "description": "最初のリリース", "startDate": "2026-10-01", "releaseDate": "2026-11-01", "status": "unreleased"}
POST   /versions/:id/release   {"releaseDate": "2026-11-01", "moveTo": 5}
DELETE /versions/:id
PUT    /todos/:id/fix-versions {"versionIds": [2, 5]}
```

Jira のリリース（修正バージョン）です。バージョンはスペースに属し、状態 `Status` は `unreleased`（未リリース）・`released`（リリース済み）・`archived`（アーカイブ）のいずれかです。課題は複数のバージョンに入れられます（課題の `FixVersionIDs`）。

```json
[
  { "ID": 2, "SpaceID": 1, "Name": "v1.0", "Description": "最初のリリース", "Status": "unreleased", "StartDate": "2026-10-01", "ReleaseDate": "2026-11-01", "CreatedAt": "2026-10-10T12:00:00Z" }
]
```

- `GET /versions` はすべてのバージョンを、スペース・作成順に返します。`StartDate`・`ReleaseDate` は開始日・リリース日（未設定は空文字）、`Description` は説明（なしは空文字）です。
- `POST /spaces/:id/versions` は未リリースのバージョンを作り、201 で `version` に入れて返します。スペース未割り当て（`:id` が 0）は 400、存在しないスペースは 404 です。名前は前後の空白を除いて 1〜100 文字で、同じスペースに同じ名前（大文字・小文字は区別しない）があれば 409 です。説明は 1000 文字以内、日付は `YYYY-MM-DD` でリリース日を開始日以降にします（どちらも空にできます）。
- `PUT /versions/:id` は名前・説明・日付・状態を変えます（`status` は必須）。未リリースに戻す・アーカイブする・アーカイブを解除するのにも使います。
- `POST /versions/:id/release` は未リリースのバージョンをリリース済みにします（未リリースでなければ 400）。`releaseDate` を省略すると、設定済みのリリース日か今日になります。`moveTo`（同じスペースの別の未リリースのバージョンの ID）を指定すると、完了していない課題（ステータスのカテゴリが `Done` でない）からこのバージョンを外し、`moveTo` のバージョンを付けます。`0` か省略なら、完了していない課題もこのバージョンのまま残します。`moveTo` が条件に合わなければ 400 です。
- `DELETE /versions/:id` はバージョンを削除し、課題からも外します。
- `PUT /todos/:id/fix-versions` は課題の修正バージョンを `versionIds` に置き換えます（`[]` ですべて外す）。バージョンは課題と同じスペースのものに限り（違えば 400）、アーカイブしたバージョンは新しくは付けられません（既に付いているものは残せます。400）。課題・バージョンが存在しなければ 404 です。
- 課題のスペースが変わると、バージョンはスペースに属するので修正バージョンから外れます。スペースを削除すると、そのバージョンも削除します。

### コンポーネント

```
GET    /components
POST   /spaces/:id/components  {"name": "フロントエンド", "description": "画面まわり", "defaultAssigneeId": 2}
PUT    /components/:id         {"name": "フロントエンド", "description": "画面まわり", "defaultAssigneeId": 0}
DELETE /components/:id
PUT    /todos/:id/components   {"componentIds": [4, 5]}
```

Jira のコンポーネント（スペース内の機能区分）です。コンポーネントはスペースに属し、既定の担当者を持てます。課題は複数のコンポーネントに入れられます（課題の `ComponentIDs`）。

```json
[
  {
    "ID": 4, "SpaceID": 1, "Name": "フロントエンド", "Description": "画面まわり",
    "DefaultAssigneeID": 2, "DefaultAssignee": { "ID": 2, "DisplayName": "佐藤 花子", "AvatarColor": "#00875A" },
    "CreatedAt": "2026-10-12T12:00:00Z"
  }
]
```

- `GET /components` はすべてのコンポーネントを、スペース・名前順に返します。`DefaultAssigneeID` は既定の担当者のユーザー ID（`0` はなし）、`DefaultAssignee` はその概要（なしは `null`）です。
- `POST /spaces/:id/components` はコンポーネントを作り、201 で `component` に入れて返します。`PUT /components/:id` は名前・説明・既定の担当者を変えます（`defaultAssigneeId` の `0` か省略でなし）。名前は前後の空白を除いて 1〜100 文字で、同じスペースに同じ名前（大文字・小文字は区別しない）があれば 409 です。説明は 1000 文字以内、存在しないユーザーは 400 です。スペース未割り当て（`:id` が 0）は 400、存在しないスペース・コンポーネントは 404 です。
- `DELETE /components/:id` はコンポーネントを削除し、課題からも外します。
- `PUT /todos/:id/components` は課題のコンポーネントを `componentIds` に置き換え（`[]` ですべて外す）、変更後の課題を `todo` に入れて返します。課題の担当者が未割り当てなら、新しく付けたコンポーネントのうち ID 順で最初に既定の担当者を持つものの、その人を担当者にします（変更履歴に残り、担当者はウォッチャーになります）。担当者がいる課題では担当者を変えません。コンポーネントは課題と同じスペースのものに限り（違えば 400）、課題・コンポーネントが存在しなければ 404 です。
- 課題のスペースが変わると、コンポーネントはスペースに属するので外れます。スペースを削除すると、そのコンポーネントも削除します。

### 保存済みフィルター

```
GET    /filters
POST   /filters                {"name": "自分のバグ", "jql": "assignee = currentUser() AND type = Bug", "favourite": true}
PUT    /filters/:id            {"name": "自分のバグ", "jql": "assignee = currentUser() AND type = Bug ORDER BY priority DESC"}
PUT    /filters/:id/favourite  {"favourite": false}
DELETE /filters/:id
POST   /todos/:id/view
```

```json
{ "ID": 1, "OwnerID": 1, "Name": "自分のバグ", "JQL": "assignee = currentUser() AND type = Bug", "Favourite": true, "CreatedAt": "2026-10-17T04:55:58Z", "UpdatedAt": "2026-10-17T04:55:58Z" }
```

- JQL（課題一覧の検索条件）に名前を付けて保存します（Jira のフィルター）。フィルターは作ったユーザーだけのもので、ほかのユーザーのフィルターは一覧に出ず、変更・削除は 404 です。
- `GET /filters` はログイン中のユーザーのフィルターを名前順に返します。`POST /filters` は 201 で `filter` に入れて返し、`PUT /filters/:id`（名前と JQL の上書き）・`PUT /filters/:id/favourite`（お気に入りのスターを付ける・外す）は 200 で `filter` に入れて返します。
- 名前は前後の空白を除いて 1〜100 文字で、同じユーザーに同じ名前（大文字・小文字は区別しない）があれば 409 です。JQL は 4000 文字以内で、空ならすべての課題です。JQL は構文だけを確かめ、誤りは `GET /issues/jql` と同じく 400 で `position` も返します（存在しない名前などは検索するときに誤りになります）。
- `POST /todos/:id/view` はログイン中のユーザーが課題を表示したことを記録します（課題ごとに最後の日時だけ）。JQL の `issueHistory()`・`ORDER BY lastViewed` に使います。課題がなければ 404 です。
- 既定のフィルター（自分の未完了課題・報告した課題・最近表示した課題・すべての課題）はフロントエンドが JQL として持ちます（`frontend/src/utils/filters.js`）。

### エピック

```
GET /epics
POST /epics        {"Title": "エピック名", "SpaceID": 0, "StartDate": "2026-10-01", "DueDate": "2026-10-31", "Color": "#6554C0"}
PUT /epics/:id     {"Title": "エピック名", "SpaceID": 2, "StartDate": "2026-10-05", "DueDate": "2026-11-10"}
DELETE /epics/:id
```

レスポンスは `{"ID": 1, "Title": "エピック名", "SpaceID": 0, "StartDate": "2026-10-01", "DueDate": "2026-10-31", "Color": "#6554C0", "CreatedAt": "2026-10-10T12:00:00Z", "UpdatedAt": "2026-10-10T12:00:00Z"}` の配列です。作成日時・最終更新日時はタスクと同じ形式で、`PUT` で `UpdatedAt` が変わります。

`StartDate`（開始日）・`DueDate`（期限日）は `YYYY-MM-DD`、`Color`（タイムラインのバーの色）は `#RRGGBB`（大文字にそろえて保存）で、空文字は未設定（色は既定の色）です。作成で省略すると未設定、`PUT` で省略（または `null`）すると今の値のまま、空文字を送ると未設定に戻します。形式が違う、または期限日が開始日より前になる場合は 400 を返します。`PUT` でスペースを変えると、子タスクも同じスペースへ移動します。削除すると子タスクはエピック未割り当てになります。

### スペース

エピックの上位の単位です（プロダクトごとのTODOリストなど）。

```
GET /spaces
GET /spaces/key-suggestion?title=スペース名
POST /spaces       {"Title": "スペース名", "Key": "KAN"}
PUT /spaces/:id    {"Title": "スペース名", "Key": "KAN"}
DELETE /spaces/:id
```

レスポンスは `{"ID": 1, "Title": "スペース名", "Key": "KAN"}` の配列です。

`Key` は課題キーの接頭辞で、全スペースで一意です。英大文字で始まる英大文字・数字の 2〜10 文字で、`TASK` は使えません。小文字で送ると大文字にして保存します。省略すると名前から作ります。形式が不正なら 400、他のスペースと重複すると 409 を返します。

`PUT` は名前とキーを変えます（どちらも必須）。キーを変えると、そのスペースのタスクの課題キーもすべて変わります（番号は変わりません。古いキーは残りません）。キーの形式と重複は `POST` と同じく 400・409 で返し、存在しないスペースは 404 です。

`GET /spaces/key-suggestion` は、名前から作ったキーの候補を `{"key": "KAN"}` で返します。英単語が 2 つ以上ならその頭文字、1 つなら先頭の 3 文字を使い（例: `Kanban` → `KAN`、`Mobile App` → `MA`）、英字がなければ `SP` にします。既存のキーと重なる場合は `KAN2` のように数字を付けます。

削除すると、属していたエピックとタスクはスペース未割り当て（`SpaceID` 0）になります。エピックとタスクの紐付けは残ります。

## アーキテクチャ

本プロジェクトはクリーンアーキテクチャを採用しており、以下のレイヤーで構成されています：

- **Domain Layer**: ビジネスロジックと関連エンティティ
- **Service Layer**: ビジネスロジックの実装
- **Repository Layer**: データアクセス層
- **Handler Layer**: HTTPリクエスト/レスポンスの処理

## データ永続化

データは MySQL の `spaces`・`epics`・`todos`・`labels`・`todo_labels`・`issue_links`・`comments`・`issue_history`・`attachments`・`worklogs`・`statuses`・`users`・`sessions`・`watchers`・`notifications`・`sprints`・`sprint_issues`・`versions`・`todo_versions`・`components`・`todo_components`・`saved_filters`・`issue_views` テーブルに保存します（添付ファイルの本体は `ATTACHMENTS_DIR` のディレクトリ）。テーブルは起動時に `internal/database` の `Migrate` が作成します（`CREATE TABLE IF NOT EXISTS`）。

- `todos.position` がタスクの表示順です。
- `EpicID`・`SpaceID` の 0 は「未割り当て」を表すため、外部キー制約は付けていません。
- `Deadline` は `DATE` 型で、期限なし（空文字）は `NULL` として保存します。
- `spaces.space_key` がスペースのキー、`todos.number` がスペース内の番号です。`issue_counters` にスペース（`space_id`。0 は未割り当て）ごとの採番済みの最大の番号を持ち、削除した番号を再利用しないようにしています。
- 課題キーの導入前のデータベースでは、起動時に `Migrate` がキーのないスペースに名前から作ったキーを ID 順に付け、番号のない（`number` が 0 の）タスクにスペースごとに ID 順で番号を振ります。CSV から取り込んだデータも、`import-csv` が最後に `Migrate` を実行して同じように番号を付けます。
- サブタスクの親課題は `todos.parent_id`（0 はサブタスクではない）です。
- ワークフローのステータスは `statuses` に、スペース `space_id`（0 は未割り当て）・名前 `name`・カテゴリ `category`・並び順 `position`・導入前のステータス `legacy_key`（既定の 4 ステータスだけ）・列の最大件数 `wip_limit`（`NULL` は制限なし）で保存し、課題は `todos.status_id` で指します。`todos.status` には互換用の値を書き込みます。ワークフローの導入前のデータベースでは、起動時に `Migrate` が各スペースと未割り当てに既定の 4 ステータス（オープン=To Do、進行中・待ち状況=In Progress、クローズ=Done）を作り、課題を同じスペースの同じステータスに対応付け、`done` をカテゴリに合わせます（何度実行しても同じ結果になります）。
- 初期見積もり・残り時間は `todos.original_estimate`・`todos.remaining_estimate`（分、`NULL` は未設定）、作業ログは `worklogs` に課題 `todo_id`・作業時間 `minutes`（分）・開始日時 `started_at`・メモ `note`・記録日時 `created_at`・編集日時 `updated_at` で保存します。
- 添付ファイルは `attachments` に、課題 `todo_id`・ファイル名 `file_name`・種類 `content_type`・バイト数 `size`・保存名 `storage_name`・添付日時 `created_at` で保存します。本体は `ATTACHMENTS_DIR` に `storage_name`（推測できないランダムな名前）で置きます。読み書きは `internal/repository` の添付ファイルの Repository だけが行います。
- 変更履歴は `issue_history` に、課題 `todo_id`・項目名 `field`・変更前 `old_value`・変更後 `new_value`・項目が指すものの変更前・変更後の ID `old_ref_id`・`new_ref_id`（スプリントの変更だけ。それ以外は `NULL`）・変更者 `user_id`（0 はなし）・記録日時 `created_at` で保存します。`old_ref_id`・`new_ref_id` と `sprints.started_at` がない既存のデータベースでは、起動時に `Migrate` が `ALTER TABLE` で追加します（既存の行は `NULL`）。記録はサービス層（`internal/service` の `historyRecorder`）で行います。
- コメントは `comments` に、課題 `todo_id`・本文 `body`・投稿者 `author_id`（0 はなし）・投稿日時 `created_at`・編集日時 `updated_at`（`NULL` は未編集）で保存します。
- ウォッチャーは `watchers`（`todo_id`・`user_id`・`created_at`）、通知は `notifications` に保存します。通知の列は、受け取るユーザー `user_id`・課題 `todo_id`・変更したユーザー `actor_id`・種類 `kind`・内容 `detail`・既読にした日時 `read_at`（`NULL` は未読）・`created_at` です。通知は、サービス層の `WithNotifications` で包んだ変更履歴の Repository が、記録に合わせて作ります。
- 課題の担当者・報告者は `todos.assignee_id`・`todos.reporter_id`（`users.id`。0 はなし）です。これらの列と `comments.author_id`・`issue_history.user_id` がない既存のデータベースでは、起動時に `Migrate` が `ALTER TABLE` で追加します（既存の行は 0 = なし）。
- 課題リンクは `issue_links` に、リンク元 `source_id`・リンク先 `target_id`・種類 `link_type`（`Blocks`・`Relates`・`Duplicates`・`Clones`）で保存します。`source_id` の課題が `target_id` の課題を `link_type` する（例: source blocks target）という向きです。
- ストーリーポイントは `todos.story_points`（`DECIMAL(8,2)`、`NULL` は未設定）です。
- スプリントは `sprints` に、スペース `space_id`・名前 `name`・ゴール `goal`・状態 `state`（`future`・`active`・`closed`）・期間 `start_date`・`end_date`（`DATE`、`NULL` は未設定）・開始した日時 `started_at`（`NULL` は未開始か記録前）・完了した日時 `completed_at`（`NULL` は未完了）・作成日時 `created_at` で保存し、課題は `todos.sprint_id`（0 はバックログ）で指します。完了したときの記録は `sprint_issues` に、スプリント `sprint_id`・課題 `todo_id`・その時点のストーリーポイント `story_points`・完了していたか `completed` で保存します。`todos.sprint_id` がない既存のデータベースでは、起動時に `Migrate` が `ALTER TABLE` で追加します（既存の課題は 0 = バックログ）。
- バージョンは `versions` に、スペース `space_id`・名前 `name`・説明 `description`・状態 `status`（`unreleased`・`released`・`archived`）・開始日 `start_date`・リリース日 `release_date`（`DATE`、`NULL` は未設定）・作成日時 `created_at` で保存します。課題の修正バージョンは多対多を `todo_versions`（`todo_id`・`version_id`）で表します。課題やバージョンを削除すると `todo_versions` の行も削除します。
- コンポーネントは `components` に、スペース `space_id`・名前 `name`・説明 `description`・既定の担当者 `default_assignee_id`（`users.id`。0 はなし）・作成日時 `created_at` で保存します。課題のコンポーネントは多対多を `todo_components`（`todo_id`・`component_id`）で表します。課題やコンポーネントを削除すると `todo_components` の行も削除します。
- 保存済みフィルターは `saved_filters` に、作ったユーザー `owner_id`（`users.id`）・名前 `name`・JQL `jql`・お気に入り `favourite`・作成日時 `created_at`・更新日時 `updated_at` で保存します。課題を表示した記録は `issue_views` に、ユーザー `user_id`・課題 `todo_id`・最後に表示した日時 `viewed_at`（`DATETIME(3)`、UTC）で保存します。課題を削除すると `issue_views` の行も削除します。
- ラベルは `labels` に持ちます。`name` の照合順序は `utf8mb4_bin` です（既定の照合順序では、ひらがなとカタカナや全角と半角が同じとみなされるため）。大文字・小文字の違いだけのラベルは、保存時に既存のラベルにまとめます。タスクとの多対多を `todo_labels`（`todo_id`・`label_id`）で表します。タスクを削除すると `todo_labels` の行も削除します（ラベル自体は残ります）。
- ユーザーは `users` に、表示名 `display_name`・メールアドレス `email`（小文字に正規化。一意）・パスワードのハッシュ `password_hash`（bcrypt）・アバターの色 `avatar_color`・作成日時 `created_at` で保存します。ログインのセッションは `sessions` に、トークンの SHA-256 `token_hash`・ユーザー `user_id`・作成日時 `created_at`・有効期限 `expires_at` で保存します。期限切れのセッションはログインのときに削除します。
- `epics` の `start_date`・`due_date`（`DATE`、NULL は未設定）と `color`（`VARCHAR(7)`、空文字は既定の色）は、エピックのタイムラインの期間と色です。これらの列がない既存のデータベースでは、起動時に `Migrate` が `ALTER TABLE` で追加します（既存のエピックは未設定になります）。
- `epics`・`todos` の `created_at`・`updated_at` は作成日時・最終更新日時（`DATETIME`、UTC、秒単位）です。これらの列がない既存のデータベースでは、起動時に `Migrate` が `ALTER TABLE` で追加します。その時点で既にあった行の作成日時・更新日時は、列を追加した日時になります。

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
