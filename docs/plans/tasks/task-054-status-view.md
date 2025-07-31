# Task 054: ステータスバーの実装

## 概要
アプリケーションの状態や情報を表示するステータスバーを実装する。

## 実装内容
1. `internal/ui/views/status_view.go`の作成:
   ```go
   type StatusView struct {
       mode        string
       model       string
       tokenCount  int
       sessionID   string
       workingDir  string
       connected   bool
       styles      Styles
   }
   ```

2. 表示情報:
   - 現在のモード（Chat/Command）
   - 使用中のAIモデル
   - トークン使用量
   - セッションID
   - 作業ディレクトリ
   - 接続状態

3. レイアウト:
   ```
   ┌─────────────────────────────────────────────────┐
   │ Chat │ gpt-4 │ 1,234 tokens │ ~/project │ ● │
   └─────────────────────────────────────────────────┘
   ```

4. 動的更新:
   - リアルタイムトークンカウント
   - 接続状態の監視
   - エラー通知
   - プログレス表示

5. インタラクション:
   - クリック可能な要素（将来）
   - ツールチップ表示
   - コンテキストメニュー

## 完了条件
- [ ] 必要な情報が表示される
- [ ] レイアウトが整理されている
- [ ] 動的更新が適切
- [ ] 画面幅に応じて調整される

## 依存関係
- task-049-ui-model
- task-051-ui-styles

## 推定作業時間
1時間