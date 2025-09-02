# Product Requirements Document - CODA (CODing Agent)

## 1. プロジェクト概要

CODA（CODing Agent）は、Goで実装されたCLIベースのコーディングアシスタントです。OpenAI（Azure OpenAI対応）のLLMを活用し、ファイルの読み書き、コード編集、プロジェクト管理などの開発タスクを支援します。Bubbleteaフレームワークを使用した洗練されたターミナルインターフェースを提供します。

## 2. 目的と目標

### 主要目的
- 開発者の生産性向上
- コーディングタスクの自動化
- 対話的なコード編集体験の提供

### 目標
- シンプルかつ強力なツールセットの提供
- 高い拡張性とカスタマイズ性
- エレガントなユーザーインターフェース
- OpenAIとAzure OpenAIの両方をサポート

## 3. 主要機能

### 3.1 コア機能
- **対話型チャット**: LLMとの自然な会話によるコーディング支援
- **ファイル操作**: ファイルの読み取り、書き込み、一覧表示
- **コード編集**: 既存ファイルの修正、新規ファイルの作成
- **プロジェクト理解**: プロジェクト構造の把握と分析
- **マルチモデル対応**: OpenAIとAzure OpenAIの切り替え可能

### 3.2 ツール機能
1. **read_file**: ファイル内容の読み取り
2. **write_file**: 新規ファイルの作成
3. **edit_file**: 既存ファイルの編集
4. **list_files**: ディレクトリ内容の一覧表示
5. **search_files**: ファイル内容の検索
6. **run_command**: シェルコマンドの実行（オプション）

### 3.3 UI機能
- **リッチなターミナルUI**: Bubbleteaによる対話的インターフェース
- **シンタックスハイライト**: コードの可読性向上
- **プログレス表示**: 長時間処理の進捗状況表示
- **ヒストリー管理**: 過去の会話履歴の参照

## 4. 技術要件

### 4.1 開発言語とフレームワーク
- **言語**: Go 1.21以上
- **UI**: github.com/charmbracelet/bubbletea
- **AIクライアント**: 
  - github.com/openai/openai-go (公式OpenAIクライアント)
  - Azure OpenAI統合対応

### 4.2 依存関係
```yaml
dependencies:
  - bubbletea: "^0.25.0"
  - openai-go: "^1.0.0"  # 公式OpenAIクライアント
  - viper: "^1.18.0"     # 設定管理
  - cobra: "^1.8.0"      # CLIフレームワーク
```

### 4.3 設定管理
YAML形式の設定ファイルによる柔軟な設定:
```yaml
# config.yaml
ai:
  provider: "openai" # or "azure"
  
  openai:
    api_key: "${OPENAI_API_KEY}"
    model: "o3"
    
  azure:
    endpoint: "${AZURE_ENDPOINT}"
    api_key: "${AZURE_API_KEY}"
    deployment_name: "o3"
    api_version: "2024-02-01"

ui:
  theme: "dark"
  syntax_highlighting: true
  
tools:
  enabled:
    - read_file
    - write_file
    - edit_file
    - list_files
    - search_files
```

## 5. アーキテクチャ

### 5.1 レイヤー構造
```
┌─────────────────────────────────┐
│         UI Layer (Bubbletea)    │
├─────────────────────────────────┤
│      Application Layer          │
│  (Commands, Handlers, State)    │
├─────────────────────────────────┤
│        Service Layer            │
│  (AI Client, Tool Manager)      │
├─────────────────────────────────┤
│      Infrastructure Layer       │
│  (Config, File System, HTTP)    │
└─────────────────────────────────┘
```

### 5.2 主要コンポーネント
1. **AIClient**: OpenAI/Azure OpenAIとの通信を抽象化
2. **ToolManager**: ツールの登録と実行を管理
3. **ConversationManager**: 会話履歴と状態管理
4. **UIController**: Bubbletea UIの制御
5. **ConfigManager**: 設定ファイルの読み込みと管理

## 6. ユーザーインターフェース

### 6.1 画面レイアウト
```
┌─────────────────────────────────────────┐
│  CODA - CODing Agent          [Model: gpt-4] │
├─────────────────────────────────────────┤
│                                         │
│  Chat History                          │
│  ────────────                          │
│  User: Help me create a new function   │
│  CODA: I'll help you create...         │
│                                         │
├─────────────────────────────────────────┤
│  Input:                                 │
│  > _                                    │
└─────────────────────────────────────────┘
```

### 6.2 キーバインド
- `Ctrl+C`: 終了
- `Ctrl+L`: 画面クリア
- `Ctrl+R`: 履歴検索
- `Tab`: オートコンプリート

## 7. セキュリティとプライバシー

- APIキーの安全な管理（環境変数、暗号化）
- ローカルファイルアクセスの制限設定
- 会話履歴の暗号化保存（オプション）

## 8. パフォーマンス要件

- 応答時間: < 100ms（UI操作）
- メモリ使用量: < 100MB（通常使用時）
- 並行処理: 複数ツールの同時実行対応

## 9. 今後の拡張性

- プラグインシステムによるツール拡張
- 複数LLMプロバイダーの追加サポート
- Webインターフェースの追加
- チーム共有機能

## 10. リリース計画

### Phase 1 (MVP)
- 基本的な会話機能
- ファイル操作ツール（read, write, edit, list）
- OpenAI統合
- 基本的なBubbletea UI

### Phase 2
- Azure OpenAI対応
- 高度なUI機能（シンタックスハイライト、プログレス表示）
- 設定ファイル管理
- エラーハンドリングの改善

### Phase 3
- プラグインシステム
- 追加ツール（検索、コマンド実行）
- パフォーマンス最適化
- ドキュメント整備