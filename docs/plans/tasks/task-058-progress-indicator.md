# Task 058: プログレスインジケーターの実装

## 概要
長時間実行される処理の進捗を視覚的に表示するコンポーネントを実装する。

## 実装内容
1. `internal/ui/components/progress_indicator.go`の作成:
   ```go
   type ProgressIndicator struct {
       spinner     spinner.Model
       progressBar progress.Model
       message     string
       percentage  float64
       isIndeterminate bool
       styles      Styles
   }
   ```

2. インジケータータイプ:
   - スピナー（処理中）
   - プログレスバー（進捗率あり）
   - ステップインジケーター
   - 複合型（スピナー＋メッセージ）

3. アニメーション:
   ```
   ⠋ Loading...
   ⠙ Processing files...
   ⠹ Almost done...
   
   [████████░░░░░░░] 53% (23/43 files)
   ```

4. コンテキスト情報:
   - 現在の処理内容
   - 推定残り時間
   - 処理済み/総数
   - エラー/警告カウント

5. 統合ポイント:
   - API呼び出し中
   - ファイル操作中
   - ツール実行中
   - 大量データ処理中

## 完了条件
- [ ] 各種インジケーターが表示される
- [ ] アニメーションがスムーズ
- [ ] 情報が分かりやすい
- [ ] パフォーマンスに影響しない

## 依存関係
- task-051-ui-styles
- task-050-ui-update

## 推定作業時間
1.5時間