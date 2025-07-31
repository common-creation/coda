# Task 005: Pre-commit Hooksの設定

## 概要
コミット前に自動的にコード品質チェックを実行するためのpre-commit hooksを設定する。

## 実装内容
1. `.pre-commit-config.yaml`の作成:
   - go fmt チェック
   - go vet の実行
   - golangci-lint の実行
   - go mod tidy チェック
   - trailing whitespace の除去
   - ファイル末尾の改行確認

2. インストールスクリプトの作成:
   - `scripts/install-hooks.sh`
   - pre-commitツールのインストール確認
   - フックのインストール

3. ドキュメントの更新:
   - README.mdに開発環境セットアップ手順を追加

## 完了条件
- [ ] pre-commit設定ファイルが作成されている
- [ ] コミット時に自動チェックが実行される
- [ ] 不正なコードがコミットされないことを確認

## 依存関係
- task-001-project-initialization
- task-002-gitignore-and-readme

## 推定作業時間
30分