# Task 013: OpenAIクライアント実装

## 概要
OpenAI APIとの通信を行うクライアントを実装し、統一インターフェースに準拠させる。

## 実装内容
1. `internal/ai/openai.go`の作成:
   ```go
   type OpenAIClient struct {
       client *openai.Client
       config AIConfig
   }
   
   func NewOpenAIClient(config AIConfig) (*OpenAIClient, error)
   ```

2. インターフェースメソッドの実装:
   - ChatCompletion: 通常のチャット完了
   - ChatCompletionStream: ストリーミング対応
   - ListModels: 利用可能なモデル一覧
   - Ping: API接続確認

3. エラーハンドリング:
   - APIエラーの変換
   - リトライロジック（429エラー時）
   - タイムアウト処理

4. 内部ヘルパー関数:
   - リクエスト/レスポンスの変換
   - ヘッダーの設定
   - レート制限の考慮

## 完了条件
- [ ] 全てのインターフェースメソッドが実装されている
- [ ] エラーが適切に変換されている
- [ ] ストリーミングが正常に動作する
- [ ] 接続テストが成功する

## 依存関係
- task-010-ai-client-interface
- task-011-ai-types-definition
- task-012-ai-error-types

## 推定作業時間
2時間