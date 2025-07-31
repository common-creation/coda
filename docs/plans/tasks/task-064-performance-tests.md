# Task 064: パフォーマンステストの作成

## 概要
アプリケーションの性能を測定し、ボトルネックを特定するテストを作成する。

## 実装内容
1. `tests/performance/`の構造:
   ```go
   type PerformanceTest struct {
       Name       string
       Setup      func()
       Benchmark  func(*testing.B)
       Threshold  PerformanceThreshold
       Cleanup    func()
   }
   
   type PerformanceThreshold struct {
       MaxDuration  time.Duration
       MaxMemory    int64
       MaxGoroutines int
   }
   ```

2. ベンチマーク項目:
   - **起動時間**:
     - コールドスタート
     - 設定読み込み
     - 初期化完了まで

   - **レスポンス時間**:
     - キー入力から表示まで
     - API呼び出しの遅延
     - ツール実行時間

   - **メモリ使用量**:
     - アイドル時
     - 大量メッセージ時
     - 長時間実行時

   - **スループット**:
     - メッセージ処理速度
     - ファイル操作速度
     - 並行処理能力

3. 負荷テスト:
   ```go
   func BenchmarkConcurrentMessages(b *testing.B) {
       // 複数セッションの同時実行
       // ストレステスト
       // リソース競合の検証
   }
   ```

4. プロファイリング:
   - CPU プロファイル
   - メモリプロファイル
   - ブロッキングプロファイル
   - トレース生成

5. パフォーマンス監視:
   - 継続的なベンチマーク
   - 性能劣化の検出
   - グラフ生成
   - アラート設定

## 完了条件
- [ ] 主要な操作がベンチマーク対象
- [ ] 性能基準が明確
- [ ] プロファイル取得が自動化
- [ ] CI/CDでの継続監視

## 依存関係
- task-062-e2e-tests

## 推定作業時間
2時間