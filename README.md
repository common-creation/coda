# CODA - CODing Assistant

![](https://i.imgur.com/stKKmbT.png)

[English](README.en.md) | 日本語

<div align="center">

[![Go Version](https://img.shields.io/badge/go-%3E%3D1.24-blue.svg)](https://go.dev/)
[![License](https://img.shields.io/badge/license-MIT-blue.svg)](LICENSE)
[![CI Status](https://github.com/common-creation/coda/workflows/CI/badge.svg)](https://github.com/common-creation/coda/actions)

自然言語での対話を通じて、コードの作成、理解、管理を支援するインテリジェントなコマンドラインコーディングアシスタント。

</div>

## 機能

- 🤖 **マルチモデルサポート**: OpenAI GPTおよびAzure OpenAIモデルに対応
- 💬 **インタラクティブチャット**: コーディングタスク用の自然言語インターフェース
- 🛠️ **ツール統合**: ビルトインファイル操作（読み取り、書き込み、編集、検索 ...）
- 🔒 **セキュリティファースト**: 意図せぬツールコールを避ける承認システム
- 📝 **コンテキスト認識**: プロジェクト構造と依存関係を理解
- 🎨 **リッチターミナルUI**: Bubbletea搭載の美しいインターフェース
- 🔧 **設定可能**: YAML/環境変数による広範な設定オプション

## クイックスタート

### インストール

#### リリースから

https://github.com/common-creation/coda/releases/latest

#### Goを使用

```bash
go install github.com/common-creation/coda@latest
```

#### ソースから

```bash
git clone https://github.com/common-creation/coda.git
cd coda
make build
```

### 設定

1. 設定を初期化:
```bash
coda config init
```

または

```bash
coda
# 初回起動 または APIキー未設定 の場合は設定ファイルが作成される
```

2. APIキーを設定:
```bash
# OpenAI用
coda config set-api-key openai [key]

# Azure OpenAI用
coda config set-api-key azure [key]
```

設定ファイルを直接編集してもかまいません。

3. (オプション) 設定をカスタマイズ:
```bash
coda config set ai.model o4-mini # デフォルトは o3
```

### 基本的な使い方

#### インタラクティブチャットモード

```bash
# インタラクティブチャットを開始
coda

# 特定のモデルを使用
coda --model o4-mini
```

## コマンド

### `coda` または `coda chat`
AIアシスタントとのインタラクティブチャットセッションを開始します。

**オプション:**
- `--model`: 使用するAIモデルを指定

### `coda config`
CODA設定を管理します。

**サブコマンド:**
- `show`: 現在の設定を表示
- `set KEY VALUE`: 設定値を設定
- `get KEY`: 特定の値を取得
- `init`: 設定ファイルを初期化
- `validate`: 設定の妥当性をチェック
- `set-api-key PROVIDER`: APIキーをhistoryに残さずに設定

### `coda version`
バージョン情報を表示します。

**オプション:**
- `--verbose, -v`: 詳細なバージョン情報を表示
- `--json`: JSON形式で出力

## 設定

CODAは以下の場所から設定を読み込みます（順番に）:
1. コマンドラインフラグ: `--config`
2. 環境変数: `CODA_CONFIG`
3. `$HOME/.coda/config.yaml`
4. `./config.yaml`

### 設定ファイルの例

```yaml
ai:
  provider: openai  # または "azure"
  model: o3
  temperature: 1
  max_tokens: 0 # 0を指定すると制限しない

tools:
  enabled: true
  auto_approve: false
  allowed_paths:
    - "."
  denied_paths:
    - "/etc"
    - "/sys"

session:
  max_history: 100
  max_tokens: 8000
  persistence: true

logging:
  level: info
  file: ~/.coda/coda.log
```

### 環境変数

すべての設定オプションは環境変数で設定できます:

```bash
export CODA_AI_PROVIDER=openai
export CODA_AI_MODEL=o4-mini
export CODA_AI_API_KEY=sk-...
```

## ワークスペース設定

CODAはワークスペースファイルを通じてプロジェクト固有の設定をサポートします:

### `.coda/CODA.md` または `.claude/CLAUDE.md` (実験的)

```markdown
# プロジェクト指示

これはNext.js 14を使用したReact TypeScriptプロジェクトです。

## ルール
- 常にTypeScript strictモードを使用
- フックを使った関数コンポーネントを優先
- プロジェクトのESLint設定に従う

## コンテキスト
- メインAPIエンドポイント: /api/v1
- データベース: Prisma ORMを使用したPostgreSQL
- 認証: NextAuth.js
```

## 利用可能なツール

CODAにはファイル操作用のビルトインツールがいくつか含まれています。
以下はその代表例です:

- **read_file**: ファイルの内容を読み取る
- **write_file**: ファイルを作成または上書き
- **edit_file**: ファイルの特定部分を変更
- **list_files**: ディレクトリの内容を一覧表示
- **search_files**: 内容や名前でファイルを検索

セキュリティのため、すべてのツール操作はデフォルトでユーザーの承認が必要です。

## セキュリティ

CODAは複数のセキュリティ対策を実装しています:

- **制限されたファイルアクセス**: 操作は許可されたパスに制限
- **承認システム**: ツール呼び出しには明示的なユーザーの同意が必要
- **パス検証**: ディレクトリトラバーサル攻撃を防止

## 開発

### 前提条件

- Go 1.24以上

### プロジェクト構造

```
coda/
├── cmd/           # CLIコマンド
├── internal/      # 内部パッケージ
│   ├── ai/       # AIクライアント実装
│   ├── chat/     # チャット処理ロジック
│   ├── config/   # 設定管理
│   ├── security/ # セキュリティ検証
│   ├── tools/    # ツール実装
│   └── ui/       # ターミナルUI (Bubbletea)
├── docs/         # ドキュメント
├── scripts/      # ビルドおよびユーティリティスクリプト
└── tests/        # テスト
```

### 開発ワークフロー

1. リポジトリをフォーク
2. 機能ブランチを作成 (`git checkout -b feature/amazing-feature`)
3. 変更をコミット (`git commit -m 'Add some amazing feature'`)
4. ブランチにプッシュ (`git push origin feature/amazing-feature`)
5. プルリクエストを開く

## トラブルシューティング

### よくある問題

**APIキーが見つからない**
```bash
# APIキーが設定されているか確認
coda config get ai.api_key

# APIキーを再設定
coda config set-api-key openai
```

**ファイル操作の権限拒否**
- `allowed_paths`設定を確認
- 適切なファイルシステム権限があることを確認

**接続タイムアウト**
- インターネット接続を確認
- プロキシの背後にいるか確認

### デバッグモード

詳細なログ記録のためデバッグモードを有効化:

```bash
coda --debug chat
```

ログは`~/.coda/coda.log`で確認できます

## ライセンス

MIT License

## 謝辞

- CLI用[Cobra](https://github.com/spf13/cobra)で構築
- [Bubbletea](https://github.com/charmbracelet/bubbletea)搭載のUI
- OpenAIおよびAzure OpenAI API経由のAI統合

## ロードマップ

- [x] 基本的なチャット機能
- [x] ファイル操作ツール
- [x] マルチモデルサポート
- [x] 設定管理
- [x] リッチターミナルUI
- [ ] ツール追加
- [ ] プラグインシステム
- [ ] ローカルモデルサポート

---

<div align="center">
Made with ❤️ by the CODA team
</div>