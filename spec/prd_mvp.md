# Product Requirements Document: ctxb (MVP)

| 項目 | 内容 |
| --- | --- |
| **Product Name** | **ctxb** (Context Builder) |
| **Version** | MVP (v0.2.0) |
| **Status** | **Final Draft** |
| **Language** | Go (Golang) |
| **Target** | Engineering Managers, Tech Leads, Developers |

## 1. Executive Summary

**"Stop feeding your AI stale context."**
`ctxb` は、ドキュメントの「鮮度」と「品質」を管理し、人間と AI の両方に最適なコンテキストを提供する CLI ツールである。
ドキュメントを "Code" として扱い、**Git のコミット履歴と Markdown のリンク構造**に基づいて、コードの変更に追従できていない「腐敗したドキュメント」を自動検知・可視化する。

## 2. Problem Statement

* **課題:** コードは頻繁に更新されるが、ドキュメントの更新は忘れられ、乖離（Rot）が進む。
* **影響:**
* **人間:** 嘘のドキュメントを読み、開発の手戻りが発生する。
* **AI:** Coding Agent (Cursor, Windsurf) に古い仕様を学習させてしまい、誤ったコード生成を引き起こす。


* **解決策:** ドキュメントとコードの依存関係を定義し、Git のタイムスタンプを用いて「鮮度」を機械的にテストする。

## 3. Core Principles

1. **No Vendor Lock-in:** 独自の依存定義ファイルを作らず、標準の Markdown リンク (`[link](../src/code.ts)`) を解析して依存関係とする。
2. **Git is Truth:** ドキュメントの正当性は、手動入力の日付ではなく、Git のコミットログに基づく。
3. **Visual & Actionable:** 腐敗状況を視覚的（CLI カラー/HTML グラフ）に提示し、即座に行動（修正）を促す。

## 4. Functional Requirements

### 4.1 CLI Commands

#### `ctxb init`

* **機能:** プロジェクトの初期化。
* **動作:** カレントディレクトリに設定ファイル `ctxb.yml` の雛形を生成する。

#### `ctxb check` (The Linter)

* **機能:** CI/CD パイプライン向けの品質ゲート。
* **検証ロジック:**
1. **Freshness (鮮度):** ドキュメント内でリンクされている「コードファイル」の最終更新日時と比較し、`Doc_Date < Code_Date` の場合に警告/エラーとする。
2. **Dead Links:** 存在しないファイルへのリンクを検知する。


* **出力:** 標準出力に問題のあるファイル一覧を表示し、エラー時は終了コード `1` を返す。

#### `ctxb docs` (The Dashboard)

* **機能:** 構造と健康状態の可視化。**`check` の診断結果を内包する。**
* **動作:**
1. **解析:** `check` と同様のロジックで全ファイルのステータス（Healthy/Stale/Broken）を判定する。
2. **メタデータ取得:** Git から「最終更新日」「最終更新者 (Author)」「Commit Hash」を取得する。
3. **レポート生成:** Mermaid.js を埋め込んだシングルページ HTML (`ctxb-report.html`) を生成し、ブラウザを開く。


* **UI仕様 (HTML):**
* **グラフ:** ファイルをノード、リンクをエッジとして描画。
* **色分け:**
* 🟢 **Green:** Healthy (最新)
* 🔴 **Red:** Stale (コードより古い / 長期間更新なし)
* 🟡 **Yellow:** Warning / Broken Link


* **詳細表示:** ノードをクリック/ホバー時、以下の情報を表示する。
* Author: `Taro Yamada`
* Last Modified: `2026-01-12`
* Status: `Stale (Depends on src/auth.ts which was updated 2 days ago)`





#### `ctxb build` (The Exporter)

* **機能:** AI 向けコンテキストの生成。
* **動作:** 設定に基づき、重要なドキュメントを結合・圧縮し、`.cursorrules` や `CLAUDE.md` などの単一ファイルを出力する。

### 4.2 Configuration (`ctxb.yml`)

```yaml
name: 'my_project'
version: '0.1.0'

paths:
  sources: ["docs"]    # ドキュメントのルート
  targets: ["src"]     # 監視対象コードのルート

rules:
  freshness:
    enabled: true
    warn_days: 30      # コード更新がなくても、30日経過で警告
    error_days: 90     # 90日でエラー

export:
  - output: ".cursorrules"
    include: ["docs/rules.md", "docs/arch/**"]

```

## 5. Technical Specifications

### 5.1 Technology Stack

* **Language:** Go (Golang)
* **Framework:** Cobra (CLI), Goldmark (Markdown Parser)
* **Distribution:** Single Binary

### 5.2 Data Flow & Logic

1. **Markdown Parsing:** AST (抽象構文木) を解析し、リンクノード (`[text](url)`) を抽出。相対パスを絶対パスへ解決する。
2. **Git Check:**
* 各ファイルに対して `git log -1 --format="%cd|%an|%h" --date=iso-strict <path>` を実行。
* 更新日時を比較し、Freshness Logic に基づいてステータスを決定。


3. **graph generation:**
* 解析結果（node, edge, status, metadata）を json データとして html テンプレート内の js 変数に注入する。
* クライアントサイド（mermaid.js）でレンダリングする。



## 6. milestones (mvp)

1. **step 1:** go プロジェクトのセットアップ & `ctxb init` 実装。
2. **step 2:** markdown 解析 & git 日付比較ロジックの実装 (`check` コマンド)。
3. **step 3:** html テンプレート作成 & mermaid 出力実装 (`docs` コマンド)。
4. **step 4:** readme 作成 & github 公開。

