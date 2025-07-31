# Task 050: UIイベントハンドリングの実装

## 概要
ユーザー入力やシステムイベントを処理するUpdateメソッドを実装する。

## 実装内容
1. `internal/ui/update.go`の作成:
   ```go
   func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
       switch msg := msg.(type) {
       case tea.KeyMsg:
           return m.handleKeyPress(msg)
       case tea.WindowSizeMsg:
           return m.handleResize(msg)
       case ChatResponseMsg:
           return m.handleChatResponse(msg)
       case ErrorMsg:
           return m.handleError(msg)
       }
       
       // 子ビューのアップデート
       return m.updateActiveView(msg)
   }
   ```

2. キーイベント処理:
   - グローバルキーバインド
   - ビュー固有のキー処理
   - モード切り替え
   - 特殊キーの処理

3. カスタムメッセージ:
   ```go
   type ChatResponseMsg struct {
       Content string
       IsStream bool
       Done    bool
   }
   
   type ToolExecutionMsg struct {
       Tool   string
       Status string
       Result interface{}
   }
   ```

4. 非同期処理:
   - API呼び出し
   - ファイル操作
   - バックグラウンドタスク
   - プログレス更新

5. 状態遷移:
   - ローディング状態
   - エラー状態
   - 成功状態
   - アイドル状態

## 完了条件
- [ ] キー入力が正しく処理される
- [ ] ウィンドウリサイズに対応
- [ ] 非同期処理が適切に管理される
- [ ] 状態遷移がスムーズ

## 依存関係
- task-049-ui-model

## 推定作業時間
2時間