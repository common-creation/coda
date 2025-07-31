# Task 007: 設定構造体の定義

## 概要
CODAの設定を管理するための構造体を定義し、設定管理の基盤を構築する。

## 実装内容
1. `internal/config/config.go`の作成:
   ```go
   type Config struct {
       // AI設定
       AI AIConfig
       // ツール設定
       Tools ToolsConfig
       // UI設定
       UI UIConfig
       // ロギング設定
       Logging LoggingConfig
   }
   
   type AIConfig struct {
       Provider string // "openai" or "azure"
       APIKey   string
       Model    string
       // OpenAI固有
       BaseURL string
       // Azure固有
       Endpoint string
       DeploymentName string
   }
   ```

2. デフォルト値の定義:
   - NewDefaultConfig()関数の実装
   - 環境変数からの読み込みサポート

3. バリデーション機能:
   - Validate()メソッドの実装
   - 必須フィールドのチェック

## 完了条件
- [ ] 設定構造体が定義されている
- [ ] デフォルト値が適切に設定される
- [ ] バリデーションが正しく動作する
- [ ] 単体テストが作成されている

## 依存関係
- task-006-dependencies-setup

## 推定作業時間
45分