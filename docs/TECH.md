# TECH

このドキュメントは、実装全体を技術者向けに俯瞰する入口です。詳細は各ドキュメントへリンクしています。

## 1. アーキテクチャ（adapter / app / domain）

- `internal/domain`: 純粋ロジック（マッチメイク、補正計算、設定値解釈）
- `internal/app/usecase`: コマンド単位の業務フロー（入出力調停、状態遷移）
- `internal/adapter`: Discord表示とSQLite永続化

詳細: [ARCHITECTURE.md](ARCHITECTURE.md)

## 2. アルゴリズム

- 8〜10人入力から 4v4 を最適化
- 10人時の探索上限: `10C8 × (8C4 / 2) = 1575`
- 指標は `Diff = abs(sumA - sumB)` 最小
- 全探索なので最適解保証

詳細: [DECISIONS.md](DECISIONS.md)

## 3. 永続化（SQLite + migrations）

- RoomState / player_stats / matches / room_settings をSQLiteへ保存
- 起動時に `migrations/*.sql` を自動適用
- 適用履歴は `schema_migrations` で管理

## 4. 運用対策

- room単位ロックで同時更新競合を回避
- `/make` `/next` にクールダウンを適用
- 長めの処理は defer 応答でInteraction制約に対応
- `/diagnose`（管理者）で lock/cooldown/room状態を可視化

## 5. 品質

- ユニット/統合テスト: `go test ./... -race`
- CI: gofmt, staticcheck, dependency direction check, go test -race
- ベンチ（手動実行）:
  - domain最悪ケースマッチメイク
  - sqlite save/load

## 6. 設定

環境変数:

- `DISCORD_TOKEN`
- `DISCORD_APP_ID`
- `DISCORD_GUILD_ID`
- `SQLITE_PATH`（未指定時 `./data.db`）

room単位設定（`/settings`）:

- `make_next_cooldown_seconds`
- `spectator_rotation_weight`
- `same_team_avoidance_weight`
- `pause_default_matches`

## 関連ドキュメント

- [ARCHITECTURE.md](ARCHITECTURE.md)
- [DECISIONS.md](DECISIONS.md)
- [RELEASING.md](RELEASING.md)
- [../CHANGELOG.md](../CHANGELOG.md)

