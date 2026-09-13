# kagerou-ecs-demo

[kagerou](https://github.com/rikukadev/kagerou) の**複数コンテナ構成**の検証用アプリ。
1 つの Fargate タスクに 3 コンテナ(gateway / api / worker)を立て、
**共有 ALB** 経由で gateway だけを公開する。

## なぜ ALB は共有なのか

ALB は 1 台あたり**月 18 ドル前後の固定費**がかかる。PR ごとに作ると
ephemeral と両立しないので、ALB・証明書・DNS・ECS クラスタは
**共有ベース**(`deploy/alb-base.yaml`、1 アカウント/VPC に 1 回)が持ち、
環境が作るのは**リスナールールとターゲットグループ**(どちらも無料)だけ。

CloudFront ベース(静的配信)は従量課金で固定費が無いため per-app が既定だが、
ALB は逆に共有が既定になる。同じ「共有ベース」でも動機が違う。

## 何を確かめられるか

| 確認したいこと | 見る場所 |
|---|---|
| 3 コンテナが本当に動いているか | 画面(api の応答 + worker の tick) |
| **コンテナ間通信**(同一タスク = localhost) | 「api コンテナ」セクション |
| 共有ボリューム越しの受け渡し | 「worker コンテナ」セクション |
| 環境ごとにホスト名が分かれているか | URL と画面の環境名 |
| close で ECS / ルール / TG が消え、**ALB は残る**か | AWS 側 |

## 構成

- 1 イメージに 3 バイナリを入れ、タスク定義の `command` で使い分ける
  (イメージの push が 1 回で済み、PR ごとの待ち時間が短い)
- `gateway` だけ ALB のターゲット。`api` は localhost:8081、`worker` は共有ボリューム
- `AssignPublicIp: ENABLED`(NAT ゲートウェイの固定費を避けるため)
- ヘルスチェックは gateway 自身の `/healthz` のみ。api / worker を見ない
  (依存の一時障害でタスクが作り直されるのを避ける)
- Dockerfile は **ENTRYPOINT ではなく CMD**。ECS のタスク定義 `Command` は
  CMD を上書きするが ENTRYPOINT は上書きしないので、ENTRYPOINT にすると
  3 コンテナ全部が同じバイナリを起動し、awsvpc(タスク内でネットワークを共有)で
  ポートが衝突する

## 入口を 2 通り用意してある

同じ 3 コンテナを、**共有 ALB**(`template.yaml` + `deploy/alb-base.yaml`)と
**API Gateway**(`template-apigw.yaml` + `deploy/apigw-base.yaml`)の
どちらからでも公開できる。設計の違いは**入口の課金形態**から生まれる:

| | 共有 ALB | API Gateway(HTTP API + VPC Link) |
|---|---|---|
| 固定費 | **月 18 ドル前後** | **なし**(リクエスト課金) |
| 入口の所有 | 共有(1 本を全環境で使う) | **環境ごと**(API も独自ドメインも環境が持つ) |
| ルーティング | リスナールール(ホストヘッダ)+ **優先度の採番が必要** | ルート(メソッド + パス)。**採番不要** |
| ターゲット | ターゲットグループ(IP) | Cloud Map(SRV)+ VPC Link |
| 環境作成 | 約 2 分 46 秒 | 約 3 分 40 秒 |
| 環境削除 | 3〜4 分 | 約 2 分 18 秒 |
| タイムアウト上限 | 最大 4000 秒 | **30 秒** |

**ALB のルートキーはホストで分かれるが、HTTP API のルートキーは
「メソッド + パス」で Host を見ない。** だから 1 つの API を共有すると全環境が
`ANY /{proxy+}` を取り合って衝突する。API Gateway は固定費が無いので、
共有せず環境ごとに API を持たせる方が素直 — 「共有すべきか」は
入口の課金形態で決まる。

```bash
kagerou up --name pr-1                                # ALB 版
kagerou up --name pr-1 --config kagerou-apigw.yaml    # API Gateway 版
```

## 実測(2026-09-13、実 AWS)

| | |
|---|---|
| 環境の作成 | **2 分 46 秒**(Fargate 起動 + ALB ヘルスチェック) |
| 環境の削除 | **4 分 9 秒**(ECS サービスの停止が主) |
| 同時 2 環境 | 干渉なし(リスナールール優先度は環境名の CRC32 から決定的に採番) |
| 未定義ホスト | 404(共有リスナーの既定アクション) |

## 手元で

```bash
go run ./cmd/api &        # :8081
go run ./cmd/gateway      # :8080 → http://localhost:8080
```

## 関連

- [kagerou](https://github.com/rikukadev/kagerou)
- [kagerou-ssr-demo](https://github.com/rikukadev/kagerou-ssr-demo) — SSR を 1 Lambda で
- [kagerou-spa-demo](https://github.com/rikukadev/kagerou-spa-demo) — static driver
- [kagerou-3tier-demo](https://github.com/rikukadev/kagerou-3tier-demo) — SPA + API + RDB
