# Task 025: セキュリティ危険パターンの定義

## 概要
セキュリティ上危険とされるファイルパターンやコンテンツパターンを定義し、検出可能にする。

## 実装内容
1. `internal/security/patterns.go`の作成:
   ```go
   type SecurityPatterns struct {
       DangerousPaths      []string
       DangerousExtensions []string
       DangerousContent    []regexp.Regexp
       SystemPaths         []string
   }
   ```

2. 危険なパスパターン:
   - `/etc/*` - システム設定ファイル
   - `/usr/bin/*` - システムバイナリ
   - `~/.ssh/*` - SSH鍵
   - `**/.git/config` - Git認証情報
   - 環境変数ファイル（.env, .envrc）

3. 危険な拡張子:
   - 実行可能ファイル（.exe, .sh, .bat）
   - システムライブラリ（.so, .dll）
   - 設定ファイル（.conf, .ini）
   - 証明書ファイル（.pem, .key）

4. 危険なコンテンツパターン:
   - シェルコマンドインジェクション
   - SQLインジェクション
   - 認証情報のパターン
   - Base64エンコードされた実行可能ファイル

5. プラットフォーム別定義:
   - Windows固有のパス
   - Linux/Unix固有のパス
   - macOS固有のパス

## 完了条件
- [ ] 主要な危険パターンが定義されている
- [ ] パターンマッチングが効率的
- [ ] 誤検出が最小限
- [ ] 簡単に拡張可能な構造

## 依存関係
- task-024-security-validator

## 推定作業時間
1時間