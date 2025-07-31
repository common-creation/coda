# Task 006: 依存関係の初期設定

## 概要
プロジェクトで使用する主要なライブラリをインストールし、go.modを更新する。

## 実装内容
1. 主要ライブラリのインストール:
   ```bash
   # CLIフレームワーク
   go get github.com/spf13/cobra
   go get github.com/spf13/viper
   
   # TUIフレームワーク
   go get github.com/charmbracelet/bubbletea
   go get github.com/charmbracelet/bubbles
   go get github.com/charmbracelet/lipgloss
   
   # OpenAI SDK
   go get github.com/sashabaranov/go-openai
   
   # ユーティリティ
   go get github.com/sirupsen/logrus
   go get github.com/stretchr/testify
   ```

2. go.modとgo.sumの整理:
   - `go mod tidy`の実行
   - 不要な依存関係の削除

3. vendor/ディレクトリの設定（オプション）:
   - `go mod vendor`の実行
   - .gitignoreにvendor/を追加するか検討

## 完了条件
- [ ] 必要な依存関係がgo.modに記載されている
- [ ] `go mod tidy`でエラーが発生しない
- [ ] `go build ./...`が成功する

## 依存関係
- task-001-project-initialization

## 推定作業時間
20分