# Task 011: AI共通型定義

## 概要
AIクライアントで使用する共通の型を定義し、OpenAI APIとの互換性を保ちながら拡張性を確保する。

## 実装内容
1. `internal/ai/types.go`の作成:
   ```go
   // チャットメッセージ
   type Message struct {
       Role      string      `json:"role"`
       Content   string      `json:"content"`
       ToolCalls []ToolCall  `json:"tool_calls,omitempty"`
   }
   
   // チャットリクエスト
   type ChatRequest struct {
       Model       string    `json:"model"`
       Messages    []Message `json:"messages"`
       Temperature float32   `json:"temperature,omitempty"`
       MaxTokens   int       `json:"max_tokens,omitempty"`
       Tools       []Tool    `json:"tools,omitempty"`
   }
   
   // ツール定義
   type Tool struct {
       Type     string       `json:"type"`
       Function FunctionDef  `json:"function"`
   }
   ```

2. ストリーミング関連の型:
   ```go
   type StreamReader interface {
       Read() (StreamChunk, error)
       Close() error
   }
   
   type StreamChunk struct {
       Delta   string
       Finish  bool
       Error   error
   }
   ```

3. 定数の定義:
   - ロール名（system, user, assistant, tool）
   - モデル名の定数
   - デフォルト値

## 完了条件
- [ ] 全ての必要な型が定義されている
- [ ] JSONタグが適切に設定されている
- [ ] 型の使用例がコメントに含まれている
- [ ] OpenAI APIとの互換性が保たれている

## 依存関係
- task-010-ai-client-interface

## 推定作業時間
45分