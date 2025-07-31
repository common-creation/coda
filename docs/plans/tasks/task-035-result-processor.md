# Task 035: ツール実行結果処理の実装

## 概要
ツール実行結果を処理し、AIへのフィードバックとユーザーへの表示を最適化する。

## 実装内容
1. `internal/chat/result_processor.go`の作成:
   ```go
   type ResultProcessor struct {
       formatter  OutputFormatter
       summarizer ContentSummarizer
       cache      ResultCache
   }
   
   func (p *ResultProcessor) ProcessResults(results []ToolResult) ProcessedOutput
   ```

2. 結果の整形:
   - ファイル内容の構文ハイライト
   - 大きな出力の要約
   - エラーメッセージの整形
   - 進捗情報の表示

3. AIフィードバック最適化:
   - 重要な情報の抽出
   - トークン数の調整
   - コンテキスト関連情報の付加
   - 実行メタデータの追加

4. キャッシング:
   - 頻繁にアクセスされる結果のキャッシュ
   - ファイル内容の差分管理
   - 有効期限の管理

5. 出力フォーマット:
   - プレーンテキスト
   - Markdown
   - JSON（デバッグ用）
   - カスタムフォーマット

## 完了条件
- [ ] 結果が見やすく整形される
- [ ] AIへのフィードバックが最適化される
- [ ] 大きな出力が適切に処理される
- [ ] キャッシュが効率的に動作する

## 依存関係
- task-033-tool-integration

## 推定作業時間
1.5時間