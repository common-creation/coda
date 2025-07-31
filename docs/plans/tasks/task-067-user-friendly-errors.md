# Task 067: ユーザーフレンドリーなエラー表示の実装

## 概要
技術的なエラーをユーザーが理解しやすい形で表示するシステムを実装する。

## 実装内容
1. `internal/errors/display.go`の作成:
   ```go
   type ErrorDisplay struct {
       translator ErrorTranslator
       formatter  ErrorFormatter
       styles     ErrorStyles
   }
   
   type UserError struct {
       Title       string
       Message     string
       Suggestion  string
       Actions     []ErrorAction
       Technical   string // デバッグモード時のみ
   }
   ```

2. エラー変換例:
   ```go
   // 技術的エラー:
   "dial tcp: lookup api.openai.com: no such host"
   
   // ユーザー向け表示:
   Title:      "接続エラー"
   Message:    "AIサービスに接続できません"
   Suggestion: "インターネット接続を確認してください"
   Actions:    ["再試行", "オフライン設定", "詳細を見る"]
   ```

3. エラー表示UI:
   ```
   ┌─ ⚠ 接続エラー ──────────────────────────┐
   │                                         │
   │ AIサービスに接続できません              │
   │                                         │
   │ 考えられる原因:                         │
   │ • インターネット接続が切断されている    │
   │ • ファイアウォールがブロックしている    │
   │ • DNSの問題                            │
   │                                         │
   │ 対処法:                                 │
   │ 1. ネットワーク接続を確認              │
   │ 2. プロキシ設定を確認                  │
   │ 3. しばらく待ってから再試行            │
   │                                         │
   │ [R]etry  [S]ettings  [H]elp  [D]etails │
   └─────────────────────────────────────────┘
   ```

4. コンテキスト対応:
   - 実行中の操作に応じた説明
   - 過去のエラー履歴考慮
   - 環境固有の提案

5. 多言語対応準備:
   - メッセージのキー管理
   - 翻訳可能な構造
   - ロケール検出

## 完了条件
- [ ] エラーが分かりやすく表示される
- [ ] 対処法が明確に提示される
- [ ] 技術詳細も確認可能
- [ ] UIが統一されている

## 依存関係
- task-065-global-error-handler
- task-051-ui-styles

## 推定作業時間
1.5時間