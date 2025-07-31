# Task 079: リリースノート作成

## 概要
バージョンリリース時に公開するリリースノートのテンプレートと自動生成の仕組みを作成する。

## 実装内容
1. `scripts/generate-release-notes.sh`の作成:
   ```bash
   #!/bin/bash
   
   # Get version and previous tag
   VERSION=$1
   PREV_TAG=$(git describe --tags --abbrev=0 HEAD^)
   
   # Generate changelog
   echo "# Release Notes for v${VERSION}"
   echo
   echo "## What's New"
   git log ${PREV_TAG}..HEAD --grep="feat:" --pretty="- %s"
   
   echo
   echo "## Bug Fixes"
   git log ${PREV_TAG}..HEAD --grep="fix:" --pretty="- %s"
   
   echo
   echo "## Breaking Changes"
   git log ${PREV_TAG}..HEAD --grep="BREAKING CHANGE" --pretty="- %s"
   ```

2. リリースノートテンプレート:
   ```markdown
   # CODA v1.0.0
   
   Released: 2024-XX-XX
   
   ## 🎉 Highlights
   - Major feature 1
   - Major feature 2
   
   ## ✨ New Features
   - Feature description (#PR)
   
   ## 🐛 Bug Fixes
   - Fix description (#PR)
   
   ## 🔧 Improvements
   - Performance improvements
   - UI enhancements
   
   ## 📝 Documentation
   - Updated guides
   - New examples
   
   ## ⚠️ Breaking Changes
   - Change description
   - Migration guide
   
   ## 📦 Dependencies
   - Updated dependency to vX.X.X
   
   ## Contributors
   Thanks to all contributors!
   ```

3. 自動生成要素:
   - コミットメッセージからの抽出
   - PR/Issue リンク
   - コントリビューター一覧
   - 依存関係の変更
   - ダウンロードリンク

4. 多言語対応:
   - 英語版（デフォルト）
   - 日本語版
   - 自動翻訳の準備

5. 配布チャネル:
   - GitHub Releases
   - プロジェクトWebサイト
   - パッケージマネージャー
   - SNS告知用サマリー

## 完了条件
- [ ] リリースノートが自動生成される
- [ ] 内容が分かりやすく整理されている
- [ ] 重要な変更が強調されている
- [ ] 移行ガイドが含まれている

## 依存関係
- task-078-build-scripts

## 推定作業時間
1時間