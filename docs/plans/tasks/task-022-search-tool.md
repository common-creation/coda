# Task 022: ファイル検索ツールの実装

## 概要
ファイル内容の検索とファイル名の検索を行うツールを実装する。

## 実装内容
1. `internal/tools/search.go`の作成:

2. **SearchFilesTool**の実装:
   ```go
   type SearchParams struct {
       Path         string   `json:"path"`
       Query        string   `json:"query"`
       FilePattern  string   `json:"file_pattern"`
       CaseSensitive bool    `json:"case_sensitive"`
       UseRegex     bool     `json:"use_regex"`
       MaxResults   int      `json:"max_results"`
       Context      int      `json:"context"` // 前後の行数
   }
   ```

3. 検索機能:
   - ファイル内容の全文検索
   - 正規表現サポート
   - マルチバイト文字対応
   - バイナリファイルのスキップ

4. 最適化:
   - 並列検索の実装
   - メモリ効率的なストリーミング処理
   - インデックスキャッシュ（オプション）
   - 早期終了条件

5. 結果の整形:
   ```go
   type SearchResult struct {
       File      string
       Line      int
       Column    int
       Match     string
       Context   []string
   }
   ```

## 完了条件
- [ ] テキスト検索が高速に動作する
- [ ] 正規表現検索が正確に動作する
- [ ] 大規模ディレクトリでも効率的
- [ ] 検索結果が見やすく整形される

## 依存関係
- task-018-tool-interface

## 推定作業時間
2時間