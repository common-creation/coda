# Task 036: ワークスペース設定の読み込み

## 概要
CLAUDE.mdやCODA.mdなどのワークスペース固有の設定ファイルを読み込み、AIの動作に反映する。

## 実装内容
1. `internal/chat/workspace.go`の作成:
   ```go
   type WorkspaceConfig struct {
       Instructions string
       Context      map[string]string
       Rules        []string
       Preferences  map[string]interface{}
   }
   
   type WorkspaceLoader struct {
       searchPaths []string
       cache       map[string]*WorkspaceConfig
   }
   ```

2. 設定ファイル検索:
   - カレントディレクトリから上位へ検索
   - .coda/CODA.md
   - .claude/CLAUDE.md
   - プロジェクトルートの設定ファイル

3. 設定の解析:
   - Markdownファイルのパース
   - YAML/JSONブロックの抽出
   - 指示文の抽出
   - メタデータの読み取り

4. 設定の適用:
   - システムプロンプトへの統合
   - AIの振る舞い調整
   - ツールの有効/無効
   - カスタムルールの適用

5. 動的リロード:
   - ファイル変更の監視
   - ホットリロード
   - キャッシュの無効化

## 完了条件
- [ ] 設定ファイルが正しく検出される
- [ ] 設定がAIの動作に反映される
- [ ] ファイル変更が検出される
- [ ] エラー時も安定動作する

## 依存関係
- task-032-prompts-management

## 推定作業時間
1.5時間