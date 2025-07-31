# Task 032: システムプロンプト管理の実装

## 概要
AIの動作を制御するシステムプロンプトを管理し、動的に構築する仕組みを実装する。

## 実装内容
1. `internal/chat/prompts.go`の作成:
   ```go
   type PromptBuilder struct {
       basePrompt   string
       toolPrompts  map[string]string
       contextInfo  ContextInfo
   }
   
   type ContextInfo struct {
       WorkingDir   string
       Platform     string
       UserName     string
       ProjectInfo  map[string]string
   }
   ```

2. プロンプト構築機能:
   - 基本プロンプトの読み込み
   - ツール情報の動的追加
   - コンテキスト情報の注入
   - テンプレート処理

3. プロンプトテンプレート:
   ```
   You are CODA, a helpful coding assistant.
   Current directory: {{.WorkingDir}}
   Available tools: {{.ToolsList}}
   {{if .ProjectInfo}}Project: {{.ProjectInfo}}{{end}}
   ```

4. 動的更新:
   - ワークスペース情報の反映
   - ツールの有効/無効切り替え
   - ユーザー設定の反映
   - 言語設定

5. プロンプト最適化:
   - トークン数の管理
   - 重要度による優先順位付け
   - 冗長な情報の削除

## 完了条件
- [ ] システムプロンプトが適切に構築される
- [ ] 動的な更新が反映される
- [ ] トークン制限内に収まる
- [ ] テンプレートが拡張可能

## 依存関係
- task-027-session-management

## 推定作業時間
1時間