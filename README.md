# Github Project Promoter (GHPP)

GitHub Projects をベースにした、プロジェクト進行ワークフローを支援する CLI ツール。

## 機能

- GitHub Projects の Item(Issue) をフェッチしてローカルにキャッシュ
- 定義済みの昇格ルールに基づいて Issue のステータスを自動 Promote
  - **計画フェーズ**: `inbox` → `plan`（上限数付き）
  - **実行フェーズ**: `ready` → `doing`（リポジトリ単位で1つまで）

## セットアップ

```bash
# ビルド
go build -o ghpp .

# 環境変数の設定（.env ファイルまたは直接指定）
export GH_TOKEN=your_token
export GHPP_OWNER=your_org
export GHPP_PROJECT_NUMBER=1
```

## コマンド

| コマンド | 説明 |
|---|---|
| `ghpp fetch` | GitHub Projects の Item(Issue) をフェッチしてローカルにキャッシュ |
| `ghpp promote` | 昇格ルールに基づいて Issue のステータスを自動 Promote |

## 環境変数

| 変数名 | 必須 | デフォルト値 | 説明 |
|---|---|---|---|
| `GH_TOKEN` | Yes | - | GitHub API トークン |
| `GHPP_OWNER` | Yes | - | GitHub Organization / User 名 |
| `GHPP_PROJECT_NUMBER` | Yes | - | GitHub Projects の番号 |
| `GHPP_STATUS_INBOX` | No | `Backlog` | inbox に対応するステータス名 |
| `GHPP_STATUS_PLAN` | No | `Plan` | plan に対応するステータス名 |
| `GHPP_STATUS_READY` | No | `Ready` | ready に対応するステータス名 |
| `GHPP_STATUS_DOING` | No | `In progress` | doing に対応するステータス名 |
| `GHPP_PLAN_LIMIT` | No | `3` | 計画フェーズの昇格上限数 |

## ライセンス

MIT
