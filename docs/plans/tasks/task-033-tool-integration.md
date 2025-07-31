# Task 033: ツールコール検出と実行の実装

## 概要
AIレスポンスからツールコールを検出し、適切なツールを実行する連携機能を実装する。

## 実装内容
1. `internal/chat/tools.go`の作成:
   ```go
   type ToolExecutor struct {
       manager    *tools.Manager
       validator  SecurityValidator
       approver   ApprovalHandler
   }
   
   func (e *ToolExecutor) DetectAndExecute(response ai.ChatResponse) ([]ToolResult, error)
   ```

2. ツールコール検出:
   - レスポンスからのツールコール抽出
   - パラメータの解析と検証
   - 複数ツールの同時呼び出し対応

3. 実行フロー:
   1. ツールコールの検出
   2. セキュリティチェック
   3. ユーザー承認（必要な場合）
   4. ツール実行
   5. 結果の収集
   6. AIへのフィードバック

4. 並列実行:
   - 独立したツールの並列実行
   - 依存関係の解決
   - 実行順序の最適化

5. エラーハンドリング:
   - ツール実行失敗の処理
   - 部分的な成功の扱い
   - リトライロジック

## 完了条件
- [ ] ツールコールが正確に検出される
- [ ] ツールが安全に実行される
- [ ] 並列実行が効率的
- [ ] エラーが適切に処理される

## 依存関係
- task-017-tool-manager
- task-024-security-validator
- task-030-chat-handler

## 推定作業時間
2時間