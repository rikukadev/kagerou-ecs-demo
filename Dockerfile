# 3 サービスを 1 イメージに入れ、ECS のタスク定義で command を変えて使い分ける。
# (イメージを 3 つ作るより push が 1 回で済み、PR ごとの待ち時間が短い)
FROM golang:1.26-alpine AS build
WORKDIR /src
COPY go.mod ./
COPY cmd ./cmd
ARG GIT_COMMIT=dev
ARG BUILT_AT=dev
RUN CGO_ENABLED=0 go build -o /out/gateway ./cmd/gateway \
 && CGO_ENABLED=0 go build -o /out/api ./cmd/api \
 && CGO_ENABLED=0 go build -o /out/worker ./cmd/worker

FROM gcr.io/distroless/static-debian12
COPY --from=build /out/gateway /out/api /out/worker /
ARG GIT_COMMIT=dev
ARG BUILT_AT=dev
ENV GIT_COMMIT=$GIT_COMMIT BUILT_AT=$BUILT_AT
EXPOSE 8080
# ECS のタスク定義 Command は **CMD を上書きするが ENTRYPOINT は上書きしない**。
# ENTRYPOINT にすると 3 コンテナ全部が gateway を起動し、awsvpc(ネットワーク
# 共有)でポートが衝突する。1 イメージを command で使い分けるなら CMD にする。
CMD ["/gateway"]
