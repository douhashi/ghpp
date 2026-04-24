# 開発ガイドライン

## アーキテクチャ

- Go による1バイナリ構成の CLI ツール

## 技術スタック

| 用途 | ライブラリ |
|---|---|
| GitHub GraphQL API | [githubv4](https://github.com/shurcooL/githubv4) |
| 環境変数 (.env) | [godotenv](https://github.com/joho/godotenv) |
| CLI オプション解析 | [flag](https://pkg.go.dev/flag) (Go 標準ライブラリ) |

## ディレクトリ構成（予定）

```
.
├── main.go          # エントリポイント
├── internal/
│   ├── config/      # 環境変数・設定の読み込み
│   ├── github/      # GitHub API クライアント
│   ├── promote/     # 昇格ロジック
│   ├── demote/      # 降格ロジック
│   └── urlutil/     # GitHub URL のパースユーティリティ
├── docs/            # ドキュメント
├── .env.example     # 環境変数のサンプル
└── go.mod
```

## コマンド

### promote

滞留 Issue を次のステータスへ昇格させる。

```
ghpp promote [flags]
  --promote-ready-enabled       plan→ready自動昇格を有効化する (env: GHPP_PROMOTE_READY_ENABLED, default: false)
  --planned-label <label>       plan→ready昇格トリガーとなるラベル名 (env: GHPP_PLANNED_LABEL, default: planned)
```

### demote

滞留 Issue を前のステータスへ降格させる。`--stale-threshold` で設定した期間（デフォルト: 2h）以上 Status 遷移が行われていないアイテムが対象。stale 判定は Issue 本体の更新日時ではなく、Status フィールド（`ProjectV2ItemFieldSingleSelectValue`）の `updatedAt` を基準とする。

```
ghpp demote [flags]
  --stale-threshold <duration>  降格対象とみなす滞留期間 (env: GHPP_STALE_THRESHOLD, default: 2h)
  --dry-run                     実際には更新せずに降格対象を表示する
```

## ビルド・実行

```bash
go build -o ghpp .
./ghpp
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
