# Task 049: Bubbleteaモデルの実装

## 概要
UIの状態を管理するBubbleteaモデルを実装する。

## 実装内容
1. `internal/ui/model.go`の作成:
   ```go
   type Model struct {
       // UI状態
       width, height int
       ready        bool
       
       // ビューコンポーネント
       chatView     ChatView
       inputView    InputView
       statusView   StatusView
       helpView     HelpView
       
       // アプリケーション状態
       activeView   ViewType
       messages     []Message
       currentInput string
       
       // 設定
       config      *Config
       keymap      KeyMap
   }
   ```

2. Bubbleteaインターフェース実装:
   ```go
   func (m Model) Init() tea.Cmd
   func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd)
   func (m Model) View() string
   ```

3. 状態管理:
   - ビュー間の切り替え
   - メッセージ履歴
   - 入力状態
   - エラー状態
   - ローディング状態

4. レイアウト管理:
   - 動的なサイズ調整
   - ビューの配置
   - スクロール位置
   - フォーカス管理

5. イベントディスパッチ:
   - 子ビューへのイベント伝播
   - グローバルイベント処理
   - カスタムコマンド

## 完了条件
- [ ] モデルが正しく初期化される
- [ ] 状態更新が適切に行われる
- [ ] ビューが正しくレンダリングされる
- [ ] レイアウトが柔軟に調整される

## 依存関係
- task-048-ui-app-structure

## 推定作業時間
2時間