# Task 009: APIキー管理システムの実装

## 概要
APIキーなどの機密情報を安全に管理するためのシステムを実装する。

## 実装内容
1. `internal/config/secrets.go`の作成:
   - キーチェーン/資格情報マネージャーとの連携
   - 環境変数からの読み込み
   - セキュアなメモリ管理

2. プラットフォーム別実装:
   - macOS: Keychainアクセス
   - Linux: Secret Service API / ファイルベース（暗号化）
   - Windows: Credential Manager

3. フォールバック機構:
   - システムキーストアが使用できない場合のファイルベース保存
   - 権限設定（600）による保護
   - 警告メッセージの表示

4. APIキー操作インターフェース:
   ```go
   type SecretsManager interface {
       GetAPIKey(provider string) (string, error)
       SetAPIKey(provider string, key string) error
       DeleteAPIKey(provider string) error
       ListProviders() ([]string, error)
   }
   ```

## 完了条件
- [ ] 各プラットフォームでAPIキーが安全に保存される
- [ ] 環境変数からの読み込みが優先される
- [ ] エラーハンドリングが適切
- [ ] 単体テスト（モック使用）が作成されている

## 依存関係
- task-007-config-structure

## 推定作業時間
1.5時間