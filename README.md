# sf-meter-api
SF作品に登場するテクノロジーの危険度やSF度合いを数値化する「SFメーター」のバックエンド API プロトタイプです。クリーンアーキテクチャを意識したレイヤー構成で、テクノロジーの情報を取得するためのドメインロジックとインフラ層の切り分けを練習することを目的にしています。

## プロジェクト概要
- 現状はメモリ上のモックデータを返す最小限の API を想定した実装です。
- `domain` / `usecase` / `infrastructure` / `presentation` / `registry` の各レイヤーを明示し、依存関係逆転のパターンを学ぶ題材になっています。
- `registry` が依存解決（DI）を担い、`main.go` から HTTP サーバーを立ち上げます。

## 技術スタック
- Go 1.24.5
- 標準ライブラリのみ（外部依存は今のところありません）

## ディレクトリ構成
```
sf-meter-api/
├── domain/              # エンティティとドメインサービス（例: Tech, TechService）
├── infrastructure/      # ドメインのリポジトリ実装（メモリ実装など）
├── presentation/        # ハンドラーやルーター（※現状は雛形、要実装）
├── registry/            # 依存解決とサーバー初期化
├── main.go              # エントリーポイント
├── go.mod
└── README.md
```

### レイヤー構成と役割
- `domain`: エンティティ (`Tech`) とユースケース境界（将来的に `TechService` など）を定義します。
- `infrastructure`: ドメインで定義したリポジトリインターフェースを実装。現在はメモリ上のデータストアのみ。
- `presentation`: HTTP ハンドラーやルーティングの責務を持つ予定の層です。ルーターは未実装なので、`presentation/router.go` に `package presentation` から始まる実装を追加してください。
- `registry`: 各レイヤーの実装を組み立て、`http.Server` を返します。初期データのシードもここで注入しています。
- `main.go`: `registry.NewHTTPServer(":8080")` を呼び出し、アプリケーションを起動します。

## 事前準備
1. Go 1.24 系をインストールしてください。Homebrew の場合は `brew install go` で導入できます。
2. レポジトリをクローンします。
   ```bash
   git clone git@github.com:nemdull/sf-meter-api.git
   cd sf-meter-api
   ```

## ローカル起動
1. `presentation/router.go` にルーター実装を追加し、最低限 `http.Handler` を返せる状態にします（例: `http.NewServeMux()` を利用）。
2. その後、以下のコマンドでサーバーを起動できます。
   ```bash
   go run ./main.go
   ```
3. デフォルトでは `http://localhost:8080` で待ち受けます。ポートを変更したい場合は `registry.NewHTTPServer` に渡すアドレスを変更してください。

## モックデータ
現時点の `registry/server.go` では以下のデータが初期投入されます。

| id | name           | description                 | score |
|----|----------------|-----------------------------|-------|
| 1  | ドミネーター   | 犯罪係数で裁く未来銃        | 35    |
| 2  | タイムマシン   | 時間を超える装置            | 5     |
| 3  | 量子通信       | 瞬時に情報伝送（量子もつれ）| 60    |

## 想定 API 仕様（WIP）
プレゼンテーション層が完成すると、次のようなエンドポイントを提供する想定です。

| メソッド | パス                 | 説明                                |
|----------|----------------------|-------------------------------------|
| GET      | `/api/v1/techs`      | テクノロジーの一覧を取得            |
| GET      | `/api/v1/techs/{id}` | 指定 ID のテクノロジーを取得        |

`GET /api/v1/techs` のレスポンス例（想定）:

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

## テスト
現状はユニットテストが未整備です。`go test ./...` でテストが追加された際に全体を実行できる構成を想定しています。

## 今後のTODO
- プレゼンテーション層（ルーティングとハンドラー）の実装
- `domain.TechService` などユースケース層の実装とテストカバレッジ追加
- 永続化層の差し替え（例: RDB や外部 API）
- CI/CD パイプラインの整備

## ライセンス
このプロジェクトは [MIT License](LICENSE) の下で公開されています。
