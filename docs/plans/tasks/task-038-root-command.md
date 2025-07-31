# Task 038: ルートコマンドの実装

## 概要
Cobraを使用してCLIの基本構造とルートコマンドを実装する。

## 実装内容
1. `cmd/root.go`の作成:
   ```go
   var rootCmd = &cobra.Command{
       Use:   "coda",
       Short: "CODA - AI-powered coding assistant",
       Long:  `CODA is an intelligent coding assistant that helps you write, 
               understand, and manage code through natural language interaction.`,
       RunE:  runRoot,
   }
   ```

2. グローバルフラグ:
   - `--config`: 設定ファイルパス
   - `--debug`: デバッグモード
   - `--quiet`: 静音モード
   - `--no-color`: カラー出力無効化
   - `--working-dir`: 作業ディレクトリ

3. 初期化処理:
   ```go
   func init() {
       cobra.OnInitialize(initConfig)
       rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file")
       rootCmd.PersistentFlags().BoolVar(&debug, "debug", false, "enable debug")
   }
   ```

4. 設定初期化:
   - 設定ファイルの読み込み
   - 環境変数の処理
   - ロガーの設定
   - 作業ディレクトリの設定

5. エラーハンドリング:
   - 分かりやすいエラーメッセージ
   - 終了コードの管理
   - スタックトレース（デバッグ時）

## 完了条件
- [ ] `coda`コマンドが実行できる
- [ ] ヘルプが適切に表示される
- [ ] グローバルフラグが機能する
- [ ] 設定が正しく初期化される

## 依存関係
- task-006-dependencies-setup
- task-008-config-loader

## 推定作業時間
1時間