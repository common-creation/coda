# Task 029: セッション永続化の実装

## 概要
アプリケーション再起動時でもセッション状態を維持するための永続化機能を実装する。

## 実装内容
1. `internal/chat/persistence.go`の作成:
   ```go
   type Persistence interface {
       SaveSession(session *Session) error
       LoadSession(id string) (*Session, error)
       ListSessions() ([]string, error)
       DeleteSession(id string) error
   }
   
   type FilePersistence struct {
       basePath string
       encoder  Encoder
   }
   ```

2. 保存戦略:
   - 自動保存（一定間隔）
   - 明示的な保存
   - クラッシュ時の復旧
   - トランザクション的な書き込み

3. データ形式:
   - メタデータとメッセージの分離
   - インデックスファイルの管理
   - チャンク化（大規模セッション対応）

4. 復旧機能:
   - 破損したセッションの検出
   - 部分的な復旧
   - バックアップからの復元
   - 整合性チェック

5. マイグレーション:
   - バージョン管理
   - スキーマ変更への対応
   - 後方互換性の維持

## 完了条件
- [ ] セッションが確実に永続化される
- [ ] アプリ再起動後も継続可能
- [ ] データ破損に対する耐性
- [ ] 効率的な読み書き性能

## 依存関係
- task-027-session-management
- task-028-history-management

## 推定作業時間
1.5時間