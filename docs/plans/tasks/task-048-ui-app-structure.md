# Task 048: UIアプリケーション構造の実装

## 概要
Bubbleteaを使用したTUIアプリケーションの基本構造を実装する。

## 実装内容
1. `internal/ui/app.go`の作成:
   ```go
   type App struct {
       program   *tea.Program
       model     Model
       config    *Config
       chatHandler *chat.Handler
   }
   
   func NewApp(config *Config) (*App, error) {
       model := NewModel(config)
       program := tea.NewProgram(model, tea.WithAltScreen())
       return &App{
           program: program,
           model:   model,
           config:  config,
       }, nil
   }
   ```

2. アプリケーションライフサイクル:
   - 初期化処理
   - メインループ
   - クリーンアップ
   - グレースフルシャットダウン

3. 依存性注入:
   - チャットハンドラー
   - ツールマネージャー
   - 設定管理
   - ロガー

4. エラー回復:
   - パニックハンドリング
   - 状態の保存
   - クラッシュレポート

5. シグナルハンドリング:
   - Ctrl+Cの処理
   - ウィンドウリサイズ
   - バックグラウンド/フォアグラウンド

## 完了条件
- [ ] アプリケーションが起動する
- [ ] 基本的なUIが表示される
- [ ] 終了処理が適切
- [ ] エラーからの回復が可能

## 依存関係
- task-006-dependencies-setup
- task-030-chat-handler

## 推定作業時間
1.5時間