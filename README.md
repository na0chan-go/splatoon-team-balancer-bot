![CI](https://github.com/na0chan-go/splatoon-team-balancer-bot/actions/workflows/ci.yml/badge.svg)

# Splatoon3 プライベートマッチ チーム分けBot

Splatoon3 のプライベートマッチでチーム分けを自動化する Discord Bot です。
参加者が **Xパワーを申告して参加**すると、Botが **チームの合計パワー差が最小になるように自動で4v4のチーム分け**を行います。

10人参加した場合は、8人をプレイヤーとして選び、残り2人を観戦に回します。

手動でのチーム分けの手間や「戦力差による不公平」を減らすことを目的としています。

## ポイント

- **10C8 × (8C4 / 2) = 1575** の全探索で最適解を保証
- **永続化（SQLite）** により再起動後も部屋状態を復元
- **決定性（seed）** により結果の再現が可能
- **観戦ローテーション** により、同点/僅差では直近観戦者が連続しにくい

## /make Embed デモ

![make-embed-demo](docs/images/make-embed-demo.svg)

---

# 主な機能

- Xパワーを入力してマッチに参加
- 最大10人まで参加可能
- 8人プレイ + 2人観戦に自動調整
- 合計Xパワー差が最小になるチーム分け
- 全探索アルゴリズムによる最適解保証
- 同条件で再計算可能（reroll）
- 観戦ローテーション（spectator_count / last_spectated_at を利用）
- match history tracking（/result で勝敗を保存）
- dynamic power adjustment（declared_xpower + rating でマッチング）
- 連打対策（room単位ロック + /make,/next の3秒クールダウン）

---

# 使用例

参加者が以下のように登録します。

/join 2400
/join 2350
/join 2300
/join 2250
/join 2200
/join 2150
/join 2100
/join 2050
/join 2000
/join 1950

チーム分けコマンドを実行

/make

Botの出力例

Alphaチーム（合計: 9000）

- PlayerA (2400)
- PlayerD (2250)
- PlayerF (2150)
- PlayerH (2050)

Bravoチーム（合計: 9000）

- PlayerB (2350)
- PlayerC (2300)
- PlayerE (2200)
- PlayerG (2100)

観戦

- PlayerI (2000)
- PlayerJ (1950)

パワー差: 0

---

# アルゴリズム

このBotでは **全探索アルゴリズム**を使用し、チームの戦力差が最も小さくなる組み合わせを求めます。

処理の流れ

1. 参加者が8人以上の場合、全ての **8人の組み合わせ** を生成
2. それぞれの8人について **4人 vs 4人のチーム分け** を生成
3. チームの合計Xパワー差を計算
4. 差が最小になる組み合わせを採用

最大探索数

10C8 × (8C4 / 2) = 1575 通り

この規模であれば数ミリ秒以内で計算可能です。

---

# アーキテクチャ

Discord
↓
Slash Command Handler
↓
Matchmaking Engine
↓
Team Result Formatter

チーム分けアルゴリズムは Discord の処理から分離されており、
`internal/domain` に純粋なビジネスロジックとして実装されています。

---

# ディレクトリ構成

cmd/bot/
Botのエントリーポイント

internal/bot/
Discordコマンド処理

internal/domain/
チーム分けアルゴリズム

internal/store/
状態管理（メモリ / DB）

internal/util/
補助関数

---

# 技術スタック

言語
Go

主要ライブラリ

- discordgo

テスト

- go test

---

# 起動方法

リポジトリをクローン

GitHubから取得

必要な環境変数を設定

- `DISCORD_TOKEN`: Bot token（`Bot ` プレフィックスは不要）
- `DISCORD_APP_ID`: Discord Application ID
- `DISCORD_GUILD_ID`: コマンドを登録するGuild ID（設定時はguild登録のみ、未設定時はglobal登録のみ）
- `SQLITE_PATH`: SQLite DBファイルパス（省略時 `./data.db`）

開発中は `DISCORD_GUILD_ID` の設定を推奨します（guild command は反映が速い）。

```bash
export DISCORD_TOKEN=your_token
export DISCORD_APP_ID=123456789012345678
export DISCORD_GUILD_ID=123456789012345678
export SQLITE_PATH=./data.db
```

Botを起動

```bash
go run cmd/bot/main.go
```

起動後、指定Guildで `/ping` を実行すると `pong` が返ります。

---

# Docker起動

イメージをビルド

```bash
docker build -t splatoon-team-balancer-bot .
```

コンテナを起動

```bash
docker run --rm \
  -e DISCORD_TOKEN=your_token \
  -e DISCORD_APP_ID=123456789012345678 \
  -e DISCORD_GUILD_ID=123456789012345678 \
  -e SQLITE_PATH=/app/data.db \
  -v "$(pwd)/data:/app" \
  splatoon-team-balancer-bot
```

補足

- 実行バイナリはコンテナ内 `/app/bot`
- `SQLITE_PATH` 未指定時は `./data.db`（= `/app/data.db`）

---

# コマンド一覧

/join <xpower>
マッチに参加

/leave
マッチから退出

/list
現在の参加者一覧

/help
最短フローと運用例を表示（Embed）

/make
チーム分けを実行

/next
現在の参加者で次試合を作成

/pause
数試合だけ一時離脱（トイレ・離席など）

/resume
一時離脱を解除

/paused
一時離脱中の一覧を表示

/whoami
自分の状態（pause残り・参加回数・観戦回数・XPower）を表示

/undo
直前の /make または /next の状態に戻す

/reroll
別の最適解を再計算

/result <alpha|bravo>
試合結果を記録してratingを更新

/reset
部屋を初期化

---

# 運用例（一時離脱）

例: 3試合だけ離席したい場合

```text
/pause matches:3 reason:トイレ
```

この状態で `/next` が実行されるたびに残り試合数が1ずつ減り、0になると自動復帰します。  
途中で戻る場合は `/resume` を実行してください。

また、`/pause` 実行時にBotが投稿する案内メッセージへ対象ユーザーが 👍 リアクションすると、pauseを即時解除できます。

---

# 初回オンボーディング

その部屋で最初に `/join` されたタイミングで、使い方Embedを1回だけ表示します。  
2回目以降の `/join` ではオンボーディングは表示されず、通常の参加/更新メッセージのみ返します。

---

# テスト

すべてのテストを実行

go test ./...

テスト内容

- 8人の場合に均等なチームが作れること
- 10人の場合に観戦2人が選ばれること
- 同じ乱数シードで結果が再現されること

---

# 今後の改善案

- 観戦ローテーション機能
- 同じチームが連続しない仕組み
- SQLiteによる永続化
- マッチ履歴の保存
- Web UI

---

# このプロジェクトの背景

Splatoonのプライベートマッチでは、
プレイヤーの実力差による不公平を防ぐために
手動でチーム分けをすることが多いです。

しかし、手作業での調整は時間がかかり、
公平なチーム分けを作るのも難しい問題があります。

このBotは、その問題を **アルゴリズムで自動解決する**ことを目的に作られています。

---

# ライセンス

[MIT](./LICENSE)
