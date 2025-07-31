# Task 053: 入力ビューの実装

## 概要
ユーザー入力を受け付ける高機能な入力エリアを実装する。

## 実装内容
1. `internal/ui/views/input_view.go`の作成:
   ```go
   type InputView struct {
       textInput    textinput.Model
       multiline    bool
       history      []string
       historyIndex int
       suggestions  []string
       styles       Styles
   }
   ```

2. 入力機能:
   - シングルライン/マルチライン切り替え
   - 履歴機能（上下キー）
   - 自動補完
   - プレースホルダー

3. マルチライン編集:
   - Shift+Enterで改行
   - Ctrl+Enterで送信
   - 基本的なエディタ機能
   - インデント保持

4. 入力補助:
   - コマンド補完（/で始まる）
   - ファイルパス補完
   - 履歴からの補完
   - スニペット機能

5. ビジュアルフィードバック:
   - 入力中インジケーター
   - 文字数カウント
   - エラー表示
   - フォーカス状態

## 完了条件
- [ ] テキスト入力が快適
- [ ] マルチラインが機能する
- [ ] 履歴機能が動作する
- [ ] 補完が有用

## 依存関係
- task-049-ui-model
- task-051-ui-styles

## 推定作業時間
1.5時間