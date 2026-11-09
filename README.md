# sf-meter-api
SF作品に登場するテクノロジーの危険度やSF度合いを数値化する「SFメーター」のバックエンド API プロトタイプです。クリーンアーキテクチャを意識したレイヤー構成で、テクノロジーの情報を取得するためのドメインロジックとインフラ層の切り分けを練習することを目的にしています。

## プロジェクト概要
- `DATABASE_URL` を設定すると PostgreSQL、未設定ならメモリ上のモックデータを返す最小限の REST API として動作します。
- `domain` / `usecase` / `infrastructure` / `presentation` / `registry` の各レイヤーを用意し、依存関係逆転のパターンを学ぶ題材になっています。
- `registry` が依存解決（DI）を担い、`main.go` から HTTP サーバーを立ち上げます。

## 技術スタック
- Go 1.24.5
- 標準ライブラリ ＋ PostgreSQL ドライバ `github.com/lib/pq`（HTTP は `net/http` の Go 1.22+ ServeMux パターン）
- PostgreSQL 16（`docker compose` で起動）

## ディレクトリ構成
```
sf-meter-api/
├── domain/              # エンティティ Tech と不変条件（Validate）
├── usecase/             # TechService と、usecase が必要とするポート（TechRepository）
├── infrastructure/      # リポジトリ実装：memory.go（メモリ）/ postgres.go（PostgreSQL）
├── presentation/        # ハンドラー、DTO、ハンドラーのテスト
├── registry/            # 依存解決（DI）。DATABASE_URL の有無で実装を切り替える
├── db/init.sql          # テーブル定義と初期データ（PostgreSQL 起動時に自動投入）
├── main.go              # エントリーポイント（ルーティングと起動）
├── Dockerfile           # マルチステージビルド（distroless）
├── docker-compose.yml   # PostgreSQL 16 + app
├── Makefile             # build / test / lint / run など
├── .env.example         # 環境変数のテンプレート
├── go.mod / go.sum
└── README.md
```

### レイヤー構成と役割
- `domain`: エンティティ (`Tech`) を定義します。
- `usecase`: ビジネスロジック (`TechService`) とポート (`TechRepository`) を提供します。
- `infrastructure`: ユースケース層のポートを満たすリポジトリ実装。現在はメモリ上のデータストアのみ。
- `presentation`: HTTP ハンドラー（`TechHandler`）、DTO、ルーティングを担当します。
- `registry`: 各レイヤーの実装を組み立て、`http.Server` を返します。初期データのシードもここで注入しています。
- `main.go`: `registry.NewHTTPServer(":8080")` を呼び出し、アプリケーションを起動します。

## 起動方法

### 1. メモリ実装で起動（DB 不要）
```bash
go run main.go
curl http://localhost:8080/techs
```
`DATABASE_URL` が未設定のときは、`infrastructure/memory.go` の初期データ（3件）を返します。

### 2. PostgreSQL と一緒に起動
```bash
cp .env.example .env            # 必要なら接続文字列を編集
docker compose up --build       # db（PostgreSQL 16）→ app の順で起動
curl http://localhost:8080/techs
```
`db/init.sql` がテーブル `techs` を作成し、SF テクノロジー 15 件を投入します。`db` には `pg_isready` のヘルスチェックがあり、`app` は DB の準備ができてから起動します。

DB だけ Docker で動かし、アプリはローカルで実行する場合：
```bash
make run-local   # docker compose up -d db → DATABASE_URL を設定して go run
```

### 環境変数
| 変数 | 説明 | 既定値 |
|---|---|---|
| `DATABASE_URL` | PostgreSQL の接続文字列。未設定ならメモリ実装 | （なし） |
| `PORT` | 待ち受けポート | `8080` |

### Makefile
`make help` で一覧が出ます。主なもの：`make build` / `make test` / `make lint`（go vet）/ `make run`（compose）/ `make run-local` / `make docker-build`。

## 事前準備
1. Go 1.24 系をインストールしてください。Homebrew の場合は `brew install go` で導入できます。
2. レポジトリをクローンします。
   ```bash
   git clone git@github.com:nemdull/sf-meter-api.git
   cd sf-meter-api
   ```
3. PostgreSQL を使う場合は Docker（`docker compose`）を用意します。

## 初期データ
メモリ実装（`infrastructure/memory.go`）の初期データは次の 3 件です。PostgreSQL では `db/init.sql` が 15 件を投入します。

| id | name             | description                                   | score |
|----|------------------|-----------------------------------------------|-------|
| 1  | ドミネーター     | 支配射撃体系                                  | 35    |
| 2  | シビラシステム   | 脳スキャンで犯罪係数を算出する予防監視システム | 91    |
| 3  | ニューラルリンク | 脳とコンピュータを直接接続するBCIデバイス      | 82    |

## API 仕様
現在提供しているエンドポイントは次の通りです。

| メソッド | パス                 | 説明                                |
|----------|----------------------|-------------------------------------|
| GET      | `/api/v1/techs`      | テクノロジーの一覧を取得            |
| GET      | `/api/v1/tech?id={id}` | 指定 ID のテクノロジーを取得      |

`GET /api/v1/techs` のレスポンス例:

```json
[
  {
    "id": 1,
    "name": "ドミネーター",
    "description": "犯罪係数で裁く未来銃",
    "score": 35
  }
]
```

`GET /api/v1/tech?id=1` のレスポンス例:

```json
{
  "id": 1,
  "name": "ドミネーター",
  "description": "犯罪係数で裁く未来銃",
  "score": 35
}
```

## データフロー
- クライアントが HTTP リクエストを送信すると、`presentation` 層のルーターがエンドポイントにマッピングされます。
- ハンドラーはユースケース層の `TechService` を呼び出し、必要なビジネスロジックを実行します。
- ユースケースは `infrastructure` 層で実装されたリポジトリ（`MemoryTechRepo` または `PostgresTechRepo`）にアクセスし、データを取得します。「見つからない」は `(nil, nil)` で返し、ハンドラーが 404 に変換します。
- レスポンスは JSON として呼び出し元に返されます。層を明確に分離することで永続化やハンドラー実装の差し替えが容易になります。

## 開発 Tips
- `presentation/dto.go` の DTO を利用することで、ドメイン構造を外部公開のレスポンスから切り離せます。
- `infrastructure/memory.go` の実装はモック用途に絞っています。RDB や外部 API 連携に差し替える場合は、ユースケース層のポートを満たす別実装を追加し、`registry` で差し替えると安全です。
- ルーティングは `presentation` 層に閉じ込め、ハンドラー内では認証・バリデーション・ユースケース呼び出しに責務を分けるとテストしやすくなります。
- 初期データを変更したいときは、メモリ実装は `infrastructure/memory.go` の `seeds`、PostgreSQL は `db/init.sql` を編集します。
- Go のバージョンは `go.mod` に合わせて更新してください。CI を導入する場合は同じバージョンを使い、`go env GOPATH` の設定に留意します。

## テスト
```bash
go test ./...
```
- `usecase/tech_service_test.go`：モックリポジトリで `TechService` を検証（正常系・空・リポジトリエラー、`Validate` の境界値）
- `presentation/handler_test.go`：`httptest` で 200 / 400 / 404 を検証

どちらも DB 接続なしで実行できます。

## 今後のTODO
- 書き込み系 API（POST/PUT/DELETE）。`PostgresTechRepo` には Create/Update/Delete を実装済みだが、ポートとハンドラーには未公開
- `domain` と `usecase` に重複している `TechRepository` インターフェースの整理
- ユースケース層のバリデーションやビジネスルールを拡張
- CI/CD パイプラインの整備

## ライセンス
このプロジェクトは [MIT License](LICENSE) の下で公開されています。
