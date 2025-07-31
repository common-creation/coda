# Task 043: 設定ファイル指定オプションの実装

## 概要
コマンドラインから設定ファイルのパスを指定できるオプションを実装する。

## 実装内容
1. 設定ファイルフラグの実装:
   ```go
   var cfgFile string
   
   func init() {
       rootCmd.PersistentFlags().StringVarP(&cfgFile, "config", "c", "", 
           "config file (default: $HOME/.config/coda/config.yaml)")
   }
   ```

2. 設定ファイル優先順位:
   1. コマンドライン引数（--config）
   2. 環境変数（CODA_CONFIG）
   3. カレントディレクトリ（.coda/config.yaml）
   4. ユーザー設定（~/.config/coda/config.yaml）
   5. システム設定（/etc/coda/config.yaml）

3. ファイル検証:
   - ファイルの存在確認
   - 読み取り権限チェック
   - YAML/JSON形式の検証
   - スキーマ検証

4. エラー処理:
   - ファイルが見つからない場合
   - パース失敗時の詳細エラー
   - 権限エラーの明確化
   - デフォルトへのフォールバック

5. 設定マージ:
   - 複数の設定源からのマージ
   - 部分的な設定の上書き
   - 環境固有の設定

## 完了条件
- [ ] --configフラグが機能する
- [ ] 設定ファイルが正しく読み込まれる
- [ ] エラーメッセージが分かりやすい
- [ ] 優先順位が正しく動作する

## 依存関係
- task-008-config-loader
- task-038-root-command

## 推定作業時間
45分