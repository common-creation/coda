# Task 062: E2Eテストの作成

## 概要
アプリケーション全体の動作を検証するEnd-to-Endテストを作成する。

## 実装内容
1. `tests/e2e/`ディレクトリ構造:
   ```
   tests/e2e/
   ├── scenarios/
   │   ├── basic_chat_test.go
   │   ├── file_operations_test.go
   │   ├── tool_execution_test.go
   │   └── error_handling_test.go
   ├── fixtures/
   │   └── test_projects/
   └── helpers/
       └── test_utils.go
   ```

2. テストシナリオ:
   - 基本的な会話フロー
   - ファイル操作の一連の流れ
   - ツール実行と承認
   - エラー回復
   - セッション管理

3. テストヘルパー:
   ```go
   type E2ETestHelper struct {
       app     *App
       input   *TestInput
       output  *TestOutput
       cleanup []func()
   }
   
   func (h *E2ETestHelper) SendMessage(msg string)
   func (h *E2ETestHelper) WaitForResponse(timeout time.Duration)
   func (h *E2ETestHelper) AssertOutput(expected string)
   ```

4. モックとスタブ:
   - AI APIのモック
   - ファイルシステムの仮想化
   - 時間の制御
   - ランダム性の固定

5. CI/CD統合:
   - ヘッドレステスト実行
   - スクリーンショット取得
   - パフォーマンス計測
   - カバレッジレポート

## 完了条件
- [ ] 主要なユースケースがカバーされている
- [ ] テストが安定して動作する
- [ ] CI/CDで自動実行される
- [ ] 失敗時の診断が容易

## 依存関係
- task-048から061（UI実装全般）

## 推定作業時間
3時間