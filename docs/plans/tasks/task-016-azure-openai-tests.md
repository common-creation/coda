# Task 016: Azure OpenAIクライアントのテスト作成

## 概要
Azure OpenAIクライアントの単体テストを作成し、Azure固有の動作を検証する。

## 実装内容
1. `internal/ai/azure_test.go`の作成:
   - Azure APIレスポンス形式のモック
   - エンドポイント構築のテスト
   - デプロイメント名の使用確認

2. Azure固有のテストケース:
   ```go
   func TestAzureEndpointConstruction(t *testing.T)
   func TestAzureAuthentication(t *testing.T)
   func TestDeploymentNameUsage(t *testing.T)
   func TestAzureErrorResponse(t *testing.T)
   ```

3. 設定検証テスト:
   - 不正なエンドポイントURL
   - デプロイメント名の欠如
   - APIキー形式の検証

4. 互換性テスト:
   - OpenAIクライアントと同じテストケースでの動作確認
   - レスポンス形式の互換性

## 完了条件
- [ ] カバレッジ80%以上を達成
- [ ] Azure固有の動作が検証されている
- [ ] エラーケースが網羅されている
- [ ] モックが正確にAzure APIを再現している

## 依存関係
- task-015-azure-openai-client

## 推定作業時間
1時間