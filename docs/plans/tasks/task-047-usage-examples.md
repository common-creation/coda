# Task 047: 使用例の追加

## 概要
各コマンドに実践的な使用例を追加し、ユーザーの理解を助ける。

## 実装内容
1. コマンド別使用例:
   ```go
   chatCmd.Example = `  # 通常のチャットセッション開始
   coda chat
   
   # 特定のモデルを使用
   coda chat --model gpt-4
   
   # 単一の質問を送信
   coda chat -m "このプロジェクトの構造を説明して"
   
   # 前回のセッションを継続
   coda chat --continue`
   ```

2. 設定コマンドの例:
   ```go
   configCmd.Example = `  # 現在の設定を表示
   coda config show
   
   # APIキーを設定
   coda config set-api-key openai
   
   # 特定の値を設定
   coda config set ai.model gpt-4
   
   # 設定ファイルを初期化
   coda config init`
   ```

3. 高度な使用例:
   - パイプラインでの使用
   - スクリプトとの統合
   - CI/CDでの活用
   - カスタム設定の利用

4. シナリオ別ガイド:
   - コードレビュー
   - バグ修正
   - リファクタリング
   - ドキュメント生成

5. ベストプラクティス:
   - 効果的なプロンプト
   - ツールの活用方法
   - セッション管理
   - セキュリティ考慮事項

## 完了条件
- [ ] 各コマンドに3つ以上の例がある
- [ ] 例が実践的で有用
- [ ] コピー&ペーストで動作する
- [ ] 一般的なユースケースをカバー

## 依存関係
- task-046-help-documentation

## 推定作業時間
45分