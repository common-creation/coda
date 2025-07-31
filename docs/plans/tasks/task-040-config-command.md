# Task 040: 設定コマンドの実装

## 概要
設定の表示、編集、検証を行うコマンドを実装する。

## 実装内容
1. `cmd/config.go`の作成:
   ```go
   var configCmd = &cobra.Command{
       Use:   "config",
       Short: "Manage CODA configuration",
       Long:  `View, edit, and validate CODA configuration settings.`,
   }
   ```

2. サブコマンド:
   - `config show`: 現在の設定表示
   - `config set`: 設定値の変更
   - `config get`: 特定の設定値取得
   - `config init`: 設定ファイル初期化
   - `config validate`: 設定の検証

3. 設定表示機能:
   ```go
   var showCmd = &cobra.Command{
       Use:   "show",
       Short: "Show current configuration",
       RunE: func(cmd *cobra.Command, args []string) error {
           // JSON/YAML形式で表示
           // センシティブ情報はマスク
       },
   }
   ```

4. 設定編集機能:
   - キーバリューでの設定
   - ネストした値のサポート
   - 型の自動変換
   - 検証付き更新

5. APIキー管理:
   - `config set-api-key`: APIキー設定
   - セキュアな保存
   - プロバイダー別管理

## 完了条件
- [ ] 設定の表示が正しく動作する
- [ ] 設定の更新が保存される
- [ ] APIキーが安全に管理される
- [ ] 検証機能が働く

## 依存関係
- task-008-config-loader
- task-009-secrets-management
- task-038-root-command

## 推定作業時間
1.5時間