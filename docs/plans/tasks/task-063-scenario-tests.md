# Task 063: シナリオベーステストの作成

## 概要
実際のユースケースに基づいた統合テストシナリオを作成する。

## 実装内容
1. テストシナリオ定義:
   ```go
   type TestScenario struct {
       Name        string
       Description string
       Steps       []TestStep
       Expected    []Assertion
   }
   
   type TestStep struct {
       Action   string
       Input    interface{}
       Wait     time.Duration
   }
   ```

2. 実用的なシナリオ:
   - **コードレビューシナリオ**:
     1. プロジェクトディレクトリを開く
     2. 「このコードをレビューして」と入力
     3. ファイル読み取りの承認
     4. レビュー結果の確認

   - **バグ修正シナリオ**:
     1. エラーメッセージを貼り付け
     2. 関連ファイルの特定
     3. 修正案の生成
     4. ファイル編集の実行

   - **リファクタリングシナリオ**:
     1. 改善したい関数を指定
     2. リファクタリング提案
     3. 変更の承認と実行
     4. テスト実行

3. アサーション:
   - 出力内容の検証
   - ファイル変更の確認
   - エラーの有無
   - パフォーマンス基準

4. データ駆動テスト:
   ```go
   scenarios := []TestScenario{
       LoadScenario("testdata/review_scenario.yaml"),
       LoadScenario("testdata/debug_scenario.yaml"),
   }
   ```

5. レポート生成:
   - 成功/失敗の詳細
   - 実行時間
   - スクリーンキャプチャ
   - デバッグログ

## 完了条件
- [ ] 10以上の実用シナリオ
- [ ] 各シナリオが独立実行可能
- [ ] 詳細なレポート生成
- [ ] メンテナンスが容易

## 依存関係
- task-062-e2e-tests

## 推定作業時間
2.5時間