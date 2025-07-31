# Task 070: トレーシング機能の実装

## 概要
処理の流れを詳細に追跡し、問題の原因を特定するためのトレーシング機能を実装する。

## 実装内容
1. `internal/debug/tracing.go`の作成:
   ```go
   type Tracer struct {
       spans      []Span
       activeSpan *Span
       sampler    Sampler
       exporter   TraceExporter
   }
   
   type Span struct {
       TraceID   string
       SpanID    string
       ParentID  string
       Name      string
       StartTime time.Time
       EndTime   time.Time
       Tags      map[string]interface{}
       Events    []Event
   }
   ```

2. トレース対象:
   - API呼び出し（リクエスト→レスポンス）
   - ツール実行（開始→終了）
   - ファイル操作
   - UI イベント処理
   - エラー発生箇所

3. 分散トレーシング:
   ```go
   // チャット処理のトレース例
   span := tracer.StartSpan("chat.HandleMessage")
   defer span.End()
   
   // AIクライアント呼び出し
   aiSpan := span.StartChild("ai.ChatCompletion")
   response, err := client.ChatCompletion(ctx, req)
   aiSpan.End()
   
   // ツール実行
   toolSpan := span.StartChild("tools.Execute")
   result := executor.Execute(toolCall)
   toolSpan.End()
   ```

4. トレースビジュアライゼーション:
   ```
   ┌─ Trace: chat.HandleMessage (1234ms) ─────┐
   │ ├─ validate.Input (2ms)                  │
   │ ├─ session.AddMessage (5ms)              │
   │ ├─ ai.ChatCompletion (987ms)            │
   │ │  ├─ http.Request (980ms)              │
   │ │  └─ parse.Response (7ms)              │
   │ ├─ tools.Detect (15ms)                   │
   │ └─ tools.Execute (225ms)                │
   │    ├─ security.Validate (10ms)          │
   │    ├─ approval.Request (150ms)          │
   │    └─ file.Write (65ms)                 │
   └──────────────────────────────────────────┘
   ```

5. サンプリング戦略:
   - 全トレース（デバッグ時）
   - エラー時のみ
   - 確率的サンプリング
   - 遅い処理のみ

6. エクスポート形式:
   - JSON Lines
   - OpenTelemetry形式
   - カスタムフォーマット

## 完了条件
- [ ] 処理フローが可視化される
- [ ] パフォーマンスボトルネックが特定可能
- [ ] エラーの原因が追跡可能
- [ ] オーバーヘッドが最小限

## 依存関係
- task-069-debug-enhancements

## 推定作業時間
2時間