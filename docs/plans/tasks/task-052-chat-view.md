# Task 052: チャットビューの実装

## 概要
メッセージの表示とスクロールを管理するチャットビューを実装する。

## 実装内容
1. `internal/ui/views/chat_view.go`の作成:
   ```go
   type ChatView struct {
       messages     []Message
       viewport     viewport.Model
       scrollOffset int
       width        int
       height       int
       styles       Styles
   }
   
   type Message struct {
       Role      string
       Content   string
       Timestamp time.Time
       IsError   bool
   }
   ```

2. メッセージ表示:
   - ロール別のスタイリング（User/Assistant/System）
   - タイムスタンプ表示
   - マークダウンレンダリング
   - コードブロックの整形

3. スクロール機能:
   - ビューポート管理
   - 自動スクロール（新規メッセージ時）
   - マニュアルスクロール
   - ページ単位のスクロール

4. メッセージフォーマット:
   ```
   ┌─ User (10:30:45) ─────────────┐
   │ ファイルを読んでください      │
   └───────────────────────────────┘
   
   ┌─ Assistant ───────────────────┐
   │ ファイルの内容は以下です:     │
   │ ```python                     │
   │ def hello():                  │
   │     print("Hello")            │
   │ ```                           │
   └───────────────────────────────┘
   ```

5. パフォーマンス最適化:
   - 仮想スクロール
   - 遅延レンダリング
   - メッセージのキャッシュ

## 完了条件
- [ ] メッセージが正しく表示される
- [ ] スクロールがスムーズ
- [ ] マークダウンが整形される
- [ ] 大量メッセージでも高速

## 依存関係
- task-049-ui-model
- task-051-ui-styles

## 推定作業時間
2時間