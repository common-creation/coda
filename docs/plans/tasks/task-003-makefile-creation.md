# Task 003: Makefileの作成

## 概要
開発作業を効率化するためのMakefileを作成し、ビルド・テスト・リントのタスクを定義する。

## 実装内容
1. Makefileの作成（以下のターゲットを含む）:
   - `build`: バイナリのビルド
   - `test`: 単体テストの実行
   - `test-coverage`: カバレッジ付きテスト
   - `lint`: golangci-lintの実行
   - `fmt`: go fmtの実行
   - `clean`: ビルド成果物のクリーンアップ
   - `install`: バイナリのインストール
   - `run`: 開発用実行

2. 変数の定義:
   - BINARY_NAME
   - BUILD_DIR
   - GO_FILES
   - VERSION（git tagから取得）

## 完了条件
- [ ] Makefileが作成されている
- [ ] 全てのターゲットが正しく動作する
- [ ] `make help`でターゲット一覧が表示される

## 依存関係
- task-001-project-initialization

## 推定作業時間
30分