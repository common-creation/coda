# Task 057: Markdown描画機能の実装

## 概要
チャットメッセージ内のMarkdownを適切に描画する機能を実装する。

## 実装内容
1. `internal/ui/components/markdown_renderer.go`の作成:
   ```go
   type MarkdownRenderer struct {
       styles          Styles
       highlighter     *SyntaxHighlighter
       maxWidth        int
       preserveNewlines bool
   }
   
   func (r *MarkdownRenderer) Render(markdown string) string
   ```

2. サポート要素:
   - 見出し（H1-H6）
   - 段落とインライン要素
   - リスト（順序付き/順序なし）
   - コードブロック（言語指定）
   - インラインコード
   - 引用ブロック
   - 水平線
   - リンク（表示のみ）

3. ターミナル向け変換:
   ```markdown
   # 見出し → ━━━ 見出し ━━━
   **太字** → 太字（太字スタイル）
   `code` → code（コードスタイル）
   > 引用 → ┃ 引用
   ```

4. レイアウト処理:
   - 幅に応じた折り返し
   - インデント管理
   - 空白の最適化
   - テーブルの簡易表示

5. エスケープ処理:
   - 特殊文字の処理
   - ANSIエスケープとの競合回避
   - セキュリティ考慮

## 完了条件
- [ ] Markdownが読みやすく表示される
- [ ] コードブロックがハイライトされる
- [ ] レイアウトが適切
- [ ] パフォーマンスが良好

## 依存関係
- task-056-syntax-highlighter
- task-051-ui-styles

## 推定作業時間
2時間