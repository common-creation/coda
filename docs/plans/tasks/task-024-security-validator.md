# Task 024: セキュリティバリデーターの実装

## 概要
ファイルアクセスの安全性を検証し、危険な操作を防ぐセキュリティシステムを実装する。

## 実装内容
1. `internal/security/validator.go`の作成:
   ```go
   type SecurityValidator interface {
       ValidatePath(path string) error
       ValidateOperation(op Operation, path string) error
       IsAllowedExtension(path string) bool
       CheckContent(content []byte) error
   }
   
   type DefaultValidator struct {
       workingDir    string
       allowedPaths  []string
       deniedPaths   []string
       maxFileSize   int64
   }
   ```

2. パス検証機能:
   - 作業ディレクトリ外へのアクセス防止
   - シンボリックリンクの解決と検証
   - 相対パストラバーサル攻撃の防止
   - システムファイルへのアクセス制限

3. 操作検証:
   - 読み取り/書き込み/実行権限のチェック
   - 危険なファイル拡張子のブロック
   - 隠しファイルへのアクセス制御
   - ルートディレクトリ操作の防止

4. コンテンツ検証:
   - 実行可能ファイルの検出
   - スクリプトインジェクションの防止
   - ファイルサイズ制限
   - エンコーディング検証

## 完了条件
- [ ] パストラバーサル攻撃が防げる
- [ ] システムファイルが保護される
- [ ] 設定可能な制限が機能する
- [ ] パフォーマンスへの影響が最小限

## 依存関係
- task-017-tool-manager

## 推定作業時間
1.5時間