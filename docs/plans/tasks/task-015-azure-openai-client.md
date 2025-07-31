# Task 015: Azure OpenAIクライアント実装

## 概要
Azure OpenAI Serviceとの通信を行うクライアントを実装し、統一インターフェースに準拠させる。

## 実装内容
1. `internal/ai/azure.go`の作成:
   ```go
   type AzureClient struct {
       client       *openai.Client
       config       AIConfig
       deploymentID string
   }
   
   func NewAzureClient(config AIConfig) (*AzureClient, error)
   ```

2. Azure固有の設定:
   - エンドポイントURL構築
   - APIバージョンの指定
   - デプロイメント名の管理
   - 認証ヘッダーの設定

3. インターフェースメソッドの実装:
   - Azure特有のエンドポイント形式への対応
   - デプロイメント名を使用したリクエスト
   - エラーレスポンスの変換

4. 設定検証:
   - 必須パラメータの確認
   - エンドポイントURLの検証
   - APIキー形式の確認

## 完了条件
- [ ] Azure OpenAI Serviceと正常に通信できる
- [ ] デプロイメント名が正しく使用される
- [ ] エラーハンドリングが適切
- [ ] OpenAIクライアントと同等の機能を提供

## 依存関係
- task-010-ai-client-interface
- task-011-ai-types-definition
- task-012-ai-error-types

## 推定作業時間
1.5時間