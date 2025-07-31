# Task 045: 非対話モードの実装

## 概要
スクリプトやCI/CD環境で使用するための非対話モードを実装する。

## 実装内容
1. 非対話モードフラグ:
   ```go
   var nonInteractive bool
   
   chatCmd.Flags().BoolVarP(&nonInteractive, "non-interactive", "n", false,
       "Run in non-interactive mode")
   chatCmd.Flags().StringP("message", "m", "", 
       "Single message to send (implies non-interactive)")
   ```

2. ワンショット実行:
   ```go
   // coda chat -m "質問内容"
   func runOneShot(message string) error {
       handler := setupChatHandler()
       err := handler.HandleMessage(ctx, message)
       // 結果を標準出力に出力
       return err
   }
   ```

3. パイプライン対応:
   - 標準入力からの読み取り
   - 標準出力への結果出力
   - エラーは標準エラー出力へ

4. 自動承認設定:
   - ツール実行の自動承認
   - --yes フラグのサポート
   - 危険な操作の制限

5. 出力フォーマット:
   - プレーンテキスト（デフォルト）
   - JSON出力（--json）
   - Markdown出力（--markdown）
   - 静音モード（--quiet）

## 完了条件
- [ ] -mフラグで単一メッセージが送信できる
- [ ] パイプラインで使用できる
- [ ] CI/CD環境で安定動作する
- [ ] 適切な終了コードが返される

## 依存関係
- task-039-chat-command

## 推定作業時間
1.5時間