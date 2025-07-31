# Task 061: ショートカット処理の実装

## 概要
高度なキーボードショートカットとコマンドパレット機能を実装する。

## 実装内容
1. `internal/ui/shortcuts.go`の作成:
   ```go
   type ShortcutManager struct {
       shortcuts   map[string]ShortcutAction
       history     []string
       macros      map[string][]ShortcutAction
   }
   
   type ShortcutAction struct {
       Name        string
       Description string
       Keys        []string
       Action      func() tea.Cmd
   }
   ```

2. コマンドパレット:
   - Ctrl+P でパレット起動
   - ファジー検索
   - 最近使用したコマンド
   - カテゴリ分類

3. 組み込みショートカット:
   ```
   Ctrl+L     : Clear chat
   Ctrl+S     : Save session
   Ctrl+O     : Open session
   Ctrl+/     : Toggle comment
   Ctrl+Space : Trigger completion
   Alt+Enter  : Submit without tools
   ```

4. マクロ機能:
   - キー操作の記録
   - マクロの再生
   - マクロの保存/読み込み
   - 編集可能

5. コンテキストアクション:
   - ファイルパスでCtrl+Click → ファイルを開く
   - エラーメッセージで → 詳細表示
   - コードブロックで → コピー/実行

## 完了条件
- [ ] ショートカットが高速動作
- [ ] コマンドパレットが便利
- [ ] マクロ機能が動作する
- [ ] 発見しやすい設計

## 依存関係
- task-060-keybindings

## 推定作業時間
2時間