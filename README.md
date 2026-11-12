# sf-meter-api

SF 作品に出てくるテクノロジーが、いまどれくらい現実に近いか。それを 0〜100 の「実現度」で返す API です。タイムマシンは 1、量子暗号通信は 88、といった具合です。

Go の標準ライブラリ（`net/http`）だけで、層を分けて依存の向きをそろえる練習として作っています。外部のパッケージは PostgreSQL のドライバー（`lib/pq`）1 つだけです。

- 画面: [sf-meter-web](https://github.com/nemdull/sf-meter-web)
- 全体の入り口: [sf-meter](https://github.com/nemdull/sf-meter)
- 作った人: [nemdull.com](https://www.nemdull.com/)

## すぐ動かす

DB は要りません。`DATABASE_URL` が無ければ、初期データ 15 件を持ったメモリ実装で動きます。

```bash
go run .
curl "http://localhost:8080/api/v1/techs?sort=-score&limit=3"
```

```json
[
  { "id": 2, "name": "シビラシステム", "description": "脳スキャンで犯罪係数を算出する全国犯罪予防システム", "score": 91, "level": "reality" },
  { "id": 8, "name": "量子暗号通信", "description": "解読不可能な量子もつれを利用した秘匿通信", "score": 88, "level": "reality" },
  { "id": 3, "name": "ニューラルリンク", "description": "脳とコンピュータを直接接続するBCIデバイス", "score": 82, "level": "reality" }
]
```

PostgreSQL と一緒に動かすとき:

```bash
cp .env.example .env            # 必要なら編集
docker compose up --build       # db（PostgreSQL 16）→ app の順で起動
```

DB だけ Docker で動かして、アプリは手元の Go で動かすなら `make run-local`。

## API

くわしい仕様は [docs/openapi.yaml](docs/openapi.yaml) にあります。

| メソッド | パス | 何をするか | トークン |
| --- | --- | --- | --- |
| GET | `/api/v1/techs` | 一覧。絞り込み・並べ替え・ページ分けができる | 不要 |
| GET | `/api/v1/techs/{id}` | 1 件 | 不要 |
| GET | `/api/v1/techs/random` | どれか 1 件 | 不要 |
| GET | `/api/v1/stats` | 件数・平均・段階ごとの内訳 | 不要 |
| POST | `/api/v1/techs` | 1 件足す | 要る |
| PUT | `/api/v1/techs/{id}` | 1 件を置き換える | 要る |
| DELETE | `/api/v1/techs/{id}` | 1 件消す | 要る |
| GET | `/health` | サーバーと保存先が動いているか | 不要 |

`/api/v1` を付ける前のパス（`GET /techs`、`GET /techs/{id}`）も、読み取りだけ残してあります。

### 一覧のパラメーター

| 名前 | 意味 | 既定値 |
| --- | --- | --- |
| `q` | 名前か説明に含まれる文字（英字の大文字・小文字は区別しない） | なし |
| `min_score` / `max_score` | 実現度の範囲（どちらも含む） | 0 / 100 |
| `sort` | `id`（登録順）、`score`（低い順）、`-score`（高い順）、`name`（名前順） | `id` |
| `limit` | 返す件数（1〜100） | 50 |
| `offset` | 先頭から飛ばす件数 | 0 |

本文は配列のままです。絞り込み後の総件数は、`X-Total-Count` ヘッダーに入ります。

```bash
curl -i "http://localhost:8080/api/v1/techs?q=通信&min_score=70"
```

### level（段階）

`score` を 3 段階に丸めた値を、`level` として一緒に返します。

| level | score | 意味 |
| --- | --- | --- |
| `fiction` | 0〜29 | まだ SF |
| `emerging` | 30〜69 | 研究が進んでいる |
| `reality` | 70〜100 | ほぼ現実 |

### 書き込み

追加・更新・削除は、**サーバーに `ADMIN_TOKEN` を設定したときだけ**使えます。設定していなければ、どんなリクエストにも 403（`writes_disabled`）を返します。うっかり公開しても、誰かに書き換えられないようにするためです。

```bash
ADMIN_TOKEN="$(openssl rand -hex 32)" go run .

curl -X POST http://localhost:8080/api/v1/techs \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"name":"軌道エレベーター","description":"宇宙まで届く昇降機","score":20}'
```

### エラー

どのエラーも同じ形で返します。利用する側は、文言ではなく `code` を見て分岐できます。

```json
{ "error": { "code": "validation_failed", "message": "score 101 is out of range [0, 100]", "field": "score" } }
```

| code | ステータス | いつ |
| --- | --- | --- |
| `invalid_parameter` | 400 | クエリやパスの値がおかしい |
| `invalid_body` | 400 | 本文が JSON でない、知らない項目がある、大きすぎる |
| `validation_failed` | 400 | 名前が空、実現度が範囲の外、など |
| `unauthorized` | 401 | トークンが無い、または違う |
| `writes_disabled` | 403 | サーバーに `ADMIN_TOKEN` が設定されていない |
| `not_found` | 404 | 指定した ID やパスが無い |
| `method_not_allowed` | 405 | そのパスでは使えないメソッド |
| `internal_error` | 500 | サーバーの中の問題（中身は外に出さず、記録にだけ残します） |

## 環境変数

| 変数 | 説明 | 既定値 |
| --- | --- | --- |
| `DATABASE_URL` | PostgreSQL の接続文字列。未設定ならメモリ実装 | （なし） |
| `PORT` | 待ち受けポート | `8080` |
| `ADMIN_TOKEN` | 書き込みに使うトークン。未設定なら書き込みを閉じる | （なし） |
| `CORS_ALLOWED_ORIGINS` | ブラウザーから呼んでよいオリジン（カンマ区切り。`*` でどこからでも） | `http://localhost:5173` と `http://127.0.0.1:5173` |

`DATABASE_URL` を設定したのに DB につながらないときは、メモリ実装に切り替えたりせず、起動に失敗します。気づかないまま別の保存先で動いてしまうのを避けるためです。

## 構成

```
sf-meter-api/
├── domain/              # Tech と、その不変条件（Validate）・段階（Level）・エラーの種類
├── usecase/             # TechService と、usecase が必要とするポート（TechRepository・TechQuery）
├── infrastructure/      # リポジトリの実装: memory.go（メモリ）/ postgres.go（PostgreSQL）/ seed.go（初期データ）
│   └── repotest/        # どの実装にも同じように当てるテスト
├── presentation/        # ルーター、ハンドラー、DTO、共通の処理（記録・CORS・トークンの確認）
├── registry/            # 依存の組み立て（DI）。DATABASE_URL の有無で実装を切り替える
├── db/init.sql          # テーブルの定義と初期データ（PostgreSQL の起動時に自動で入る）
├── docs/openapi.yaml    # API の仕様
├── main.go              # 起動と、安全な停止
├── Dockerfile           # マルチステージビルド（distroless）
├── docker-compose.yml   # PostgreSQL 16 + app
└── Makefile             # build / test / lint / run など（make help で一覧）
```

依存の向きは、外側から内側への一方通行です。

```
presentation ──▶ usecase ──▶ domain
                    ▲
infrastructure ─────┘        （usecase が決めたポートを、infrastructure が実装する）

registry は全部を知っていて、起動のときに組み立てる
```

- `domain` は、ほかのどの層も import しません
- `usecase` は、保存先が何かを知りません。`TechRepository` というポート（インターフェース）だけを見ています
- `presentation` も、`TechService` そのものではなく、自分の側で決めた `TechUseCase` を見ています。テストで差し替えるためです
- 「見つからない」は `domain.ErrNotFound`、入力の誤りは `*domain.ValidationError` で返し、ハンドラーが 404 と 400 に変換します

## テスト

```bash
go test -race ./...        # PostgreSQL のテストは飛ばす
make test-db               # DB のコンテナを起こして、PostgreSQL のテストも走らせる
```

| 場所 | 見ていること |
| --- | --- |
| `domain/tech_test.go` | 不変条件の境界（名前の長さ、実現度の範囲）、段階の区切り |
| `usecase/tech_service_test.go` | 入力を確かめてから保存すること、集計、`Random` の選び方（乱数を差し替えて固定） |
| `infrastructure/repotest/` | 絞り込み・並び順・件数・見つからないときのエラー。**メモリ実装と PostgreSQL 実装の両方に、同じテストを当てる** |
| `infrastructure/seed_test.go` | メモリ実装の初期データと `db/init.sql` が、同じ内容であること |
| `presentation/handler_test.go` | ルーターからメモリ実装までをつないで、パス・ステータス・JSON の形を確かめる |
| `presentation/router_test.go` | 404 / 405、CORS、トークン、panic からの復帰、中のエラーを外に出さないこと、**実装のパスと `docs/openapi.yaml` がずれていないこと** |

保存先を差し替えても結果が変わらない、というのがこの構成の売りなので、そこはテストで確かめています。PostgreSQL のテストは、手元では `TEST_DATABASE_URL` があるときだけ走り、CI では毎回走ります。

## CI

プルリクエストと main への push で、GitHub Actions が次を確かめます（[.github/workflows/ci.yml](.github/workflows/ci.yml)）。

1. `gofmt` がかかっているか、`go mod tidy` で差分が出ないか、`go vet`
2. PostgreSQL 16 を立てて、`go test -race`（PostgreSQL のテストが飛ばされていないことも確かめる）
3. バイナリとコンテナのビルド。コンテナを起動して `/health` に答えること

## 初期データ

メモリ実装（`infrastructure/seed.go`）と PostgreSQL（`db/init.sql`）で、同じ 15 件が入ります。変えるときは両方を直してください。片方だけ直すと、テストが落ちて教えてくれます。

## 関連する記事

- [Go言語で気づいたシンプル設計の真髄 ─ Rails・Java経験者がハマったポイントまとめ](https://qiita.com/nemdull/items/7ff73572ccc3c78d311d)

## ライセンス

[MIT License](LICENSE)
