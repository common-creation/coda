# Task 078: バイナリビルドスクリプトの作成

## 概要
複数プラットフォーム向けのバイナリを効率的にビルドするスクリプトを作成する。

## 実装内容
1. `scripts/build.sh`の作成:
   ```bash
   #!/bin/bash
   set -e
   
   VERSION=${VERSION:-$(git describe --tags --always --dirty)}
   BUILD_DATE=$(date -u +"%Y-%m-%dT%H:%M:%SZ")
   COMMIT=$(git rev-parse HEAD)
   
   LDFLAGS="-s -w \
     -X main.Version=${VERSION} \
     -X main.Commit=${COMMIT} \
     -X main.Date=${BUILD_DATE}"
   
   # Build for multiple platforms
   PLATFORMS=(
     "darwin/amd64"
     "darwin/arm64"
     "linux/amd64"
     "linux/arm64"
     "windows/amd64"
   )
   ```

2. クロスコンパイル設定:
   ```bash
   for PLATFORM in "${PLATFORMS[@]}"; do
     GOOS=${PLATFORM%/*}
     GOARCH=${PLATFORM#*/}
     OUTPUT="dist/coda-${VERSION}-${GOOS}-${GOARCH}"
     
     if [ "$GOOS" = "windows" ]; then
       OUTPUT="${OUTPUT}.exe"
     fi
     
     echo "Building for $PLATFORM..."
     GOOS=$GOOS GOARCH=$GOARCH go build \
       -ldflags "$LDFLAGS" \
       -o "$OUTPUT" \
       ./cmd/coda
   done
   ```

3. 最適化オプション:
   - バイナリサイズ削減（-s -w）
   - 実行速度最適化
   - CGO無効化（可能な場合）
   - UPX圧縮（オプション）

4. ビルド後処理:
   - チェックサム生成
   - 署名（macOS/Windows）
   - アーカイブ作成
   - リリースノート生成

5. CI/CD統合:
   ```yaml
   # .github/workflows/release.yml
   - name: Build binaries
     run: |
       make release-build
       
   - name: Upload artifacts
     uses: actions/upload-artifact@v3
     with:
       name: binaries
       path: dist/
   ```

## 完了条件
- [ ] 全対象プラットフォームでビルド成功
- [ ] バイナリサイズが最適化されている
- [ ] バージョン情報が埋め込まれる
- [ ] CI/CDで自動実行される

## 依存関係
- task-004-github-actions-ci
- task-041-version-command

## 推定作業時間
1.5時間