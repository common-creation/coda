# Task 044: デバッグモードの実装

## 概要
開発とトラブルシューティング用のデバッグモードを実装する。

## 実装内容
1. デバッグフラグの実装:
   ```go
   var debug bool
   
   rootCmd.PersistentFlags().BoolVar(&debug, "debug", false, 
       "Enable debug output")
   rootCmd.PersistentFlags().BoolVar(&trace, "trace", false, 
       "Enable trace-level output")
   ```

2. ログレベル管理:
   ```go
   func setupLogging() {
       if debug {
           log.SetLevel(log.DebugLevel)
       } else if trace {
           log.SetLevel(log.TraceLevel)
       } else {
           log.SetLevel(log.InfoLevel)
       }
   }
   ```

3. デバッグ情報の出力:
   - API リクエスト/レスポンス
   - ツール実行の詳細
   - プロンプトの内容
   - トークン使用量
   - 実行時間の計測

4. トレース機能:
   - 関数呼び出しの追跡
   - goroutineの監視
   - メモリ使用量
   - CPU使用率

5. デバッグコマンド:
   - `coda debug info`: システム情報表示
   - `coda debug config`: 解決済み設定表示
   - `coda debug tools`: ツール情報表示

## 完了条件
- [ ] --debugフラグで詳細ログが出力される
- [ ] API通信の内容が確認できる
- [ ] パフォーマンス情報が表示される
- [ ] 本番環境では無効化される

## 依存関係
- task-038-root-command

## 推定作業時間
1時間