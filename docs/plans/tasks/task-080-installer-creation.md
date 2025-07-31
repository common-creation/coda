# Task 080: インストーラー作成

## 概要
各プラットフォーム向けの使いやすいインストーラーを作成する。

## 実装内容
1. インストールスクリプト:
   ```bash
   # scripts/install.sh
   #!/bin/bash
   
   CODA_VERSION="${CODA_VERSION:-latest}"
   INSTALL_DIR="${INSTALL_DIR:-/usr/local/bin}"
   
   detect_platform() {
       OS=$(uname -s | tr '[:upper:]' '[:lower:]')
       ARCH=$(uname -m)
       
       case "$ARCH" in
           x86_64) ARCH="amd64" ;;
           aarch64|arm64) ARCH="arm64" ;;
       esac
       
       echo "${OS}-${ARCH}"
   }
   
   download_binary() {
       PLATFORM=$(detect_platform)
       URL="https://github.com/common-creation/coda/releases/download/${VERSION}/coda-${PLATFORM}"
       
       echo "Downloading CODA ${VERSION} for ${PLATFORM}..."
       curl -sSL "$URL" -o "$TEMP_FILE"
   }
   ```

2. Homebrewフォーミュラ:
   ```ruby
   # Formula/coda.rb
   class Coda < Formula
     desc "AI-powered coding assistant"
     homepage "https://github.com/common-creation/coda"
     version "1.0.0"
     
     if OS.mac? && Hardware::CPU.arm?
       url "https://github.com/.../coda-darwin-arm64"
       sha256 "..."
     elsif OS.mac?
       url "https://github.com/.../coda-darwin-amd64"
       sha256 "..."
     end
     
     def install
       bin.install "coda"
     end
   end
   ```

3. Windowsインストーラー:
   - MSIパッケージ作成
   - Scoopマニフェスト
   - Chocolateyパッケージ
   - PowerShellスクリプト

4. パッケージマネージャー対応:
   - apt/yum リポジトリ
   - Snap パッケージ
   - Flatpak
   - Docker イメージ

5. インストール後設定:
   ```bash
   post_install() {
       # Create config directory
       mkdir -p ~/.config/coda
       
       # Shell completion
       coda completion bash > /etc/bash_completion.d/coda
       
       # First run message
       echo "CODA installed successfully!"
       echo "Run 'coda config init' to get started"
   }
   ```

## 完了条件
- [ ] ワンライナーでインストール可能
- [ ] 主要パッケージマネージャー対応
- [ ] アンインストールが清潔
- [ ] 自動アップデート機能

## 依存関係
- task-078-build-scripts
- task-072-installation-guide

## 推定作業時間
2時間