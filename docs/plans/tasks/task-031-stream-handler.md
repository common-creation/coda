# Task 031: ストリーミング処理の実装

## 概要
AIからのストリーミングレスポンスをリアルタイムで処理し、UIに表示する仕組みを実装する。

## 実装内容
1. `internal/chat/stream.go`の作成:
   ```go
   type StreamHandler struct {
       output      io.Writer
       buffer      *bytes.Buffer
       onChunk     func(string)
       onComplete  func(string)
       onError     func(error)
   }
   
   func (h *StreamHandler) ProcessStream(ctx context.Context, stream ai.StreamReader) error
   ```

2. ストリーム処理機能:
   - チャンク単位での受信
   - バッファリングと表示
   - 部分的なマークダウン処理
   - プログレス表示

3. ツールコール検出:
   - ストリーム中のツールコール識別
   - 部分的なJSON解析
   - ツール実行の準備

4. エラーハンドリング:
   - 接続断の検出
   - 部分的な再送信
   - タイムアウト処理
   - グレースフルな中断

5. パフォーマンス最適化:
   - 効率的なバッファ管理
   - 不要な再描画の防止
   - メモリ使用量の制御

## 完了条件
- [ ] ストリーミングがスムーズに表示される
- [ ] 中断時も適切に処理される
- [ ] ツールコールが正しく検出される
- [ ] メモリリークがない

## 依存関係
- task-011-ai-types-definition
- task-030-chat-handler

## 推定作業時間
1.5時間