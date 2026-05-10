# Github Project Promoter (GHPP)

## 概要

GitHub Projects をベースにした、プロジェクト進行のワークフローを支援するCLIツール。

## 解決する課題

GitHub Projects のステータス管理を手動で行う運用コストを削減し、定義済みの昇格ルールに基づいてIssueのステータスを自動的にPromote（昇格）させる。

## 主な機能

- **Promote**: GitHub Projects から Item(Issue) の一覧を取得し、昇格ルールに基づいてステータスを昇格させる
- **ステータスマッピング**: 管理対象ステータス (`inbox`, `plan`, `ready`, `doing`) と GitHub Issue 側のレーン(status) の対応を環境変数で指定可能

## 環境変数

| 変数名 | デフォルト値 | 説明 |
|---|---|---|
| `GH_TOKEN` | - | GitHub API トークン |
| `GHPP_OWNER` | - | GitHub Organization または User 名 |
| `GHPP_PROJECT_NUMBER` | - | GitHub Projects の番号 |
| `GHPP_STATUS_INBOX` | `Backlog` | inbox に対応する GitHub Projects のステータス名 |
| `GHPP_STATUS_PLAN` | `Plan` | plan に対応するステータス名 |
| `GHPP_STATUS_READY` | `Ready` | ready に対応するステータス名 |
| `GHPP_STATUS_DOING` | `In progress` | doing に対応するステータス名 |
| `GHPP_PLAN_LIMIT` | `3` | 計画フェーズで一度に昇格する上限数 |
| `GHPP_WORKFLOW` | `full` | ワークフローモード（`full` / `simple`）。`simple` は Backlog と In progress のみで運用 |

## 設定ファイル

バイナリと同じディレクトリに `.env` ファイルを配置することで、環境変数を設定できる。
