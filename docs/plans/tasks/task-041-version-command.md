# Task 041: バージョン表示コマンドの実装

## 概要
アプリケーションのバージョン情報を表示するコマンドを実装する。

## 実装内容
1. `cmd/version.go`の作成:
   ```go
   var versionCmd = &cobra.Command{
       Use:   "version",
       Short: "Display version information",
       Long:  `Display detailed version information about CODA.`,
       RunE:  runVersion,
   }
   ```

2. バージョン情報の管理:
   ```go
   // ビルド時に注入される変数
   var (
       Version   = "dev"
       Commit    = "unknown"
       Date      = "unknown"
       GoVersion = runtime.Version()
   )
   ```

3. 表示フォーマット:
   - 通常表示: `CODA version 1.0.0`
   - 詳細表示（--verbose）:
     ```
     CODA version 1.0.0
     Commit: abc123def
     Built: 2024-01-15T10:30:00Z
     Go version: go1.21.5
     Platform: darwin/amd64
     ```

4. 追加情報:
   - ビルド日時
   - Gitコミットハッシュ
   - Goバージョン
   - OS/アーキテクチャ
   - 有効な機能フラグ

5. 更新チェック（オプション）:
   - 最新バージョンの確認
   - 更新可能通知
   - 自動更新の案内

## 完了条件
- [ ] バージョン情報が表示される
- [ ] ビルド情報が正確
- [ ] 詳細モードが機能する
- [ ] CI/CDで情報が注入される

## 依存関係
- task-038-root-command

## 推定作業時間
30分