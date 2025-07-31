# Task 014: OpenAIクライアントのテスト作成

## 概要
OpenAIクライアントの単体テストを作成し、モックを使用して外部依存なしにテストを実行できるようにする。

## 実装内容
1. `internal/ai/openai_test.go`の作成:
   - HTTPモックサーバーの使用
   - 各メソッドのテストケース
   - エラーケースのテスト

2. テストケースの実装:
   ```go
   func TestChatCompletion(t *testing.T)
   func TestChatCompletionStream(t *testing.T)
   func TestListModels(t *testing.T)
   func TestPing(t *testing.T)
   func TestErrorHandling(t *testing.T)
   func TestRetryLogic(t *testing.T)
   ```

3. モックレスポンスの作成:
   - 正常系レスポンス
   - エラーレスポンス（401, 429, 500等）
   - ストリーミングレスポンス

4. テストヘルパー:
   - モックサーバーのセットアップ
   - テスト用設定の生成
   - アサーションヘルパー

## 完了条件
- [ ] カバレッジ80%以上を達成
- [ ] 全ての公開メソッドがテストされている
- [ ] エッジケースがカバーされている
- [ ] テストが高速に実行される（外部依存なし）

## 依存関係
- task-013-openai-client-implementation

## 推定作業時間
1.5時間