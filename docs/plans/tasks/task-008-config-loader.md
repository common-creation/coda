# Task 008: YAML設定ローダーの実装

## 概要
YAMLファイルから設定を読み込むローダーを実装し、設定の永続化と管理を可能にする。

## 実装内容
1. `internal/config/loader.go`の作成:
   - Load(path string) (*Config, error)関数
   - Save(path string, config *Config) error関数
   - 設定ファイルパスの解決ロジック

2. 設定ファイル検索順序の実装:
   1. コマンドライン引数で指定されたパス
   2. 環境変数 CODA_CONFIG_PATH
   3. カレントディレクトリの.coda/config.yaml
   4. ホームディレクトリの~/.config/coda/config.yaml

3. 設定マージ機能:
   - 複数の設定源からのマージ
   - 環境変数による上書き（CODA_AI_API_KEY等）

4. サンプル設定ファイルの作成:
   - `config.example.yaml`

## 完了条件
- [ ] YAMLファイルの読み書きが正常に動作する
- [ ] 設定ファイルの検索順序が正しく実装されている
- [ ] 環境変数による上書きが機能する
- [ ] エラーハンドリングが適切に実装されている
- [ ] 単体テストでカバレッジ80%以上

## 依存関係
- task-007-config-structure

## 推定作業時間
1時間