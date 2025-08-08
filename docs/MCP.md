# CODA MCP Integration Design

## 概要

このドキュメントは、CODA（CODing Agent）にMCP（Model Context Protocol）サポートを追加するための設計仕様を定義します。MCPはAnthropicが開発したオープンな標準規格で、AIアプリケーションと外部ツール/データソースを統一的に接続するためのプロトコルです。

## 設計目標

1. **互換性**: Claude Code、Cursor、Clineなどの既存MCPクライアントとmcp.json形式で互換性を持つ
2. **非同期性**: MCPサーバーの起動・管理を非同期で行い、メインアプリケーションをブロックしない
3. **拡張性**: 複数のMCPサーバーを同時に管理でき、動的な追加・削除が可能
4. **ユーザビリティ**: MCPサーバーのステータスを簡単に確認できる

## アーキテクチャ

### 全体構成

```
┌─────────────┐     ┌──────────────┐     ┌───────────────┐
│    CODA     │     │ MCP Client   │     │ MCP Servers   │
│   (Main)    │────▶│   Manager    │────▶│ (External)    │
└─────────────┘     └──────────────┘     └───────────────┘
        │                   │                      │
        │                   ├─ Server 1 (stdio)  │
        │                   ├─ Server 2 (HTTP)   │
        └─ Tools Registry   └─ Server 3 (SSE)    │
```

### コンポーネント設計

#### 1. MCP Client Manager (`internal/mcp/`)

主要な責務:
- mcp.json設定ファイルの読み込みと解析
- MCPサーバーの起動・停止・再起動
- サーバーとの通信管理
- ツール/リソース/プロンプトの統合管理

```go
// internal/mcp/types.go
package mcp

type Config struct {
    Servers map[string]ServerConfig `json:"mcpServers"`
}

type ServerConfig struct {
    Command string            `json:"command"`
    Args    []string          `json:"args"`
    Env     map[string]string `json:"env,omitempty"`
    Type    string            `json:"type,omitempty"` // stdio, http, sse
    URL     string            `json:"url,omitempty"`   // for http/sse
}

type Manager interface {
    // 設定の読み込みと検証
    LoadConfig(paths []string) error
    
    // サーバー管理
    StartServer(name string) error
    StopServer(name string) error
    RestartServer(name string) error
    StartAll() error
    StopAll() error
    
    // ステータス管理
    GetServerStatus(name string) ServerStatus
    GetAllStatuses() map[string]ServerStatus
    
    // ツール/リソース管理
    ListTools() ([]ToolInfo, error)
    ListResources() ([]ResourceInfo, error)
    ListPrompts() ([]PromptInfo, error)
    
    // ツール実行
    ExecuteTool(serverName, toolName string, params map[string]interface{}) (interface{}, error)
}

type ServerStatus struct {
    Name        string
    State       State // Starting, Running, Error, Stopped
    Error       error
    StartedAt   time.Time
    Transport   string
    Capabilities ServerCapabilities
}

type State int

const (
    StateStarting State = iota
    StateRunning
    StateError
    StateStopped
)
```

#### 2. Transport層の実装

公式SDKの`github.com/modelcontextprotocol/go-sdk`を使用:

```go
// internal/mcp/transport.go
import (
    "github.com/modelcontextprotocol/go-sdk/mcp"
    "github.com/modelcontextprotocol/go-sdk/jsonrpc"
)

type TransportFactory interface {
    CreateTransport(config ServerConfig) (mcp.Transport, error)
}

// Stdio Transport
type StdioTransport struct {
    cmd      *exec.Cmd
    client   *mcp.Client
    reader   io.ReadCloser
    writer   io.WriteCloser
}

// HTTP/SSE Transport
type HTTPTransport struct {
    url      string
    headers  map[string]string
    client   *mcp.Client
}
```

#### 3. Tool統合 (`internal/tools/`)

既存のToolシステムとMCPツールを統合:

```go
// internal/tools/mcp_tool.go
type MCPTool struct {
    ServerName string
    ToolName   string
    Schema     jsonschema.Schema
    manager    mcp.Manager
}

func (t *MCPTool) Name() string {
    return fmt.Sprintf("mcp_%s_%s", t.ServerName, t.ToolName)
}

func (t *MCPTool) Execute(args map[string]interface{}) (interface{}, error) {
    return t.manager.ExecuteTool(t.ServerName, t.ToolName, args)
}
```

### 設定ファイル形式

#### mcp.json

CODAは以下の優先順位でmcp.jsonを探索:
1. `./.mcp.json` (プロジェクトローカル)
2. `~/.coda/mcp.json` (ユーザーグローバル)
3. `$CODA_CONFIG_DIR/mcp.json` (環境変数指定)

形式はClaude Code/Cursor/Clineと互換:

```json
{
  "mcpServers": {
    "filesystem": {
      "command": "npx",
      "args": ["-y", "@modelcontextprotocol/server-filesystem", "/path/to/files"],
      "env": {
        "LOG_LEVEL": "debug"
      }
    },
    "github": {
      "command": "npx",
      "args": ["-y", "@modelcontextprotocol/server-github"],
      "env": {
        "GITHUB_TOKEN": "${GITHUB_TOKEN}"
      }
    },
    "remote-api": {
      "type": "sse",
      "url": "https://api.example.com/mcp",
      "headers": {
        "Authorization": "Bearer ${API_TOKEN}"
      }
    }
  }
}
```

環境変数の展開をサポート（`${VAR_NAME}`形式）。

### 起動フロー

1. **初期化時**:
   ```go
   // cmd/root.go の初期化処理で
   mcpManager := mcp.NewManager()
   
   // 非同期で設定読み込みとサーバー起動
   go func() {
       if err := mcpManager.LoadConfig(mcpConfigPaths); err != nil {
           log.Error("Failed to load MCP config", "error", err)
           return
       }
       
       if err := mcpManager.StartAll(); err != nil {
           log.Error("Failed to start MCP servers", "error", err)
       }
   }()
   ```

2. **ツール登録**:
   - MCPサーバーが起動したら、利用可能なツールを動的に登録
   - システムプロンプトに自動的に追加

3. **グレースフルシャットダウン**:
   - アプリケーション終了時にすべてのMCPサーバーを停止

### UI統合

#### Input Mode での Ctrl+M

MCPサーバーのステータス表示:

```
MCP Server Status:
─────────────────────────────────────────
• filesystem    [Running]  Started 2m ago
• github        [Running]  Started 2m ago  
• remote-api    [Error]    Connection refused
• databricks    [Starting] Initializing...
─────────────────────────────────────────
Press any key to continue...
```

実装:
```go
// internal/ui/keys.go に追加
case key.Matches(msg, m.keymap.MCPStatus):
    return m.showMCPStatus()

// internal/ui/mcp_status.go
func (m *Model) showMCPStatus() tea.Cmd {
    statuses := m.mcpManager.GetAllStatuses()
    // ステータス表示の生成
}
```

### エラーハンドリング

1. **起動エラー**: サーバーが起動できない場合、エラーをログに記録し、ステータスをErrorに設定
2. **通信エラー**: 再試行ロジックを実装（指数バックオフ）
3. **タイムアウト**: 各操作に適切なタイムアウトを設定

### セキュリティ考慮事項

1. **環境変数**: APIキーなどの機密情報は環境変数経由で渡す
2. **パス検証**: ファイルシステムアクセスは既存のセキュリティ層を通す
3. **コマンド実行**: サーバーコマンドの実行前に検証
4. **ネットワーク**: HTTPSを推奨、証明書検証を有効化

## 実装計画

### フェーズ1: 基盤実装
- [ ] MCP Manager インターフェースと基本実装
- [ ] mcp.json パーサーと設定管理
- [ ] Stdio Transport の実装
- [ ] 基本的なエラーハンドリング

### フェーズ2: ツール統合
- [ ] MCPツールのTool インターフェース実装
- [ ] 動的ツール登録メカニズム
- [ ] システムプロンプトへの自動追加

### フェーズ3: UI統合
- [ ] Ctrl+M でのステータス表示
- [ ] エラー通知の改善
- [ ] プログレス表示

### フェーズ4: 高度な機能
- [ ] HTTP/SSE Transport の実装
- [ ] OAuth認証サポート
- [ ] リソース/プロンプト機能の実装
- [ ] 動的なサーバー追加/削除

## テスト戦略

1. **単体テスト**: 各コンポーネントの個別テスト
2. **統合テスト**: MCPサーバーとの実際の通信テスト
3. **互換性テスト**: Claude Code/Cursor形式のmcp.jsonの読み込みテスト

## 参考資料

- [Model Context Protocol 公式ドキュメント](https://modelcontextprotocol.io/)
- [MCP Go SDK](https://github.com/modelcontextprotocol/go-sdk)
- [Claude Code MCP ドキュメント](https://docs.anthropic.com/en/docs/claude-code/mcp)