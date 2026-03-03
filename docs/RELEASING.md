# Releasing

このドキュメントは、`v1.0.0` 以降のリリース手順をまとめます。

## 事前チェック

1. `main` が最新であることを確認
2. ローカルで以下を実行して成功を確認
   - `go test ./... -race`
   - `staticcheck ./...`
3. README と CHANGELOG が更新されていることを確認

## リリース作成手順（GitHub）

1. バージョンタグを作成して push

```bash
git checkout main
git pull origin main
git tag -a v1.0.0 -m "v1.0.0"
git push origin v1.0.0
```

2. GitHub の Releases で `Draft a new release`
3. Tag に `v1.0.0` を選択
4. Title を `v1.0.0` に設定
5. リリースノートに CHANGELOG の `v1.0.0` を反映
6. `Publish release`

## リリース後確認

1. Release ページにタグ/ノートが表示されること
2. CI が green であること
3. README の Quickstart で起動できること（Go または Docker）

