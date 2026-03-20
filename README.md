# Github Project Promoter (GHPP)

GitHub Projects をベースにした、プロジェクト進行ワークフローを支援する CLI ツール。
定義済みの昇格ルールに基づいて Issue のステータスを自動的に Promote（昇格）させ、ステータス管理の運用コストを削減します。

## ステータスフロー

```
inbox → plan → ready → doing
```

GHPP は上記フローのうち、以下2つの昇格を自動化します。

| フェーズ | 遷移 | 制約 |
|---|---|---|
| **計画フェーズ** | `inbox` → `plan` | 一度に昇格する個数に上限あり（デフォルト: 3） |
| **実行フェーズ** | `ready` → `doing` | リポジトリ単位で1つまで |

## インストール

### リリースバイナリ

[Releases](https://github.com/douhashi/ghpp/releases) から対応プラットフォームのバイナリをダウンロードしてください。

### ソースからビルド

```bash
go build -o ghpp .
```

## 使い方

```
ghpp <command> [flags]
```

### サブコマンド

#### `promote`

昇格ルールに基づいて Issue のステータスを自動 Promote し、結果を JSON で出力します。

```bash
# 環境変数で設定済みの場合
ghpp promote

# フラグで指定
ghpp promote --token ghp_xxx --owner my-org --project-number 1

# 計画フェーズの上限数を変更
ghpp promote --plan-limit 5

# ステータス名をカスタマイズ
ghpp promote --status-inbox "Todo" --status-plan "Planned"
```

**動作の詳細:**

1. **計画フェーズ** — `inbox` ステータスの Issue を `plan` に昇格。`--plan-limit` で指定された上限数まで昇格します。
2. **実行フェーズ** — `ready` ステータスの Issue を `doing` に昇格。同一リポジトリで既に `doing` の Issue がある場合はスキップされます。

**出力例:**

```json
{
  "summary": { "promoted": 4, "skipped": 2, "total": 6 },
  "phases": {
    "plan": {
      "summary": { "promoted": 3, "skipped": 1, "total": 4 },
      "results": {
        "promoted": [
          {
            "item": { "id": "...", "title": "Issue title", "url": "https://...", "status": "Backlog" },
            "key": "plan-douhashi-ghpp-123",
            "to_status": "Plan"
          }
        ],
        "skipped": []
      }
    },
    "doing": {
      "summary": { "promoted": 1, "skipped": 1, "total": 2 },
      "results": {
        "promoted": [],
        "skipped": [
          {
            "item": { "id": "...", "title": "Issue title", "url": "https://...", "status": "Ready" },
            "reason": "repository already has doing issue"
          }
        ]
      }
    }
  }
}
```

- `phases.plan` / `phases.doing` は常にキーが存在（0件でも省略されない）
- 各 `results.promoted` / `results.skipped` は0件の場合 `[]`（`null` ではない）

## 設定

パラメータは **コマンドラインフラグ** または **環境変数** で指定できます。

**優先順位**: コマンドラインフラグ > 環境変数 > デフォルト値

### フラグ一覧

| フラグ | 環境変数 | 必須 | デフォルト値 | 説明 |
|---|---|---|---|---|
| `--token` | `GH_TOKEN` | Yes | - | GitHub API トークン |
| `--owner` | `GHPP_OWNER` | Yes | - | GitHub Organization / User 名 |
| `--project-number` | `GHPP_PROJECT_NUMBER` | Yes | - | GitHub Projects の番号 |
| `--status-inbox` | `GHPP_STATUS_INBOX` | No | `Backlog` | inbox に対応するステータス名 |
| `--status-plan` | `GHPP_STATUS_PLAN` | No | `Plan` | plan に対応するステータス名 |
| `--status-ready` | `GHPP_STATUS_READY` | No | `Ready` | ready に対応するステータス名 |
| `--status-doing` | `GHPP_STATUS_DOING` | No | `In progress` | doing に対応するステータス名 |
| `--plan-limit` | `GHPP_PLAN_LIMIT` | No | `3` | 計画フェーズの昇格上限数 |

> **セキュリティに関する注意**: `--token` でトークンを渡すと、プロセス一覧やシェル履歴にトークンが残る可能性があります。環境変数 `GH_TOKEN` または `.env` ファイルでの指定を推奨します。

### 環境変数 / `.env` ファイル

バイナリと同じディレクトリに `.env` ファイルを配置することで環境変数を設定できます。

```bash
GH_TOKEN=ghp_xxxxxxxxxxxx
GHPP_OWNER=my-org
GHPP_PROJECT_NUMBER=1
GHPP_PLAN_LIMIT=5
```

## ライセンス

MIT
