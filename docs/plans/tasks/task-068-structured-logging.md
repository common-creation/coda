# Task 068: 構造化ログの実装

## 概要
デバッグとモニタリングのための構造化ログシステムを実装する。

## 実装内容
1. `internal/logging/logger.go`の作成:
   ```go
   type Logger struct {
       level      LogLevel
       outputs    []LogOutput
       fields     Fields
       sampling   SamplingConfig
   }
   
   type LogEntry struct {
       Timestamp   time.Time
       Level       LogLevel
       Message     string
       Fields      Fields
       Caller      string
       StackTrace  string
   }
   ```

2. ログレベルと出力:
   ```go
   // 構造化ログの例
   logger.With(Fields{
       "user_id":    session.ID,
       "action":     "tool_execution",
       "tool":       "read_file",
       "duration":   elapsed,
   }).Info("Tool executed successfully")
   ```

3. コンテキスト伝播:
   ```go
   type contextKey string
   
   func WithLogger(ctx context.Context, logger *Logger) context.Context
   func FromContext(ctx context.Context) *Logger
   ```

4. ログ出力先:
   - **開発環境**: コンソール（カラー出力）
   - **本番環境**: JSON形式ファイル
   - **デバッグ**: 詳細ログファイル
   - **エラー**: 別ファイルに分離

5. パフォーマンス考慮:
   - 非同期書き込み
   - バッファリング
   - ログローテーション
   - サンプリング（高頻度ログ）

6. プライバシー保護:
   ```go
   type Sanitizer interface {
       Sanitize(Fields) Fields
   }
   
   // APIキー、個人情報のマスキング
   // ファイルパスの匿名化オプション
   ```

## 完了条件
- [ ] 構造化ログが出力される
- [ ] パフォーマンスへの影響が最小
- [ ] ログから問題を追跡可能
- [ ] プライバシーが保護される

## 依存関係
- task-044-debug-mode

## 推定作業時間
1.5時間