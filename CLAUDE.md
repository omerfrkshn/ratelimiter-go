# ratelimiter-go — proje notları

## Yerel Redis

```bash
docker compose up -d      # redis:7-alpine, localhost:6379
docker compose down       # kapat
```

Entegrasyon testleri Redis'e bağlanamazsa `t.Skip` ile atlanır, başarısız
olmaz — CI'da her zaman `services.redis` ile ayağa kaldırılır (bkz.
`.github/workflows/ci.yml`).

## Test komutları

```bash
go test ./...                                        # unit testler
go test -race ./...                                   # race detector (cgo + gcc gerekir)
go test -tags=integration -race ./redislimiter/...    # Redis gerektirir
```

`-race` Windows'ta cgo gerektirir, cgo da bir C derleyicisi ister — bu
makinede MinGW-w64 (`gcc`) winget ile kuruldu. Linux CI'da (Ubuntu) gcc
zaten hazır geliyor, ekstra kurulum gerekmiyor.

## Yük testi (benchmark)

```bash
go build -o demo.exe ./cmd/demo
./demo.exe -algorithm=<fixed-window|sliding-log|sliding-counter|token-bucket> -rate=50 -period=5s -addr=:8098 &
vegeta attack -targets=loadtest/targets.txt -rate=20/1s -duration=20s -timeout=5s \
  | vegeta report -type=text
```

Sonuçlar [`BENCHMARKS.md`](BENCHMARKS.md)'de: limit, toplam istek, kabul/red
sayısı, p50/p95/p99 formatında. Yeni bir yük testi yapılırsa aynı formatı
koru ve `loadtest/results/<algoritma>.txt` altına ham vegeta raporunu ekle.

## Paket/isimlendirme yapısı

- `limiter/` — 4 bellek-içi algoritma, hepsi `Limiter` arayüzünü uygular
  (`Allow(key string) bool`). Dosya adı = algoritma adı (`tokenbucket.go`),
  test dosyası aynı pakette (`tokenbucket_test.go`), clock `time.Now`
  yerine enjekte edilebilir unexported `clock` alanı ile test edilir.
- `redislimiter/` — dağıtık mod, Redis + Lua script. Context alan
  metotlar `(bool, error)` döner (bellek-içi `Limiter`den farklı arayüz,
  çünkü ağ hatası olabilir).
- `middleware/` — net/http middleware, herhangi bir `limiter.Limiter`'ı
  sarar.
- `cmd/demo/` — kütüphaneyi gösteren minimal HTTP servisi.

## CI

GitHub Actions (`.github/workflows/ci.yml`): build → `go test -race` →
Redis entegrasyon testi → `golangci-lint`. main'e her push/PR'da çalışır.
