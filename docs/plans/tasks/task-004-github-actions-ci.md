# Task 004: GitHub Actions CI/CDの設定

## 概要
継続的インテグレーション/デプロイメントのためのGitHub Actionsワークフローを設定する。

## 実装内容
1. `.github/workflows/ci.yml`の作成:
   - プッシュ/プルリクエスト時のトリガー
   - Go環境のセットアップ（複数バージョンのマトリックス）
   - 依存関係のキャッシュ
   - ビルドの実行
   - テストの実行（カバレッジレポート付き）
   - リントの実行

2. `.github/workflows/release.yml`の作成:
   - タグプッシュ時のトリガー
   - マルチプラットフォームビルド（Linux, macOS, Windows）
   - GoReleaserを使用したリリース
   - バイナリのアップロード

## 完了条件
- [ ] CIワークフローが正常に動作する
- [ ] プルリクエストでステータスチェックが表示される
- [ ] リリースワークフローがドラフトで作成されている

## 依存関係
- task-001-project-initialization
- task-003-makefile-creation

## 推定作業時間
45分