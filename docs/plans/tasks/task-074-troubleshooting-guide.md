# Task 074: トラブルシューティングガイドの作成

## 概要
ユーザーが遭遇する可能性のある問題と解決方法をまとめたトラブルシューティングガイドを作成する。

## 実装内容
1. `docs/TROUBLESHOOTING.md`の構成:
   ```markdown
   # Troubleshooting Guide
   
   ## Common Issues
   
   ### Installation Problems
   #### "command not found"
   **Problem**: After installation, `coda` command is not recognized
   **Solution**: 
   1. Check if binary is in PATH
   2. Restart terminal
   3. Manually add to PATH
   
   ### Connection Issues
   #### "Failed to connect to AI service"
   **Possible Causes**:
   - Network connectivity
   - Firewall blocking
   - Invalid API key
   
   ### Performance Issues
   ### UI Problems
   ### Tool Execution Errors
   ```

2. エラーメッセージ索引:
   ```markdown
   ## Error Message Reference
   
   | Error Message | Cause | Solution |
   |---------------|-------|----------|
   | "API key not found" | Missing configuration | Run `coda config set-api-key` |
   | "Rate limit exceeded" | Too many requests | Wait or upgrade plan |
   | "Permission denied" | File access rights | Check file permissions |
   ```

3. デバッグ手順:
   - ログファイルの場所
   - デバッグモードの使用
   - 診断コマンド
   - 情報収集スクリプト

4. FAQ形式:
   - Q: CODAが遅い
   - Q: ファイルが編集できない
   - Q: セッションが消えた
   - Q: UIが崩れる

5. 問題報告テンプレート:
   ```markdown
   ## Reporting Issues
   
   When reporting an issue, please include:
   - CODA version (`coda version`)
   - OS and terminal info
   - Error messages
   - Steps to reproduce
   - Debug logs (if available)
   ```

## 完了条件
- [ ] 一般的な問題がカバーされている
- [ ] 解決手順が明確
- [ ] 検索しやすい構成
- [ ] 問題報告方法が明確

## 依存関係
- task-067-user-friendly-errors

## 推定作業時間
1.5時間