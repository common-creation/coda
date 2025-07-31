# Task 072: インストールガイドの作成

## 概要
様々な環境でのインストール方法を詳細に説明するガイドを作成する。

## 実装内容
1. `docs/INSTALL.md`の作成:
   ```markdown
   # Installation Guide
   
   ## System Requirements
   - Go 1.21+ (for building from source)
   - Terminal with 256 color support
   - Internet connection
   
   ## Quick Install
   
   ### macOS
   ```bash
   brew tap common-creation/coda
   brew install coda
   ```
   
   ### Linux
   ```bash
   curl -sSL https://get.coda.dev | bash
   ```
   
   ### Windows
   ```powershell
   scoop bucket add coda https://github.com/common-creation/scoop-coda
   scoop install coda
   ```
   ```

2. 詳細インストール手順:
   - プリビルトバイナリ
   - パッケージマネージャー
   - Docker イメージ
   - ソースからのビルド

3. 環境別の注意事項:
   - WSL2での使用
   - SSH経由での使用
   - コンテナ内での実行
   - 企業プロキシ環境

4. 初期設定ガイド:
   ```bash
   # 設定ファイルの初期化
   coda config init
   
   # APIキーの設定
   coda config set-api-key openai
   
   # 動作確認
   coda chat -m "Hello, CODA!"
   ```

5. トラブルシューティング:
   - 権限エラー
   - パスの設定
   - 依存関係の問題
   - ネットワーク関連

## 完了条件
- [ ] 主要OSがカバーされている
- [ ] 手順が明確で分かりやすい
- [ ] エラー時の対処法が記載
- [ ] 動作確認方法が明確

## 依存関係
- なし

## 推定作業時間
1時間