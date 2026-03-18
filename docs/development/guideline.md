# 開発ガイドライン

## アーキテクチャ

- Go による1バイナリ構成の CLI ツール

## 技術スタック

| 用途 | ライブラリ |
|---|---|
| GitHub GraphQL API | [githubv4](https://github.com/shurcooL/githubv4) |
| 環境変数 (.env) | [godotenv](https://github.com/joho/godotenv) |

## ディレクトリ構成（予定）

```
.
├── main.go          # エントリポイント
├── internal/
│   ├── config/      # 環境変数・設定の読み込み
│   ├── github/      # GitHub API クライアント
│   ├── promote/     # 昇格ロジック
│   └── cache/       # ローカルキャッシュ管理
├── docs/            # ドキュメント
├── .env.example     # 環境変数のサンプル
└── go.mod
```

## ビルド・実行

```bash
go build -o gh-project-promoter .
./gh-project-promoter
```

## リリース

GoReleaser を使用してリリースする。

### ローカルでのテストビルド

```bash
goreleaser release --snapshot --clean
```

### リリース手順

1. バージョンタグを作成する: `git tag -a v1.0.0 -m "Release v1.0.0"`
2. タグをプッシュする: `git push origin v1.0.0`
3. GitHub Actions が自動的に GoReleaser を実行し、GitHub Releases にバイナリがアップロードされる

### 設定ファイルの検証

```bash
goreleaser check
```
