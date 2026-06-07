# Go REST API Tutorial Workspace

このリポジトリは、ルートをあなたの学習用ワークスペースとして使う前提です。
進め方は「テストを先に書く」TDD ベースです。

- 教材: `docs/`
- あなたが実装する場所: `cmd/`, `internal/`, `integration/`
- 完成版の参照実装: `reference/complete-app/`

## 進め方

1. `mise install`
2. `docs/go-rest-api-tutorial.md` を読みながら、ルート配下に自分で TDD で実装する
3. 詰まったときだけ `reference/complete-app/` を見る

## 学習用ワークスペース

今は次のディレクトリだけ置いてあります。

- `cmd/api/`
- `internal/config/`
- `internal/api/`
- `internal/task/`
- `internal/store/sqlite/`
- `integration/`

ここに自分のコードを書いていってください。

## 最初のセットアップ

```bash
mise install
go mod init github.com/yourname/go-rest-api
go mod tidy
```

## 完成版を動かしたい場合

```bash
cd reference/complete-app
go test ./...
go test -tags=integration ./...
make run
```
