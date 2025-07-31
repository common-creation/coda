# Task 028: 会話履歴管理の実装

## 概要
チャットの会話履歴を管理し、過去の会話を参照・検索できるシステムを実装する。

## 実装内容
1. `internal/chat/history.go`の作成:
   ```go
   type History struct {
       Sessions    []SessionSummary
       TotalChats  int
       StoragePath string
   }
   
   type SessionSummary struct {
       ID        string
       Title     string
       StartTime time.Time
       EndTime   time.Time
       Messages  int
       Tags      []string
   }
   ```

2. 履歴管理機能:
   - Save(session *Session): 履歴保存
   - Load(id string): 履歴読み込み
   - Search(query string): 履歴検索
   - Delete(id string): 履歴削除
   - Export(format string): エクスポート

3. 自動タイトル生成:
   - 最初の数メッセージから要約
   - AIを使用した要約（オプション）
   - カスタムタイトル設定

4. 検索機能:
   - キーワード検索
   - 日付範囲検索
   - タグによるフィルタリング
   - 正規表現サポート

5. ストレージ形式:
   - JSON形式での保存
   - 圧縮オプション
   - 暗号化オプション（将来）

## 完了条件
- [ ] 履歴の保存と読み込みが正常動作
- [ ] 検索機能が効率的に動作
- [ ] 大量の履歴でもパフォーマンス良好
- [ ] データの整合性が保たれる

## 依存関係
- task-027-session-management

## 推定作業時間
1.5時間