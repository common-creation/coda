# Task 051: UIスタイル定義の実装

## 概要
lipglossを使用してUIコンポーネントの統一的なスタイルを定義する。

## 実装内容
1. `internal/ui/styles.go`の作成:
   ```go
   type Styles struct {
       // 基本カラー
       Primary      lipgloss.Color
       Secondary    lipgloss.Color
       Success      lipgloss.Color
       Error        lipgloss.Color
       
       // コンポーネントスタイル
       ChatMessage  lipgloss.Style
       UserInput    lipgloss.Style
       StatusBar    lipgloss.Style
       Border       lipgloss.Style
       
       // テキストスタイル
       Bold         lipgloss.Style
       Italic       lipgloss.Style
       Code         lipgloss.Style
   }
   ```

2. テーマ管理:
   ```go
   type Theme interface {
       GetStyles() Styles
       GetName() string
   }
   
   var (
       LightTheme Theme
       DarkTheme  Theme
       CustomTheme Theme
   )
   ```

3. カラースキーム:
   - ライトテーマ
   - ダークテーマ
   - ハイコントラスト
   - カスタムテーマ

4. レスポンシブスタイル:
   - 画面サイズに応じた調整
   - パディングの動的調整
   - フォントサイズ（ターミナルサポート時）

5. アクセシビリティ:
   - カラーブラインド対応
   - コントラスト比の確保
   - 読みやすさの最適化

## 完了条件
- [ ] 統一的なスタイルが定義されている
- [ ] テーマ切り替えが可能
- [ ] レスポンシブに対応
- [ ] アクセシブルなデザイン

## 依存関係
- task-048-ui-app-structure

## 推定作業時間
1.5時間