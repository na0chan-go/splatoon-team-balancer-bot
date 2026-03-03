# Architecture

このドキュメントは、Splatoon Team Balancer Bot のレイヤー構成と主要なデータフローをまとめたものです。

## レイヤー構成

### `internal/domain`

- 純粋ロジック層
- 役割:
  - チーム分け（全探索で Diff 最小化）
  - 観戦ローテーションの評価
  - レート補正の計算ルール
- 制約:
  - Discord SDK や SQLite への直接依存を持たない
  - 時刻・乱数は呼び出し側（usecase）から受け取る

### `internal/app/usecase`

- ユースケース層
- 役割:
  - コマンド単位の業務フローを実行
  - RoomState の読み込み、ドメイン処理呼び出し、保存
  - seed/time の生成、権限制御、クールダウン/ロック制御
- 依存:
  - `domain`
  - `store` の interface
  - `adapter` には interface 越しで接続

### `internal/adapter`

- 外部I/Oアダプタ層
- `adapter/discord`:
  - Embed生成
  - Discord Interaction 応答の整形
- `adapter/store`:
  - SQLite 実装
  - SQL・テーブル操作を集約

## 依存方向

- `domain` -> (`app`, `adapter`) への依存は禁止
- `app` -> `adapter/discord` への直接依存は禁止
- CI の `scripts/check-dependency-rules.sh` で検証

## 主要データフロー

### `/join -> /make -> /next -> /result`

1. `/join`
   - 呼び出しユーザーを RoomState.Players に追加/更新
   - XPower はバリデーションして保存
   - 初回オンボーディング表示フラグを更新
2. `/make`
   - 現在の参加者（8〜10人）を対象にマッチメイク
   - seed を生成して domain に渡す
   - LastResult / LastSeed / Snapshot を保存
3. `/next`
   - 既存参加者を再利用して次試合を作成
   - pause 中プレイヤーを除外
   - 観戦ローテーション・同チーム回避重みを設定値から反映
4. `/result`
   - 直近 LastResult に対して勝敗を登録
   - player_stats（wins/losses/rating_delta）を更新
   - matches に履歴を保存

## 永続化の概略

SQLite に room 単位で状態を保存し、再起動後も運用を継続可能にしています。

- RoomState 系:
  - `guild_id + channel_id` をキーに保存
  - players
  - pause 状態
  - LastResult / LastSeed / LastPlayersSnapshot
  - onboarding フラグ、設定、履歴など
- マッチ履歴:
  - `matches` テーブルに試合結果を保存
- プレイヤー統計:
  - `player_stats` テーブルに rating_delta / wins / losses / last_played_at を保存

