# Task 065: グローバルエラーハンドラーの実装

## 概要
アプリケーション全体で一貫したエラー処理を行うグローバルエラーハンドラーを実装する。

## 実装内容
1. `internal/errors/handler.go`の作成:
   ```go
   type ErrorHandler struct {
       logger      Logger
       reporters   []ErrorReporter
       fallback    FallbackHandler
       context     *ErrorContext
   }
   
   type ErrorContext struct {
       SessionID   string
       UserAction  string
       Timestamp   time.Time
       Metadata    map[string]interface{}
   }
   ```

2. エラー分類:
   ```go
   type ErrorCategory int
   const (
       UserError      ErrorCategory = iota // ユーザー起因
       SystemError                         // システムエラー
       NetworkError                        // ネットワーク関連
       ConfigError                         // 設定エラー
       SecurityError                       // セキュリティ違反
   )
   ```

3. エラー処理フロー:
   1. エラーのキャプチャ
   2. カテゴリ分類
   3. コンテキスト情報付加
   4. ロギング
   5. ユーザー通知
   6. リカバリー試行

4. ユーザー向けメッセージ:
   ```go
   func (h *ErrorHandler) UserMessage(err error) string {
       // 技術的詳細を隠蔽
       // 分かりやすい説明
       // 対処法の提示
       // サポート情報
   }
   ```

5. エラーレポート:
   - ローカルログファイル
   - デバッグ情報の収集
   - クラッシュダンプ
   - 匿名化された統計

## 完了条件
- [ ] 全エラーが適切に処理される
- [ ] ユーザーメッセージが分かりやすい
- [ ] デバッグ情報が十分
- [ ] プライバシーが保護される

## 依存関係
- task-012-ai-error-types

## 推定作業時間
1.5時間