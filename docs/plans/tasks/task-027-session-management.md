# Task 027: チャットセッション管理の実装

## 概要
チャットの会話状態を管理し、コンテキストを保持するセッション管理システムを実装する。

## 実装内容
1. `internal/chat/session.go`の作成:
   ```go
   type Session struct {
       ID          string
       StartedAt   time.Time
       LastActive  time.Time
       Messages    []Message
       Context     map[string]interface{}
       MaxTokens   int
       TokenCount  int
   }
   
   type SessionManager struct {
       sessions map[string]*Session
       mu       sync.RWMutex
       maxAge   time.Duration
   }
   ```

2. セッション管理機能:
   - NewSession(): 新規セッション作成
   - GetSession(id string): セッション取得
   - UpdateSession(id string, msg Message): メッセージ追加
   - CleanupSessions(): 期限切れセッション削除

3. トークン管理:
   - メッセージ追加時のトークンカウント
   - 上限到達時の古いメッセージ削除
   - コンテキストウィンドウの管理

4. セッションコンテキスト:
   - 現在の作業ディレクトリ
   - ユーザー設定
   - アクティブなツール
   - カスタムメタデータ

## 完了条件
- [ ] セッションの作成と取得が正常動作
- [ ] トークン制限が適切に機能
- [ ] 自動クリーンアップが動作
- [ ] スレッドセーフな実装

## 依存関係
- task-011-ai-types-definition

## 推定作業時間
1時間