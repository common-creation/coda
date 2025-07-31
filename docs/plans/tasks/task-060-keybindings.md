# Task 060: キーバインディングの実装

## 概要
アプリケーション全体のキーマップを定義し、カスタマイズ可能にする。

## 実装内容
1. `internal/ui/keybindings.go`の作成:
   ```go
   type KeyMap struct {
       // グローバル
       Quit        key.Binding
       Help        key.Binding
       Clear       key.Binding
       
       // ナビゲーション
       ScrollUp    key.Binding
       ScrollDown  key.Binding
       PageUp      key.Binding
       PageDown    key.Binding
       
       // 編集
       Submit      key.Binding
       Cancel      key.Binding
       Complete    key.Binding
       
       // カスタムバインディング
       Custom      map[string]key.Binding
   }
   ```

2. デフォルトキーマップ:
   ```go
   func DefaultKeyMap() KeyMap {
       return KeyMap{
           Quit:       key.NewBinding(key.WithKeys("ctrl+c")),
           Help:       key.NewBinding(key.WithKeys("?")),
           ScrollUp:   key.NewBinding(key.WithKeys("up", "k")),
           ScrollDown: key.NewBinding(key.WithKeys("down", "j")),
       }
   }
   ```

3. モード別キーマップ:
   - ノーマルモード（Vim風）
   - インサートモード
   - コマンドモード
   - 検索モード

4. カスタマイズ機能:
   - 設定ファイルからの読み込み
   - ランタイムでの変更
   - キーマップのリセット
   - 競合検出

5. ヘルプ統合:
   - キーバインド一覧の自動生成
   - コンテキスト別表示
   - ショートカットガイド

## 完了条件
- [ ] キーバインドが直感的
- [ ] カスタマイズが可能
- [ ] Vim/Emacsスタイルサポート
- [ ] ヘルプに反映される

## 依存関係
- task-050-ui-update
- task-055-help-view

## 推定作業時間
1.5時間