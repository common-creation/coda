# Task 039: チャットコマンドの実装

## 概要
インタラクティブなチャットセッションを開始するコマンドを実装する。

## 実装内容
1. `cmd/chat.go`の作成:
   ```go
   var chatCmd = &cobra.Command{
       Use:   "chat",
       Short: "Start an interactive chat session",
       Long:  `Start an interactive chat session with the AI assistant.`,
       RunE:  runChat,
   }
   ```

2. コマンドフラグ:
   - `--model`: 使用するAIモデル
   - `--no-stream`: ストリーミング無効化
   - `--session`: セッションID指定
   - `--continue`: 前回のセッション継続
   - `--no-tools`: ツール実行無効化

3. チャットループ:
   ```go
   func runChat(cmd *cobra.Command, args []string) error {
       // セットアップ
       handler := setupChatHandler()
       
       // メインループ
       for {
           input := readInput()
           if shouldExit(input) break
           
           err := handler.HandleMessage(ctx, input)
           if err != nil {
               handleError(err)
           }
       }
   }
   ```

4. 入力処理:
   - マルチライン入力のサポート
   - 履歴機能（上下キー）
   - 補完機能
   - Ctrl+C/Ctrl+Dの処理

5. 表示機能:
   - カラー出力
   - Markdown表示
   - プログレスインジケーター
   - エラー表示

## 完了条件
- [ ] `coda chat`でセッションが開始する
- [ ] 対話的なやり取りができる
- [ ] 各種フラグが機能する
- [ ] 終了処理が適切

## 依存関係
- task-030-chat-handler
- task-038-root-command

## 推定作業時間
1.5時間