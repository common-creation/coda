# Task 069: デバッグモードの充実

## 概要
開発者やパワーユーザー向けの高度なデバッグ機能を実装する。

## 実装内容
1. `internal/debug/debug.go`の作成:
   ```go
   type DebugManager struct {
       enabled     bool
       level       DebugLevel
       collectors  []DataCollector
       inspector   StateInspector
       profiler    Profiler
   }
   ```

2. デバッグ情報パネル:
   ```
   ┌─ Debug Info ─────────────────────────────┐
   │ Session: abc123-def456                    │
   │ Uptime: 00:15:32                         │
   │ Memory: 45.2 MB / 100 MB                 │
   │ Goroutines: 12                           │
   │ API Calls: 23 (2 failed)                 │
   │ Avg Response: 234ms                      │
   │                                          │
   │ Last Request:                            │
   │ > POST /v1/chat/completions              │
   │ > Model: gpt-4                           │
   │ > Tokens: 1,234 / 4,096                  │
   │ > Duration: 567ms                        │
   └──────────────────────────────────────────┘
   ```

3. リクエスト/レスポンス記録:
   ```go
   type RequestTrace struct {
       ID        string
       Timestamp time.Time
       Method    string
       URL       string
       Headers   map[string][]string
       Body      []byte
       Response  ResponseTrace
       Error     error
   }
   ```

4. 状態インスペクター:
   - 現在のセッション内容
   - メモリ内のキャッシュ
   - アクティブな goroutine
   - 開いているファイル
   - ネットワーク接続

5. パフォーマンスプロファイラー:
   - 関数実行時間の計測
   - ホットパスの特定
   - メモリアロケーション追跡
   - ブロッキング箇所の検出

6. デバッグコマンド:
   ```
   :debug toggle     - デバッグモード切り替え
   :debug level 3    - デバッグレベル設定
   :debug dump       - 状態ダンプ
   :debug profile    - プロファイル開始
   :debug trace      - トレース有効化
   ```

## 完了条件
- [ ] 詳細なデバッグ情報が取得可能
- [ ] パフォーマンスの問題を特定可能
- [ ] 本番環境では無効化される
- [ ] デバッグ操作が簡単

## 依存関係
- task-068-structured-logging
- task-044-debug-mode

## 推定作業時間
2時間