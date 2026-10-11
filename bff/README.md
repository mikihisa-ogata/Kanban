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
  "SpaceID": 0
}
```

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
  "SpaceID": 0
}
```

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
  { "ID": 1, "TodoID": 3, "Field": "created", "OldValue": null, "NewValue": null, "CreatedAt": "2026-10-11T01:00:00Z" },
  { "ID": 2, "TodoID": 3, "Field": "status", "OldValue": "オープン", "NewValue": "進行中", "CreatedAt": "2026-10-11T01:05:00Z" },
  { "ID": 3, "TodoID": 3, "Field": "storyPoints", "OldValue": "3", "NewValue": "5", "CreatedAt": "2026-10-11T01:05:00Z" }
]
```

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

`PUT /todos/:id` のほか、エピックのスペース変更・エピックやスペースの削除・親課題の移動で課題が変わった場合も記録します。ステータスとストーリーポイントは値と日時を残すので、ある時点のステータスとポイントを後から再計算できます。

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
    "CreatedAt": "2026-10-11T01:00:00Z",
    "UpdatedAt": "2026-10-11T01:05:00Z"
  }
]
```

- `Body` は Markdown の本文です。空や空白だけの本文、10000 文字を超える本文は 400 を返します。
- `CreatedAt` は投稿日時、`UpdatedAt` は最後に編集した日時です（UTC の RFC 3339 形式）。編集していないコメントの `UpdatedAt` は `null` です。
- `POST` は 201 で作成したコメントを、`PUT` は 200 で編集後のコメントを `comment` に入れて返します。
- 課題やコメントが存在しなければ 404 を返します。
- 投稿者はまだ持ちません（ユーザー機能の追加後に付けます）。

### ワークフローのステータス

```
GET /statuses
```

すべてのワークフローのステータス（ボードの列）を、スペース・並び順の順に返します。`SpaceID` が `0` のものは、スペース未割り当ての課題のワークフローです。スペースを作ると、そのスペースに既定の 4 ステータスを作ります。スペースを削除すると、そのスペースのステータスも削除します（課題は未割り当てのワークフローへ移ります）。

```json
[
  { "ID": 1, "SpaceID": 0, "Name": "オープン", "Category": "ToDo", "Position": 1 },
  { "ID": 2, "SpaceID": 0, "Name": "進行中", "Category": "InProgress", "Position": 2 },
  { "ID": 3, "SpaceID": 0, "Name": "待ち状況", "Category": "InProgress", "Position": 3 },
  { "ID": 4, "SpaceID": 0, "Name": "クローズ", "Category": "Done", "Position": 4 }
]
```

### エピック

```
GET /epics
POST /epics        {"Title": "エピック名", "SpaceID": 0}
PUT /epics/:id     {"Title": "エピック名", "SpaceID": 2}
DELETE /epics/:id
```

レスポンスは `{"ID": 1, "Title": "エピック名", "SpaceID": 0, "CreatedAt": "2026-10-10T12:00:00Z", "UpdatedAt": "2026-10-10T12:00:00Z"}` の配列です。作成日時・最終更新日時はタスクと同じ形式で、`PUT` で `UpdatedAt` が変わります。`PUT` でスペースを変えると、子タスクも同じスペースへ移動します。削除すると子タスクはエピック未割り当てになります。

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

データは MySQL の `spaces`・`epics`・`todos`・`labels`・`todo_labels`・`issue_links`・`comments`・`issue_history`・`attachments`・`worklogs`・`statuses` テーブルに保存します（添付ファイルの本体は `ATTACHMENTS_DIR` のディレクトリ）。テーブルは起動時に `internal/database` の `Migrate` が作成します（`CREATE TABLE IF NOT EXISTS`）。

- `todos.position` がタスクの表示順です。
- `EpicID`・`SpaceID` の 0 は「未割り当て」を表すため、外部キー制約は付けていません。
- `Deadline` は `DATE` 型で、期限なし（空文字）は `NULL` として保存します。
- `spaces.space_key` がスペースのキー、`todos.number` がスペース内の番号です。`issue_counters` にスペース（`space_id`。0 は未割り当て）ごとの採番済みの最大の番号を持ち、削除した番号を再利用しないようにしています。
- 課題キーの導入前のデータベースでは、起動時に `Migrate` がキーのないスペースに名前から作ったキーを ID 順に付け、番号のない（`number` が 0 の）タスクにスペースごとに ID 順で番号を振ります。CSV から取り込んだデータも、`import-csv` が最後に `Migrate` を実行して同じように番号を付けます。
- サブタスクの親課題は `todos.parent_id`（0 はサブタスクではない）です。
- ワークフローのステータスは `statuses` に、スペース `space_id`（0 は未割り当て）・名前 `name`・カテゴリ `category`・並び順 `position`・導入前のステータス `legacy_key`（既定の 4 ステータスだけ）で保存し、課題は `todos.status_id` で指します。`todos.status` には互換用の値を書き込みます。ワークフローの導入前のデータベースでは、起動時に `Migrate` が各スペースと未割り当てに既定の 4 ステータス（オープン=To Do、進行中・待ち状況=In Progress、クローズ=Done）を作り、課題を同じスペースの同じステータスに対応付け、`done` をカテゴリに合わせます（何度実行しても同じ結果になります）。
- 初期見積もり・残り時間は `todos.original_estimate`・`todos.remaining_estimate`（分、`NULL` は未設定）、作業ログは `worklogs` に課題 `todo_id`・作業時間 `minutes`（分）・開始日時 `started_at`・メモ `note`・記録日時 `created_at`・編集日時 `updated_at` で保存します。
- 添付ファイルは `attachments` に、課題 `todo_id`・ファイル名 `file_name`・種類 `content_type`・バイト数 `size`・保存名 `storage_name`・添付日時 `created_at` で保存します。本体は `ATTACHMENTS_DIR` に `storage_name`（推測できないランダムな名前）で置きます。読み書きは `internal/repository` の添付ファイルの Repository だけが行います。
- 変更履歴は `issue_history` に、課題 `todo_id`・項目名 `field`・変更前 `old_value`・変更後 `new_value`・記録日時 `created_at` で保存します。記録はサービス層（`internal/service` の `historyRecorder`）で行います。
- コメントは `comments` に、課題 `todo_id`・本文 `body`・投稿日時 `created_at`・編集日時 `updated_at`（`NULL` は未編集）で保存します。
- 課題リンクは `issue_links` に、リンク元 `source_id`・リンク先 `target_id`・種類 `link_type`（`Blocks`・`Relates`・`Duplicates`・`Clones`）で保存します。`source_id` の課題が `target_id` の課題を `link_type` する（例: source blocks target）という向きです。
- ストーリーポイントは `todos.story_points`（`DECIMAL(8,2)`、`NULL` は未設定）です。
- ラベルは `labels` に持ちます。`name` の照合順序は `utf8mb4_bin` です（既定の照合順序では、ひらがなとカタカナや全角と半角が同じとみなされるため）。大文字・小文字の違いだけのラベルは、保存時に既存のラベルにまとめます。タスクとの多対多を `todo_labels`（`todo_id`・`label_id`）で表します。タスクを削除すると `todo_labels` の行も削除します（ラベル自体は残ります）。
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
