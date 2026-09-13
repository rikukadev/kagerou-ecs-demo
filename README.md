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
