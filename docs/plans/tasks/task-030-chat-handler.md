# Task 030: メインチャットハンドラーの実装

## 概要
ユーザー入力を受け取り、AIとのやり取りを制御するメインのチャットハンドラーを実装する。

## 実装内容
1. `internal/chat/handler.go`の作成:
   ```go
   type ChatHandler struct {
       aiClient    ai.Client
       toolManager *tools.Manager
       session     *SessionManager
       config      *Config
   }
   
   func (h *ChatHandler) HandleMessage(ctx context.Context, input string) error
   ```

2. メッセージ処理フロー:
   1. 入力の前処理（トリミング、検証）
   2. セッションへのメッセージ追加
   3. システムプロンプトの構築
   4. AIへのリクエスト送信
   5. レスポンスの処理
   6. ツールコールの検出と実行
   7. 結果の表示

3. エラーハンドリング:
   - API接続エラー
   - タイムアウト
   - 無効な入力
   - ツール実行エラー

4. コンテキスト管理:
   - 会話履歴の管理
   - システムプロンプトの更新
   - ワーキングディレクトリの追跡

5. 特殊コマンド:
   - /clear - セッションクリア
   - /save - セッション保存
   - /load - セッション読み込み
   - /help - ヘルプ表示

## 完了条件
- [ ] 基本的な会話が成立する
- [ ] エラーが適切に処理される
- [ ] ツール実行が統合されている
- [ ] 特殊コマンドが動作する

## 依存関係
- task-010-ai-client-interface
- task-017-tool-manager
- task-027-session-management

## 推定作業時間
2時間