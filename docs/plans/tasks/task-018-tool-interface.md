# Task 018: ツールインターフェースの定義

## 概要
すべてのツールが実装すべき統一的なインターフェースを定義し、ツールの拡張性を確保する。

## 実装内容
1. `internal/tools/interface.go`の作成:
   ```go
   type Tool interface {
       // ツール名を返す
       Name() string
       // ツールの説明を返す
       Description() string
       // パラメータスキーマを返す（JSON Schema形式）
       Schema() ToolSchema
       // ツールを実行する
       Execute(ctx context.Context, params map[string]interface{}) (interface{}, error)
       // 実行前の検証（オプション）
       Validate(params map[string]interface{}) error
   }
   
   type ToolSchema struct {
       Type       string                 `json:"type"`
       Properties map[string]Property    `json:"properties"`
       Required   []string              `json:"required"`
   }
   ```

2. ツール実行コンテキスト:
   ```go
   type ExecutionContext struct {
       WorkingDir string
       User       string
       Timeout    time.Duration
       Logger     Logger
   }
   ```

3. ツール結果の型:
   ```go
   type Result struct {
       Success bool
       Data    interface{}
       Error   error
       Logs    []string
   }
   ```

4. ヘルパー関数:
   - パラメータの型変換
   - スキーマ検証
   - 共通エラー処理

## 完了条件
- [ ] インターフェースが明確に定義されている
- [ ] JSON Schema互換のスキーマ定義
- [ ] 実装例がドキュメントに含まれている
- [ ] 拡張性が考慮されている

## 依存関係
- task-017-tool-manager

## 推定作業時間
45分