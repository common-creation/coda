# Task 021: ファイルリスト表示ツールの実装

## 概要
ディレクトリ内のファイル一覧を取得し、整形して表示するツールを実装する。

## 実装内容
1. `internal/tools/list.go`の作成:

2. **ListFilesTool**の実装:
   ```go
   type ListFilesParams struct {
       Path      string   `json:"path"`
       Recursive bool     `json:"recursive"`
       Pattern   string   `json:"pattern"`
       MaxDepth  int      `json:"max_depth"`
       ShowHidden bool    `json:"show_hidden"`
       Sort      string   `json:"sort"` // name, size, time
   }
   ```

3. 主要機能:
   - ディレクトリの再帰的探索
   - ファイルパターンマッチング（glob/regex）
   - ファイル情報の収集（サイズ、更新日時、権限）
   - ツリー形式での表示オプション

4. フィルタリング機能:
   - ファイルタイプ別フィルタ
   - サイズによるフィルタ
   - 更新日時によるフィルタ
   - .gitignoreの考慮

5. 出力形式:
   - JSON形式
   - ツリー形式
   - 詳細リスト形式

## 完了条件
- [ ] ディレクトリ一覧が正確に取得できる
- [ ] フィルタリングが正常に動作する
- [ ] 大量ファイルでもパフォーマンスが良好
- [ ] アクセス権限エラーが適切に処理される

## 依存関係
- task-018-tool-interface

## 推定作業時間
1.5時間