# Task 010: AIクライアント統一インターフェースの定義

## 概要
OpenAIとAzure OpenAIの両方に対応する統一的なインターフェースを定義する。

## 実装内容
1. `internal/ai/client.go`の作成:
   ```go
   type Client interface {
       // チャット完了
       ChatCompletion(ctx context.Context, req ChatRequest) (*ChatResponse, error)
       // ストリーミングチャット
       ChatCompletionStream(ctx context.Context, req ChatRequest) (StreamReader, error)
       // モデル一覧取得
       ListModels(ctx context.Context) ([]Model, error)
       // ヘルスチェック
       Ping(ctx context.Context) error
   }
   ```

2. ファクトリー関数の実装:
   ```go
   func NewClient(config AIConfig) (Client, error)
   ```

3. コンテキスト管理:
   - タイムアウト設定
   - キャンセレーション
   - リトライポリシー

## 完了条件
- [ ] インターフェースが明確に定義されている
- [ ] ファクトリー関数が実装されている
- [ ] ドキュメントコメントが充実している
- [ ] インターフェースの使用例が含まれている

## 依存関係
- task-007-config-structure

## 推定作業時間
30分