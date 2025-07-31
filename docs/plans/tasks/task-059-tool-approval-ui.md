# Task 059: ツール承認UIの実装

## 概要
ツール実行前にユーザーの承認を求める対話的なUIコンポーネントを実装する。

## 実装内容
1. `internal/ui/components/tool_approval.go`の作成:
   ```go
   type ToolApprovalDialog struct {
       tool        string
       parameters  map[string]interface{}
       risks       []string
       choices     []Choice
       selected    int
       styles      Styles
   }
   
   type Choice struct {
       Label   string
       Value   ApprovalResponse
       Default bool
   }
   ```

2. 承認ダイアログ:
   ```
   ┌─ Tool Execution Request ─────────────────────┐
   │ Tool: write_file                             │
   │ File: /home/user/project/main.go            │
   │                                             │
   │ This operation will:                        │
   │ • Create or overwrite the file              │
   │ • Write 125 lines of code                   │
   │                                             │
   │ Preview:                                    │
   │ ┌─────────────────────────────────────┐    │
   │ │ package main                        │    │
   │ │ import "fmt"                        │    │
   │ │ ...                                 │    │
   │ └─────────────────────────────────────┘    │
   │                                             │
   │ [Y]es  [N]o  [A]lways  [V]iew  [?]Help     │
   └─────────────────────────────────────────────┘
   ```

3. リスク表示:
   - 操作の影響範囲
   - 潜在的なリスク
   - 取り消し可能性
   - 推奨事項

4. プレビュー機能:
   - ファイル変更の差分
   - 実行コマンドの表示
   - 影響を受けるファイル一覧

5. 承認オプション:
   - Yes（一回限り）
   - No（拒否）
   - Always（セッション中常に許可）
   - Never（セッション中常に拒否）

## 完了条件
- [ ] 承認ダイアログが明確
- [ ] 操作内容が理解しやすい
- [ ] キーボード操作が快適
- [ ] 選択が記憶される

## 依存関係
- task-034-approval-system
- task-051-ui-styles

## 推定作業時間
2時間