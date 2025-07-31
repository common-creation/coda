# Task 046: コマンドヘルプの充実

## 概要
各コマンドの詳細なヘルプとドキュメントを整備する。

## 実装内容
1. ヘルプテキストの改善:
   ```go
   rootCmd.Long = `CODA (Coding Agent) is an AI-powered assistant that helps you
   write, understand, and manage code through natural language interaction.
   
   It provides intelligent code suggestions, can read and modify files,
   search through codebases, and execute various development tasks.`
   ```

2. コマンド別ヘルプ:
   - 各コマンドの詳細説明
   - 利用可能なフラグの説明
   - デフォルト値の明示
   - 環境変数の説明

3. カテゴリ別グループ化:
   ```go
   rootCmd.AddGroup(&cobra.Group{
       ID:    "core",
       Title: "Core Commands:",
   })
   
   chatCmd.GroupID = "core"
   configCmd.GroupID = "core"
   ```

4. ヘルプトピック:
   ```go
   var helpGettingStarted = &cobra.Command{
       Use:   "getting-started",
       Short: "Getting started with CODA",
       Long:  gettingStartedText,
   }
   ```

5. インタラクティブヘルプ:
   - ヘルプの検索機能
   - 関連コマンドの提案
   - よくある質問（FAQ）
   - トラブルシューティング

## 完了条件
- [ ] 全コマンドにヘルプがある
- [ ] ヘルプが分かりやすく整理されている
- [ ] 初心者にも理解しやすい
- [ ] エラー時に適切なヘルプが表示される

## 依存関係
- task-038-root-command
- task-039-chat-command
- task-040-config-command

## 推定作業時間
1時間