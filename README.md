# ratelimiter-go

Dört farklı rate limiting (istek sınırlama) algoritmasının, gerçek yük
testiyle doğrulanmış bir Go implementasyonu: fixed window, sliding window
log, sliding window counter ve token bucket. Tek-node bellek-içi kullanım
için sıfır bağımlılık; birden fazla uygulama instance'ı arkasında tutarlı
limit uygulamak için Redis üzerinden dağıtık bir mod.

## Neden dört algoritma

Her biri farklı bir trade-off temsil ediyor; "en iyi" algoritma yok,
doğru algoritma iş yüküne bağlı.

| Algoritma | Bellek/anahtar | Hassasiyet | Burst toleransı | Ne zaman kullan |
|---|---|---|---|---|
| **Fixed window** | O(1) | Düşük — pencere sınırında 2x'e kadar geçiş olabilir | Yok (istemsiz) | Kabaca yeterli, en ucuz seçenek |
| **Sliding window log** | O(rate) — her isteğin zaman damgası tutulur | Tam — hiçbir sınır efekti yok | Yok | Doğruluk kritikse ve rate düşükse |
| **Sliding window counter** | O(1) | Yaklaşık — iki pencereyi ağırlıklandırır | Yok | Log'un doğruluğuna yakın, log'un maliyeti olmadan — pratikte en çok kullanılan |
| **Token bucket** | O(1) | Ortalama oranı korur | Var — anlık burst'e izin verir | İstemci davranışı "bursty ama iyi niyetli" ise |

Ölçülmüş sayılar için bkz. [BENCHMARKS.md](BENCHMARKS.md) — aynı sürdürülebilir
limitin üzerinde sabit yükte, üç pencere tabanlı algoritma da %50 kabul
oranında kilitlenirken token bucket başlangıç burst'ü sayesinde %62.3'e
çıkıyor.

## Kurulum

```bash
go get github.com/omerfrkshn/ratelimiter-go
```

## Kullanım (tek-node, bellek-içi)

```go
import (
	"time"

	"github.com/omerfrkshn/ratelimiter-go/limiter"
)

// 60 saniyede 100 istek
rl := limiter.NewTokenBucket(limiter.Limit{Rate: 100, Period: time.Minute})

if rl.Allow(userID) {
	// isteği işle
} else {
	// 429 döndür
}
```

Dört algoritmanın hepsi aynı `limiter.Limiter` arayüzünü uygular
(`Allow(key string) bool`), yani birini diğeriyle değiştirmek tek satır.

## HTTP middleware

```go
import "github.com/omerfrkshn/ratelimiter-go/middleware"

handler := middleware.RateLimit(middleware.Config{
	Limiter: limiter.NewTokenBucket(limiter.Limit{Rate: 100, Period: time.Minute}),
	KeyFunc: middleware.KeyByIP, // veya kendi KeyFunc'ın (örn. API key'e göre)
})(mux)
```

Limit aşıldığında `429 Too Many Requests` döner.

### Demo servisi çalıştırma

```bash
go run ./cmd/demo -algorithm=token-bucket -rate=100 -period=1m -addr=:8080
curl -i http://localhost:8080/
```

`-algorithm`: `fixed-window` | `sliding-log` | `sliding-counter` | `token-bucket`

## Dağıtık mod (Redis)

Tek-node implementasyonların hepsi process belleğinde tutulur — bir load
balancer arkasında N instance varsa, her instance kendi limitini ayrı
uygular ve gerçek limit N katına çıkar. `redislimiter` paketi bunu, atomik
bir Lua script ile Redis'te paylaşılan bir sorted set üzerinden çözer:

```go
import (
	"github.com/redis/go-redis/v9"
	"github.com/omerfrkshn/ratelimiter-go/limiter"
	"github.com/omerfrkshn/ratelimiter-go/redislimiter"
)

client := redis.NewClient(&redis.Options{Addr: "localhost:6379"})
rl := redislimiter.NewSlidingWindow(client, limiter.Limit{Rate: 100, Period: time.Minute})

ok, err := rl.Allow(ctx, userID)
```

Yerelde Redis'i ayağa kaldırmak için:

```bash
docker compose up -d
```

Bunun gerçekten çalıştığının kanıtı, iki bağımsız Redis client'ının (iki
ayrı "instance" simülasyonu) aynı anahtara paralel istek gönderip toplamda
limiti aşmadığını doğrulayan entegrasyon testi:

```bash
go test -tags=integration ./redislimiter/... -v -race
```

## Test etme

```bash
go test ./...                              # tüm unit testler
go test -race ./...                        # race detector (cgo + gcc gerektirir)
go test -tags=integration ./redislimiter/... -race   # Redis gerektirir, docker compose up -d önce
```

## CI

GitHub Actions her push/PR'da şunları çalıştırır: `go build`, `go test -race`,
`golangci-lint run`. Bkz. [`.github/workflows/ci.yml`](.github/workflows/ci.yml).

## Lisans

MIT — bkz. [LICENSE](LICENSE).
