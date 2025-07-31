# Task 012: AIエラー型定義

## 概要
AIクライアントで発生する可能性のあるエラーを体系的に定義し、適切なエラーハンドリングを可能にする。

## 実装内容
1. `internal/ai/errors.go`の作成:
   ```go
   // 基本エラー型
   type Error struct {
       Type    ErrorType
       Message string
       Cause   error
       Details map[string]interface{}
   }
   
   // エラータイプ
   type ErrorType string
   
   const (
       ErrTypeAuthentication ErrorType = "authentication"
       ErrTypeRateLimit      ErrorType = "rate_limit"
       ErrTypeInvalidRequest ErrorType = "invalid_request"
       ErrTypeNetwork        ErrorType = "network"
       ErrTypeTimeout        ErrorType = "timeout"
       ErrTypeServerError    ErrorType = "server_error"
   )
   ```

2. エラー判定ヘルパー:
   ```go
   func IsRateLimitError(err error) bool
   func IsAuthenticationError(err error) bool
   func IsRetryableError(err error) bool
   ```

3. エラーラッピング:
   - 元のエラー情報の保持
   - スタックトレースの追加（デバッグモード時）
   - コンテキスト情報の付加

## 完了条件
- [ ] 全てのエラータイプが定義されている
- [ ] エラー判定関数が実装されている
- [ ] エラーメッセージが分かりやすい
- [ ] 単体テストでエラー処理が検証されている

## 依存関係
- task-011-ai-types-definition

## 推定作業時間
30分