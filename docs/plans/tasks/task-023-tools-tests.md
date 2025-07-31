# Task 023: ツール群の単体テスト作成

## 概要
実装した各ツール（ファイル操作、リスト、検索）の包括的な単体テストを作成する。

## 実装内容
1. テストファイルの作成:
   - `internal/tools/file_test.go`
   - `internal/tools/list_test.go`
   - `internal/tools/search_test.go`

2. ファイル操作ツールのテスト:
   - 一時ディレクトリでの動作確認
   - 権限エラーのシミュレーション
   - 大きなファイルの処理
   - 同時実行時の安全性

3. リストツールのテスト:
   - 複雑なディレクトリ構造での動作
   - シンボリックリンクの処理
   - フィルタリングの正確性
   - パフォーマンステスト

4. 検索ツールのテスト:
   - 各種エンコーディングでの検索
   - 正規表現パターンのテスト
   - エッジケース（空ファイル、巨大ファイル）
   - 並列処理の正確性

5. 共通テストユーティリティ:
   ```go
   // テスト用のファイル/ディレクトリ作成
   func CreateTestFileTree(t *testing.T) string
   // 結果の検証ヘルパー
   func AssertToolResult(t *testing.T, result interface{}, expected interface{})
   ```

## 完了条件
- [ ] 各ツールのカバレッジが80%以上
- [ ] エッジケースが網羅されている
- [ ] テストが独立して実行可能
- [ ] CI環境でも安定して動作する

## 依存関係
- task-020-file-tools
- task-021-list-tool
- task-022-search-tool

## 推定作業時間
2時間