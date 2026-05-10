# Github Project Promoter (GHPP)

GitHub Projects をベースにした、プロジェクト進行ワークフローを支援する CLI ツール。
定義済みの昇格ルールに基づいて Issue のステータスを自動的に Promote（昇格）させ、ステータス管理の運用コストを削減します。

## ステータスフロー

GHPP は2つのワークフローモードを提供します。

### `full` モード（デフォルト）

```
inbox → plan → ready → doing
```

| フェーズ | 遷移 | 制約 |
|---|---|---|
| **計画フェーズ** | `inbox` → `plan` | 一度に昇格する個数に上限あり（デフォルト: 3） |
| **準備フェーズ** | `plan` → `ready` | 指定ラベル付きの Issue のみ（デフォルト無効） |
| **実行フェーズ** | `ready` → `doing` | リポジトリ単位で1つまで |

### `simple` モード

```
inbox → doing
```

| フェーズ | 遷移 | 制約 |
|---|---|---|
| **実行フェーズ** | `inbox` → `doing` | リポジトリ単位で1つまで |

`--workflow=simple`（または `GHPP_WORKFLOW=simple`）で切り替え。Plan / Ready ステータスは使用しません。

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

# simple モード（Backlog → In progress の1段階のみ）
ghpp promote --workflow simple
```

**動作の詳細:**

1. **計画フェーズ** — `inbox` ステータスの Issue を `plan` に昇格。`--plan-limit` で指定された上限数まで昇格します。`--promote-plan-enabled=false` を指定すると計画フェーズの自動昇格は実行されません。人手や別フローで Plan へ積む運用に向けた opt-out オプションです。
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

#### `project init`

テンプレート GitHub Project (Projects v2) を複製し、対象リポジトリにリンクします。

```bash
# git リポジトリ配下で実行（カレントの remote.origin.url から repo を自動検出）
ghpp project init --token ghp_xxx --title "My Project"

# リポジトリを明示
ghpp project init --token ghp_xxx --title "My Project" --repo my-org/my-repo

# 別 owner にプロジェクトを作る
ghpp project init --token ghp_xxx --title "My Project" --owner other-org --repo my-org/my-repo

# テンプレートを差し替え（デフォルト: douhashi/25）
ghpp project init --title "My Project" --template-owner my-org --template-number 7

# dry-run（API mutation を呼ばずに resolve のみ実行）
ghpp project init --title "My Project" --dry-run

# 同名プロジェクトがあっても作成（衝突チェックをスキップ）
ghpp project init --title "My Project" --force
```

**動作の詳細:**

1. 同名プロジェクトの有無をチェック（`--force` で省略可能）
2. テンプレートプロジェクトと作成先 owner / リポジトリの GraphQL ノード ID を解決
3. `copyProjectV2` でテンプレートを複製
4. `linkProjectV2ToRepository` で対象リポジトリにリンク
5. 新プロジェクトの workflow 一覧を取得
6. JSON で結果を出力

**フラグ一覧:**

| フラグ | 環境変数 | 必須 | デフォルト値 | 説明 |
|---|---|---|---|---|
| `--token` | `GH_TOKEN` | Yes | - | GitHub API トークン（`project` write スコープと `repo` スコープが必要） |
| `--title` | - | Yes | - | 新規プロジェクトのタイトル |
| `--owner` | `GHPP_OWNER` | No | `--repo` の owner | プロジェクトを作成する user / org |
| `--repo` | - | No | git remote から自動検出 | リンクするリポジトリ（`owner/name` 形式） |
| `--template-owner` | `GHPP_TEMPLATE_OWNER` | No | `douhashi` | テンプレートプロジェクトの owner |
| `--template-number` | `GHPP_TEMPLATE_NUMBER` | No | `25` | テンプレートプロジェクトの番号 |
| `--force` | - | No | `false` | 同名プロジェクトが存在しても作成 |
| `--dry-run` | - | No | `false` | API mutation を呼ばずに resolve のみ実行 |

**出力例:**

```json
{
  "dry_run": false,
  "project": {
    "id": "PVT_kwDOA...",
    "number": 12,
    "url": "https://github.com/users/my-org/projects/12",
    "title": "My Project"
  },
  "linked_repository": "my-org/my-repo",
  "workflows": [
    { "name": "Item closed", "number": 1, "enabled": true },
    { "name": "Auto-add to project", "number": 2, "enabled": false }
  ],
  "manual_setup_needed": [
    "Auto-add to project: this workflow is NOT carried over by copyProjectV2. Open the new project's Workflows page and enable 'Auto-add to project' for the linked repository."
  ]
}
```

**トークンスコープ要件:**

- `project`（write）スコープ: `copyProjectV2` と `linkProjectV2ToRepository` の実行に必須
- `repo` スコープ: 対象リポジトリの ID 解決に必須

スコープ不足の場合は、エラーメッセージにトークン更新ページの URL（<https://github.com/settings/tokens>）が含まれます。

**テンプレート:**

デフォルトテンプレート `douhashi/25` は、本ツールの想定するステータスフロー（inbox/plan/ready/doing）に対応した workflow 構成を持っています。独自テンプレートを使う場合は `--template-owner` / `--template-number` を指定してください。

> **Note**: GitHub の `copyProjectV2` API は **"Auto-add to project" workflow を新プロジェクトにコピーしません**。出力 JSON の `manual_setup_needed` で案内されるとおり、新プロジェクトの Workflows 画面から手動で有効化してください。

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
| `--promote-plan-enabled` | `GHPP_PROMOTE_PLAN_ENABLED` | No | `true` | 計画フェーズ (backlog→plan) の自動昇格を有効化 |
| `--promote-ready-enabled` | `GHPP_PROMOTE_READY_ENABLED` | No | `false` | 準備フェーズ (plan→ready) の自動昇格を有効化（ラベルゲート） |
| `--planned-label` | `GHPP_PLANNED_LABEL` | No | `planned` | 準備フェーズの昇格トリガーとなるラベル名 |
| `--workflow` | `GHPP_WORKFLOW` | No | `full` | ワークフローモード（`full` / `simple`）。`simple` は Backlog と In progress の2状態のみで運用 |

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
