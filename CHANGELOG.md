# Changelog

このプロジェクトの主要変更を記録します。

## v1.0.0

- Discord Bot の最小起動（`/ping`）を実装
- 参加者管理コマンド（`/join`, `/leave`, `/list`）を実装
- 全探索マッチメイク（8〜10人、4v4 + 観戦、seed決定性）を実装
- `make` / `next` / `reroll` / `undo` の試合進行フローを実装
- `pause` / `resume` / `paused` とリアクション復帰（👍）を実装
- `result` による勝敗登録、履歴保存、rating_delta補正を実装
- `whoami` / `help` / `settings` / `export` を実装
- 出力を Embed 中心に統一し表示整形を集約
- SQLite 永続化（RoomState・履歴・統計・設定）を実装
- Docker マルチステージビルドと永続化運用を整備
- CI 強化（gofmt / staticcheck / go test -race / 依存方向チェック）
- レイヤー分離（adapter/app/domain）と設計ドキュメントを整備
- SQLite マイグレーション管理（`migrations/` + `schema_migrations`）を導入

