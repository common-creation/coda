# Task 055: ヘルプビューの実装

## 概要
キーバインドやコマンドの説明を表示するヘルプビューを実装する。

## 実装内容
1. `internal/ui/views/help_view.go`の作成:
   ```go
   type HelpView struct {
       visible     bool
       sections    []HelpSection
       viewport    viewport.Model
       activeTab   int
       styles      Styles
   }
   
   type HelpSection struct {
       Title    string
       Bindings []KeyBinding
   }
   ```

2. ヘルプコンテンツ:
   - グローバルキーバインド
   - ビュー固有のキーバインド
   - コマンド一覧
   - ショートカット
   - Tips & Tricks

3. 表示形式:
   ```
   ┌─ Help ─────────────────────────────────┐
   │ Global Commands                         │
   │ ───────────────                        │
   │ Ctrl+C    Exit application            │
   │ Ctrl+L    Clear screen                │
   │ ?         Toggle this help             │
   │                                        │
   │ Chat Commands                          │
   │ ─────────────                         │
   │ Enter     Send message                 │
   │ Ctrl+U    Clear input                  │
   └────────────────────────────────────────┘
   ```

4. ナビゲーション:
   - タブ切り替え
   - スクロール
   - 検索機能
   - クイックジャンプ

5. コンテキストヘルプ:
   - 現在のモードに応じた内容
   - 動的な更新
   - カスタマイズ可能

## 完了条件
- [ ] ヘルプが分かりやすく表示される
- [ ] キーバインドが網羅されている
- [ ] ナビゲーションが直感的
- [ ] トグル表示が適切

## 依存関係
- task-049-ui-model
- task-051-ui-styles

## 推定作業時間
1時間