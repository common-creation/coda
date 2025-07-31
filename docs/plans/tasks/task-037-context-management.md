# Task 037: チャットコンテキスト管理の実装

## 概要
会話のコンテキストを効率的に管理し、関連情報をAIに提供するシステムを実装する。

## 実装内容
1. `internal/chat/context.go`の作成:
   ```go
   type ContextManager struct {
       currentPath   string
       openFiles     map[string]FileContext
       recentTools   []ToolExecution
       projectInfo   ProjectInfo
       maxContextSize int
   }
   
   type FileContext struct {
       Path         string
       LastModified time.Time
       Highlights   []CodeHighlight
       References   []string
   }
   ```

2. コンテキスト追跡:
   - 開いているファイルの管理
   - 最近実行したツールの記録
   - エラー発生箇所の追跡
   - 参照関係の把握

3. コンテキスト最適化:
   - 関連性スコアリング
   - 重要度による優先順位付け
   - 古い情報の自動削除
   - トークン数の管理

4. プロジェクト情報:
   - 言語/フレームワークの検出
   - 依存関係の把握
   - ビルドシステムの認識
   - 設定ファイルの解析

5. コンテキスト永続化:
   - セッション間でのコンテキスト保持
   - 効率的なシリアライゼーション
   - 増分更新

## 完了条件
- [ ] コンテキストが適切に追跡される
- [ ] AIに関連情報が提供される
- [ ] メモリ効率が良好
- [ ] パフォーマンスに影響しない

## 依存関係
- task-027-session-management
- task-036-workspace-support

## 推定作業時間
2時間