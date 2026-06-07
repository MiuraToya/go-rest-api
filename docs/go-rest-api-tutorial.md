# Go で簡易 REST API を TDD で作りながら学ぶチュートリアル

この教材は、Python を 5 年使ってきた人が Go のバックエンド開発に入ることを前提にしています。
文法だけで終わらず、実務でよく見る構成、REST API の作り方、SQLite を使った IT、そして TDD の進め方まで一通り体験する構成です。

このリポジトリの役割:

- `docs/`: 教材
- リポジトリルート: あなたの実装場所
- `reference/complete-app/`: 完成版

以降、「実装する」と書いてある箇所は、リポジトリルート配下を指します。

## 1. この教材で作るもの

作るのは `tasks` を管理する REST API です。

- `GET /healthz`
- `GET /tasks`
- `GET /tasks/{id}`
- `POST /tasks`
- `PATCH /tasks/{id}`
- `DELETE /tasks/{id}`

学べる要素:

- `package`, `import`
- `struct`, method, `interface`
- `if`, `switch`, `for`, `defer`
- pointer を使った「未指定」と「空文字」の区別
- `context.Context`
- 標準ライブラリの HTTP ルーティング
- `database/sql` と SQLite
- error wrapping
- Unit Test / Integration Test
- goroutine と graceful shutdown
- TDD の `Red -> Green -> Refactor`

## 2. このリポジトリの使い方

学習中は、完成版を直接いじらずにルートで自分のコードを書きます。

```text
docs/                   <- 教材
reference/complete-app/ <- 完成版
cmd/                    <- あなたが実装する場所
internal/               <- あなたが実装する場所
integration/            <- あなたが実装する場所
```

困ったときだけ `reference/complete-app/` を見てください。
最初から答えを見る前提にすると、文法と設計が頭に残りにくいです。

## 3. セットアップ

### 3-1. `mise` を入れる

macOS なら Homebrew で入れるのが手軽です。

```bash
brew install mise
```

確認:

```bash
mise --version
```

このリポジトリには [mise.toml](../mise.toml) があり、Go のバージョンは `1.26.4` に固定しています。

ここで `mise` を使う理由は、学習中の Go バージョン差異を消したいからです。
Go は標準ライブラリの振る舞いや `go` コマンドの挙動がバージョンで少しずつ変わります。
教材と手元のバージョンがずれると、「自分のコードが悪いのか、バージョン差分なのか」が切り分けにくくなります。

### 3-2. Go をインストールする

```bash
mise install
```

このコマンドで Go が入る理由は、ルートの [mise.toml](../mise.toml) に次が書いてあるからです。

```toml
[tools]
go = "1.26.4"
```

`mise install` は「今いるディレクトリの `mise.toml` を読んで、そこに書かれたツールを入れる」コマンドです。
つまり、ここでは明示的に `go` を引数で渡していなくても問題ありません。

これをやらないと困ること:

- `go` コマンド自体が使えない
- 教材どおりに進めても、ローカルの Go バージョン差分で挙動がずれる
- `go test` の失敗原因がコードではなく環境かもしれなくなる

シェル統合がまだなら、通常は `~/.zshrc` に次を入れます。

```bash
eval "$(mise activate zsh)"
```

### 3-3. Go module を初期化する

ルートで次を実行します。

```bash
go mod init github.com/yourname/go-rest-api
```

練習用なので module 名は厳密でなくて構いません。
ただし、GitHub に置く前提なら `github.com/<yourname>/<repo>` 形式にしておくのが一般的です。

このコマンドの目的は 3 つあります。

- このルートを 1 つの Go プロジェクトとして宣言する
- `go.mod` を作って依存管理の起点にする
- 自分の package を import するときの基準名を決める

たとえば module 名を `github.com/yourname/go-rest-api` にすると、ルート配下の package はこう import されます。

```go
import "github.com/yourname/go-rest-api/internal/task"
```

これをやらないと困ること:

- `go test ./...` を「このプロジェクト全体」に対して素直に回しづらい
- `internal/task` などを import するときの基準パスが決まらない
- `go mod tidy` で何を依存として管理するか決められない

### 3-4. ディレクトリを確認する

すでに空ディレクトリは置いてあります。最終的にこうなれば OK です。

```text
cmd/api/main.go
internal/config/config.go
internal/api/
internal/task/
internal/store/sqlite/
integration/
```

### 3-5. 依存関係を解決する

最初は依存がなくても構いませんが、SQLite ドライバを入れたあとや import を増やしたあとに実行します。

```bash
go mod tidy
```

`go.mod` は Python の `pyproject.toml` や `requirements.txt` に近い役割です。
`go mod tidy` は「使っている import を見て、必要な依存だけを `go.mod` / `go.sum` に反映する」コマンドです。

これをやる理由:

- 依存の入れ忘れを防ぐ
- 逆に不要になった依存を消せる
- 他の人が clone したときに、同じ依存セットを復元しやすくなる

Go では package manager に個別 install するより、「コードを書いてから `go mod tidy` で整える」流れがかなり一般的です。

## 4. 先に押さえる Go 文法

### 4-1. `package` と `import`

Go ファイルは package 宣言から始まります。

```go
package task
```

- package はコードのまとまり
- Go では directory と package の対応がかなり明確
- import は他 package の読み込み

### 4-2. 変数宣言と型推論

```go
var count int
name := "gopher"
limit := 20
```

- `var` は明示的な宣言
- `:=` は関数内で使う短縮宣言
- Go は型推論するが、動的型付けではない

### 4-3. `struct`

```go
type Task struct {
	ID          int64
	Title       string
	Description string
	Status      Status
}
```

- Go には class がない
- データのまとまりは `struct`
- 大文字始まりの field は package 外から参照可能

### 4-4. method

```go
func (t Task) Validate() error {
	if t.Title == "" {
		return errors.New("title is required")
	}
	return nil
}
```

- `(t Task)` は receiver
- Python の `self` に少し近い
- 値 receiver と pointer receiver を使い分ける

完成版では [internal/task/task.go](../reference/complete-app/internal/task/task.go) で `Validate()` を定義しています。

### 4-5. `interface`

```go
type Repository interface {
	List(ctx context.Context, filter Filter) ([]Task, error)
	GetByID(ctx context.Context, id int64) (Task, error)
}
```

- 明示的な `implements` は不要
- 必要な method を持っていれば満たす
- 小さく保つのが定石

### 4-6. `if`, `switch`, `for`

```go
if err != nil {
	return err
}

switch status {
case StatusTodo:
	...
default:
	...
}

for rows.Next() {
	...
}
```

- 条件式に丸括弧は不要
- `if err != nil` は最頻出
- Go には `while` がなく、繰り返しは `for` に統一

### 4-7. slice と map

```go
var tasks []Task
tasks = append(tasks, task)

codes := map[string]int{"ok": 200}
```

- `[]Task` は slice
- `append` で追加
- map は Python の dict に近い

### 4-8. pointer

今回の教材では `PATCH` の未指定表現に使います。

```go
type UpdateInput struct {
	Title  *string `json:"title"`
	Status *Status `json:"status"`
}
```

- `nil` なら未指定
- 値があれば更新意思あり

### 4-9. `error`

Go は例外ではなく `error` を返します。

```go
id, err := strconv.ParseInt(rawID, 10, 64)
if err != nil {
	return 0, fmt.Errorf("parse id: %w", err)
}
```

- 複数戻り値を多用する
- `%w` で元の error を包める

### 4-10. `defer`

```go
rows, err := db.QueryContext(ctx, query)
if err != nil {
	return err
}
defer rows.Close()
```

Python の `with` 文に近い場面があります。

### 4-11. `context.Context`

```go
func (m *Manager) List(ctx context.Context, filter Filter) ([]Task, error)
```

- ほぼ慣習として第一引数
- HTTP から DB まで渡す
- timeout / cancel を伝播する

### 4-12. goroutine

```go
go func() {
	serverErrors <- server.ListenAndServe()
}()
```

- `go` を付けると並行実行
- 最初は乱用しない方がよい

### 4-13. zero value

Go の変数にはデフォルト値があります。

- `int` は `0`
- `string` は `""`
- `bool` は `false`
- pointer は `nil`

## 5. この教材の進め方は TDD

このチュートリアルでは、常に次の順で進めます。

1. `Red`
2. `Green`
3. `Refactor`

### 5-1. `Red`

- まずテストを書く
- まだ実装していないので失敗させる
- compile error でも最初の失敗としては許容

なぜ先に失敗させるのか:

- 何を固定したいかを先に言語化できる
- 実装に引きずられず、外から見た振る舞いを先に決められる
- 後から「何となく通ったコード」ではなく「このテストを通すためのコード」になる

### 5-2. `Green`

- テストを通す最小限の実装を書く
- 先回りして機能を盛らない

なぜ最小限にするのか:

- 最初から設計を広げると、何が本当に必要か見えにくくなる
- Go は書けることが多いが、最初の一歩を小さくした方が責務分離を学びやすい
- 不要な抽象化を減らせる

### 5-3. `Refactor`

- 重複を減らす
- 命名を整える
- 責務の位置を見直す
- テストを壊さずに構造だけよくする

ここで重要なのは、「機能追加」と「設計改善」を分けることです。
Python だと実装しながら同時に整えたくなりがちですが、TDD では Green と Refactor を分けた方が判断がぶれません。

### 5-4. この教材のテスト戦略

テスト戦略はテストピラミッドです。

- Unit Test
  - service と handler を速く回す
- Integration Test
  - SQLite をつないで HTTP から DB まで確認する

理由は、学習対象が Go 文法と設計の基礎だからです。
まず UT で責務ごとのフィードバックを早く回し、結合不整合だけ IT で押さえる形が最も学びやすいです。

## 6. TDD で実装する順番

ここからは、実際にどう進めるかをイテレーション単位で書きます。
基本は 1 イテレーションごとに `go test` を回してください。

### 6-1. イテレーション 0: 土台を作る

最初にやること:

- `go mod init ...`
- `go mod tidy`
- 必要なら空ファイルを作る

この段階ではまだテストを書かなくて構いません。
ここは TDD の前提準備です。

理由:

- `go mod init` 前は、そもそも Go プロジェクトとしての単位が未確定
- package を切る場所と import path が未確定な状態でテストを書くと、失敗理由が環境と設計で混ざる
- まずは「テストを実行できる土台」だけ先に作る

### 6-2. イテレーション 1: `Task` の作成ルールを UT で決める

最初に書くテスト:

- `internal/task/service_test.go`
- 振る舞い:
  - 有効な入力なら task を作成できる
  - title の前後空白が除去される
  - status が `todo` で初期化される

まず書くべきテスト名の例:

```go
func TestCreateTask_ValidInput_ReturnsCreatedTask(t *testing.T)
```

この時点では compile error でも大丈夫です。
`Task`, `CreateInput`, `Manager`, `Repository` がまだ無いからです。

最初に service から始める理由:

- domain ルールが一番変わりやすく、一番先に固定したいから
- HTTP や DB をまだ考えなくてよいので、Go の基本文法に集中できるから
- Python 経験者でも「入力を受けて値を返す」層は理解しやすいから

`Green` で実装する場所:

- `internal/task/task.go`
- `internal/task/service.go`

ここで学ぶこと:

- `struct`
- `interface`
- method
- `time.Time`
- `context.Context`

この段階では repository は stub で十分です。
実 DB をつなぐのはまだ早く、今は「Task をどう作るべきか」という業務ルールだけを固定するのが目的です。

完成版の参照:

- [internal/task/task.go](../reference/complete-app/internal/task/task.go)
- [internal/task/service.go](../reference/complete-app/internal/task/service.go)
- [internal/task/service_test.go](../reference/complete-app/internal/task/service_test.go)

### 6-3. イテレーション 2: validation を UT で足す

次に書くテスト:

- title が空なら `ErrInvalidTask`
- 不正な status なら `ErrInvalidTask`
- limit 未指定なら既定値 20

おすすめのテスト名:

```go
func TestUpdateTask_InvalidStatus_ReturnsValidationError(t *testing.T)
func TestListTasks_EmptyLimit_UsesDefaultLimit(t *testing.T)
```

`Green` でやること:

- `ErrInvalidTask`
- `Status.Validate()`
- `Filter.Normalize()`
- `validateID()`

`Refactor` の観点:

- validation を handler ではなく service / domain 側に寄せる
- 既定値ロジックを一箇所にまとめる

この順番にする理由:

- 作成成功系だけ先に通してから、失敗系を足した方が流れを追いやすい
- いきなり正常系と異常系を全部入れると、Go の型と error 処理に慣れる前に情報量が増えすぎる

### 6-4. イテレーション 3: handler の JSON 変換を UT で決める

次は HTTP 層に入ります。

最初に書くテスト:

- `internal/api/handler_test.go`
- 振る舞い:
  - 想定外フィールドを含む JSON は 400
  - 正しい query parameter なら JSON レスポンスを返す

テスト名の例:

```go
func TestCreateTask_UnknownField_ReturnsBadRequest(t *testing.T)
func TestListTasks_ValidQuery_ReturnsTasks(t *testing.T)
```

`Green` で実装する場所:

- `internal/api/handler.go`

ここで初めて HTTP 層に行く理由:

- service の契約が先に固まっていれば、handler は「HTTP と service の変換」に専念できる
- 逆に service が曖昧なまま handler を書くと、どこに validation を置くかがぶれやすい

ここで学ぶこと:

- `httptest`
- `json.Decoder`
- `DisallowUnknownFields`
- `http.ResponseWriter`

完成版の参照:

- [internal/api/handler.go](../reference/complete-app/internal/api/handler.go)
- [internal/api/handler_test.go](../reference/complete-app/internal/api/handler_test.go)

### 6-5. イテレーション 4: router と middleware を最小限で通す

ここでは大きなテストはまだ増やさず、必要最小限で実装します。

`Green` で実装する場所:

- `internal/api/router.go`
- `internal/api/middleware.go`

最初は次だけで十分です。

- `GET /healthz`
- `GET /tasks`
- `POST /tasks`

その後で:

- `GET /tasks/{id}`
- `PATCH /tasks/{id}`
- `DELETE /tasks/{id}`

を足してください。

実務でも最初から全 endpoint を広げるより、縦に 1 本通してから広げる方が安全です。

ここで大きいテストを増やしすぎない理由:

- router は配線の色が強く、service ほどロジックの中心ではない
- 先に handler の責務を固めてから router を載せた方が、失敗原因を追いやすい

完成版の参照:

- [internal/api/router.go](../reference/complete-app/internal/api/router.go)
- [internal/api/middleware.go](../reference/complete-app/internal/api/middleware.go)

### 6-6. イテレーション 5: SQLite をつなぐ前に IT を書く

ここで初めて Integration Test を書きます。

最初に書くテスト:

- `integration/api_integration_test.go`
- 振る舞い:
  - `POST /tasks` で作成できる
  - `GET /tasks` で取得できる

最初のテスト名の例:

```go
func TestTaskLifecycle_SQLiteBackedAPI_WorksEndToEnd(t *testing.T)
```

最初は当然失敗します。
repository も DB 初期化もまだ無いからです。

ここで初めて IT に行く理由:

- 内側の振る舞いが UT で固まったあとに、結合点の不整合だけを見たいから
- 最初から DB をつなぐと、「Go 文法の理解」と「DB 連携の問題」が混ざって学習効率が落ちるから

`Green` で実装する場所:

- `internal/store/sqlite/task_repository.go`

ここで学ぶこと:

- `database/sql`
- `sql.DB`
- `QueryContext`, `ExecContext`
- `sql.ErrNoRows`
- migration

完成版の参照:

- [internal/store/sqlite/task_repository.go](../reference/complete-app/internal/store/sqlite/task_repository.go)
- [integration/api_integration_test.go](../reference/complete-app/integration/api_integration_test.go)

### 6-7. イテレーション 6: 更新と削除を IT で広げる

次に既存の IT を広げます。

追加する振る舞い:

- `PATCH /tasks/{id}` で status を更新できる
- `DELETE /tasks/{id}` 後に 404 になる

この段階で:

- handler
- service
- repository

の 3 層すべてに変更が入ります。

これは実務でもよくある「仕様追加時の縦切り変更」です。
1 本の failing integration test から入ると、全体の接続ミスを見つけやすいです。

この段階で学ぶこと:

- pointer を使った partial update
- not found の扱い
- service / repository / handler をまたぐ変更

### 6-8. イテレーション 7: `main.go` と設定を仕上げる

最後にアプリ起動まで通します。

実装する場所:

- `internal/config/config.go`
- `cmd/api/main.go`

やること:

- env から設定を読む
- DB を開く
- migration を流す
- router を組み立てる
- graceful shutdown を入れる

ここは厳密な TDD がやややりづらい箇所です。
そのため、この教材では「内側は TDD、配線の最後だけ薄く実装」で進めます。
実務でもこの割り切りは普通にあります。

理由:

- `main.go` はロジックというより wiring の責務が中心
- テスト可能性より、読みやすく薄く保つことの方が重要
- ここまでで内側の振る舞いが固まっていれば、最後の配線は比較的安全に進められる

完成版の参照:

- [internal/config/config.go](../reference/complete-app/internal/config/config.go)
- [cmd/api/main.go](../reference/complete-app/cmd/api/main.go)

## 7. まずどこに何を書くか

### 7-1. `internal/task/`

ここには domain model と service を置きます。

- `Task`
- `Status`
- `Filter`
- `CreateInput`
- `UpdateInput`
- `Repository interface`
- `Service`

### 7-2. `internal/api/`

ここには HTTP 層を置きます。

- handler
- router
- middleware

### 7-3. `internal/store/sqlite/`

ここには SQLite の repository 実装を置きます。

- `Open`
- `Migrate`
- `Repository`

### 7-4. `cmd/api/`

ここには起動処理を書きます。

- config 読み込み
- DB 初期化
- service と router の配線
- HTTP server 起動
- graceful shutdown

### 7-5. `integration/`

ここには SQLite をつないだ IT を置きます。

## 8. 実行コマンド

イテレーションごとに、まず変更範囲だけ回します。

これは実務でも重要です。
毎回いきなり全部回すより、まず変更箇所に近いテストを回した方がフィードバックが速く、失敗原因も追いやすいです。

### 8-1. service を触ったとき

```bash
go test ./internal/task
```

### 8-2. handler を触ったとき

```bash
go test ./internal/api
```

### 8-3. 全 UT を回すとき

```bash
go test ./...
```

### 8-4. IT を回すとき

```bash
go test -tags=integration ./...
```

完成版をすぐ動かしたい場合:

```bash
cd reference/complete-app
go test ./...
go test -tags=integration ./...
make run
```

## 9. 実務っぽく進めるためのコツ

- 1 イテレーション 1 振る舞いに絞る
- まず UT で内側を固め、結合点だけ IT で確認する
- コンパイルを通すことより、「どの振る舞いを固定したいか」を先に決める
- 実装はテストを通す最小限にする
- `Refactor` で初めて構造改善に入る

特に Python 経験者は、最初から完成形を頭の中で大きく作りすぎることがあります。
Go では小さく通してから広げる方がかなりうまくいきます。

## 10. Python 経験者向けの見方

### 10-1. `class` より `struct`

Python では class ベースで組み立てがちですが、Go ではまず `struct` と関数で十分です。
継承はなく、interface で振る舞いの契約を表します。

### 10-2. 例外ではなく `error`

Go は基本的に戻り値で `error` を返します。
冗長に見えても、失敗地点がかなり明示されます。

### 10-3. `None` の代わりに pointer

`PATCH` のように「未指定」と「空文字」を分けたいとき、Go では pointer がよく使われます。

### 10-4. goroutine は乱用しない

Go は並行処理が書きやすいですが、最初の CRUD 学習では増やしすぎない方がよいです。

## 11. 次にやるとよいこと

この教材の次にやるなら、順番としては次がおすすめです。

1. `PUT /tasks/{id}` を追加する
2. pagination を追加する
3. transaction を使う処理を入れる
4. 認証 middleware を入れる
5. Docker / CI / lint を追加する
6. `sqlmock` や Testcontainers も触る

## 12. 一般的な Go バックエンドの作法

- `main` は薄くする
- 標準ライブラリで済むならまず標準ライブラリ
- interface は producer 側ではなく consumer 側に置く
- `context.Context` は第一引数で渡す
- JSON decode 時は `DisallowUnknownFields` を検討する
- error は握り潰さず `%w` で包む
- ログには request ID を入れる
- DB をまたぐテストを最低 1 本は入れる

## 13. 完成版を読む順番

完成版を読むなら次の順番がおすすめです。

1. [internal/task/service_test.go](../reference/complete-app/internal/task/service_test.go)
2. [internal/api/handler_test.go](../reference/complete-app/internal/api/handler_test.go)
3. [integration/api_integration_test.go](../reference/complete-app/integration/api_integration_test.go)
4. [internal/task/task.go](../reference/complete-app/internal/task/task.go)
5. [internal/task/service.go](../reference/complete-app/internal/task/service.go)
6. [internal/api/handler.go](../reference/complete-app/internal/api/handler.go)
7. [internal/store/sqlite/task_repository.go](../reference/complete-app/internal/store/sqlite/task_repository.go)
8. [cmd/api/main.go](../reference/complete-app/cmd/api/main.go)
