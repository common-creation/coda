# Task 034: ツール実行承認システムの実装

## 概要
危険な操作や重要な変更を行う前にユーザーの承認を求めるシステムを実装する。

## 実装内容
1. `internal/chat/approval.go`の作成:
   ```go
   type ApprovalHandler interface {
       RequestApproval(tool string, params map[string]interface{}) (bool, error)
       SetApprovalMode(mode ApprovalMode)
       GetHistory() []ApprovalRecord
   }
   
   type ApprovalMode int
   const (
       ApproveAll ApprovalMode = iota
       ApproveWrite
       ApproveNone
       Interactive
   )
   ```

2. 承認が必要な操作:
   - ファイルの書き込み/削除
   - システムコマンドの実行
   - 外部APIへのアクセス
   - 大量のファイル操作

3. 承認UI:
   - 操作内容の明確な表示
   - 影響範囲の提示
   - Y/N/A（Yes/No/Always）の選択
   - 詳細情報の表示オプション

4. 承認履歴:
   - 承認/拒否の記録
   - タイムスタンプ
   - 操作内容の保存
   - 監査ログ出力

5. 自動承認ルール:
   - 特定パスの自動承認
   - 特定操作の事前承認
   - セッション単位の設定

## 完了条件
- [ ] 危険な操作で承認が求められる
- [ ] UIが分かりやすい
- [ ] 履歴が正確に記録される
- [ ] 設定が柔軟に変更可能

## 依存関係
- task-033-tool-integration

## 推定作業時間
1.5時間