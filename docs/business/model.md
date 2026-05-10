# Promote ライフサイクル

## ワークフローモード

GHPP は2つのワークフローモードを提供する。

| モード | 自動遷移 | 使用するステータス | デフォルト |
|---|---|---|---|
| `full` | `inbox → plan → ready → doing` の3段階 | inbox / plan / ready / doing | ✓ |
| `simple` | `inbox → doing` の1段階 | inbox / doing |  |

`--workflow` フラグまたは `GHPP_WORKFLOW` 環境変数で切り替える。

> **以下「1. 計画フェーズ」「2. 準備フェーズ」「3. 実行フェーズ」は `full` モードの仕様**。`simple` モードについては末尾の「Simple モード仕様」を参照。

## ステータスフロー（full モード）

```
inbox → plan → ready → doing
```

GHPP は上記フローのうち、以下3つの昇格を自動化する。

## 1. 計画フェーズ（inbox → plan）

Issue を `inbox` から `plan` ステータスに昇格させる。

### 動作

- `inbox` ステータスの Issue を取得し、`plan` ステータスに変更する
- 昇格した Issue の一覧を JSON で返す

### 制約

- Plan カラムの WIP 上限（環境変数 `GHPP_PLAN_LIMIT` で上書き可能、デフォルト3）
- Plan 状態の Issue 数が PlanLimit 以上の場合、Backlog からの昇格は行わない
- Plan 状態が PlanLimit 未満の場合は `PlanLimit - 現在の Plan 数` の件数だけ昇格する（空き枠を埋める）
- Ready / Doing の Issue 数はカウント対象外

### 設定

| フラグ | 環境変数 | デフォルト | 説明 |
|-------|---------|-----------|------|
| `--promote-plan-enabled` | `GHPP_PROMOTE_PLAN_ENABLED` | `true` | 計画フェーズを有効化する |
| `--plan-limit` | `GHPP_PLAN_LIMIT` | `3` | Plan カラムの WIP 上限 |

- `--promote-plan-enabled=false` を指定すると計画フェーズはスキップされる（出力 JSON の `phases.plan` は空状態で残る）

## 2. 準備フェーズ（plan → ready）

Issue を `plan` から `ready` ステータスに昇格させる。**デフォルト無効**。

### 動作

- `plan` ステータスかつ指定ラベルを保持している Issue を `ready` ステータスに変更する
- 昇格後もラベルは剥がさない（永続マーカーとして保持）
- カスケード昇格を許可する（同一 `promote` 実行内で `plan → ready → doing` まで進む Issue があり得る）

### 制約

- **ラベルゲート**: 指定ラベルを持つ Issue のみが対象（ラベル付与自体がゲートとなるため上限なし）
- ラベルマッチは単一ラベルのみ（複数ラベル指定は扱わない）

### 設定

| フラグ | 環境変数 | デフォルト | 説明 |
|-------|---------|-----------|------|
| `--promote-ready-enabled` | `GHPP_PROMOTE_READY_ENABLED` | `false` | 準備フェーズを有効化する |
| `--planned-label` | `GHPP_PLANNED_LABEL` | `planned` | 昇格トリガーとなるラベル名 |

- `--promote-ready-enabled=true` かつ `--planned-label` が空の場合は設定エラー

## 3. 実行フェーズ（ready → doing）

Issue を `ready` から `doing` ステータスに昇格させる。

### 動作

- `ready` ステータスの Issue を取得し、`doing` ステータスに変更する

### 制約

- **リポジトリ単位で1つまで**: `doing` に昇格できるのは、各リポジトリにつき1つの Issue のみ
- すでに同リポジトリの Issue が `doing` にある場合、昇格しない
- リポジトリの判定は Issue URL から `owner/repository` を抽出して行う

## 出力フォーマット

Promote コマンドはフェーズ別サマリ付き JSON を出力する。

```json
{
  "summary": {
    "promoted": 4,
    "skipped": 2,
    "total": 6
  },
  "phases": {
    "plan": {
      "summary": {
        "promoted": 3,
        "skipped": 1,
        "total": 4
      },
      "results": [
        {
          "item": { "id": "...", "title": "...", "url": "...", "status": "..." },
          "action": "promoted",
          "to_status": "Plan"
        }
      ]
    },
    "ready": {
      "summary": {
        "promoted": 1,
        "skipped": 0,
        "total": 1
      },
      "results": [...]
    },
    "doing": {
      "summary": {
        "promoted": 1,
        "skipped": 1,
        "total": 2
      },
      "results": [...]
    }
  }
}
```

- トップレベルの `summary` は全フェーズの合計値
- `phases.plan` / `phases.ready` / `phases.doing` は常にキーが存在する（0件でも省略されない）
- 各フェーズの `results` は0件の場合 `[]`（`null` ではない）
- 各 result の `action` は `"promoted"` または `"skipped"`

### キーフォーマット

各 result の `key` は `{phase}-{owner}-{repository}-{issue_no}` 形式で生成される。

- `owner` は最大5文字、`repository` は最大10文字に切り詰められる
- `phase` と `issue_no` には切り詰めを適用しない
- キー全体は概ね最大32文字だが、`phase` と `issue_no` に切り詰めはないため、`issue_no` が大きい場合は超過しうる

---

## Demote コマンド仕様

demote コマンドは stale（更新から一定時間経過）なアイテムを降格させる。

### 対象フェーズ

| フェーズ | 条件 | 降格先 |
|--------|------|-------|
| doing  | `doing` ステータスで stale | `ready` |

> **注意**: `plan` フェーズ（Plan → Backlog）の降格は demote コマンドでは行わない。

### 出力フォーマット

```json
{
  "dry_run": false,
  "summary": {
    "demoted": 1,
    "skipped": 1,
    "total": 2
  },
  "phases": {
    "doing": {
      "summary": {
        "demoted": 1,
        "skipped": 1,
        "total": 2
      },
      "results": {
        "demoted": [...],
        "skipped": [...]
      }
    }
  }
}
```

- `phases.doing` は常にキーが存在する（0件でも省略されない）
- `results.demoted` / `results.skipped` は0件の場合 `[]`（`null` ではない）

> **simple モードでの降格先は `Backlog`（=inbox）**。降格先以外の挙動（stale 判定など）は `full` モードと同一。

---

## Simple モード仕様

`--workflow=simple` を指定すると、`inbox` と `doing` の2ステータスのみで運用するモードに切り替わる。

### 自動遷移

- promote: `Backlog → In progress`（plan / ready フェーズはスキップ）
- demote: stale な doing は `Backlog` に降格

### フラグの扱い

| フラグ | simple モードでの扱い |
|---|---|
| `--promote-plan-enabled` | silently 無視（warning は出さない） |
| `--promote-ready-enabled` | silently 無視（warning は出さない） |
| `--planned-label` | silently 無視（warning は出さない） |
| `--plan-limit` | silently 無視（plan フェーズが走らないため） |
| `--stale-threshold` | full モードと同一の挙動 |
| `--dry-run` | full モードと同一の挙動 |

### 制約

- 「リポジトリ単位で1つまで」ルールは維持される（doing に既存アイテムがあるリポジトリの Backlog はスキップされる）

### 出力 JSON

`phases.plan` / `phases.ready` キーは省略されず、`promoted` / `skipped` がそれぞれ空配列で出力される。
これにより full / simple 両モードで同一スキーマを保つ。

```json
{
  "phases": {
    "plan":  { "summary": { "promoted": 0, "skipped": 0, "total": 0 }, "results": { "promoted": [], "skipped": [] } },
    "ready": { "summary": { "promoted": 0, "skipped": 0, "total": 0 }, "results": { "promoted": [], "skipped": [] } },
    "doing": { "summary": { "promoted": 1, "skipped": 0, "total": 1 }, "results": { "promoted": [...], "skipped": [] } }
  }
}
```
