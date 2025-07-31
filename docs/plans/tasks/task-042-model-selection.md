# Task 042: モデル選択オプションの実装

## 概要
コマンドラインからAIモデルを選択できるオプションを実装する。

## 実装内容
1. モデル選択フラグの追加:
   ```go
   // グローバルフラグとして追加
   rootCmd.PersistentFlags().StringP("model", "m", "", "AI model to use")
   
   // 環境変数サポート
   viper.BindPFlag("ai.model", rootCmd.PersistentFlags().Lookup("model"))
   ```

2. モデル一覧機能:
   ```go
   var listModelsCmd = &cobra.Command{
       Use:   "list-models",
       Short: "List available AI models",
       RunE: func(cmd *cobra.Command, args []string) error {
           // プロバイダー別にモデル一覧表示
       },
   }
   ```

3. モデル検証:
   - 指定されたモデルの存在確認
   - プロバイダーとの互換性チェック
   - デフォルトモデルへのフォールバック

4. プロバイダー自動選択:
   - モデル名からプロバイダーを推定
   - 明示的なプロバイダー指定も可能
   - エラー時の明確なメッセージ

5. モデル別設定:
   - 温度パラメータ
   - 最大トークン数
   - その他のモデル固有設定

## 完了条件
- [ ] --modelフラグでモデルが選択できる
- [ ] 無効なモデルが検出される
- [ ] モデル一覧が表示できる
- [ ] 設定との統合が適切

## 依存関係
- task-007-config-structure
- task-038-root-command

## 推定作業時間
1時間