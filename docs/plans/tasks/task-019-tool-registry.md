# Task 019: ツール登録システムの実装

## 概要
ツールの動的な登録と管理を行うレジストリシステムを実装する。

## 実装内容
1. `internal/tools/registry.go`の作成:
   ```go
   type Registry struct {
       tools      map[string]ToolFactory
       categories map[string][]string
       mu         sync.RWMutex
   }
   
   type ToolFactory func() Tool
   ```

2. 登録メソッド:
   - RegisterFactory(name string, factory ToolFactory)
   - RegisterCategory(category string, toolNames []string)
   - Unregister(name string)
   - GetByCategory(category string) []Tool

3. 自動登録機能:
   ```go
   // init()関数での自動登録サポート
   func init() {
       DefaultRegistry.RegisterFactory("read_file", NewReadFileTool)
   }
   ```

4. プラグインサポートの基盤:
   - 外部ツールの動的ロード準備
   - バージョン管理
   - 依存関係チェック

## 完了条件
- [ ] ツールの動的登録が可能
- [ ] カテゴリ別のツール管理ができる
- [ ] スレッドセーフな実装
- [ ] 重複登録の防止機能

## 依存関係
- task-018-tool-interface

## 推定作業時間
45分