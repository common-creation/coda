# Task 056: シンタックスハイライトの実装

## 概要
コードブロックに対して言語別のシンタックスハイライトを適用する機能を実装する。

## 実装内容
1. `internal/ui/components/syntax_highlighter.go`の作成:
   ```go
   type SyntaxHighlighter struct {
       theme      HighlightTheme
       languages  map[string]Language
       cache      map[string]HighlightedCode
   }
   
   type HighlightedCode struct {
       Language string
       Lines    []HighlightedLine
       Theme    HighlightTheme
   }
   ```

2. 言語サポート:
   - Go
   - Python
   - JavaScript/TypeScript
   - Rust
   - JSON/YAML
   - Markdown
   - Shell

3. トークナイザー:
   - 言語別の字句解析
   - キーワード識別
   - 文字列/コメント検出
   - 演算子/区切り文字

4. カラーテーマ:
   ```go
   type HighlightTheme struct {
       Keyword    lipgloss.Style
       String     lipgloss.Style
       Comment    lipgloss.Style
       Function   lipgloss.Style
       Number     lipgloss.Style
       Operator   lipgloss.Style
   }
   ```

5. パフォーマンス最適化:
   - 増分ハイライト
   - キャッシング
   - 遅延処理
   - 部分的な再レンダリング

## 完了条件
- [ ] 主要言語がサポートされている
- [ ] ハイライトが正確
- [ ] パフォーマンスが良好
- [ ] テーマがカスタマイズ可能

## 依存関係
- task-051-ui-styles
- task-052-chat-view

## 推定作業時間
2.5時間