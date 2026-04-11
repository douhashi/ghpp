# Promote ライフサイクル

## ステータスフロー

```
inbox → plan → ready → doing
```

GHPP は上記フローのうち、以下2つの昇格を自動化する。

## 1. 計画フェーズ（inbox → plan）

Issue を `inbox` から `plan` ステータスに昇格させる。

### 動作

- `inbox` ステータスの Issue を取得し、`plan` ステータスに変更する
- 昇格した Issue の一覧を JSON で返す

### 制約

- 一度に昇格する個数に上限を設ける（環境変数 `GHPP_PLAN_LIMIT` で上書き可能）

## 2. 実行フェーズ（ready → doing）

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
- `phases.plan` / `phases.doing` は常にキーが存在する（0件でも省略されない）
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
