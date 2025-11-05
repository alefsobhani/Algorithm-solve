# RideLite — میکروسرویس درخواست سفر، مچینگ و ETA

RideLite یک پروژهٔ مرجع برای نمایش مهارت‌های بک‌اند با Go است که جریان کامل یک سامانهٔ تاکسی‌آنلاین را با معماری میکروسرویسی شبیه‌سازی می‌کند. این پروژه برای مصاحبه‌ها و ارزیابی شرکت‌هایی مانند اسنپ طراحی شده است و بر مفاهیم حیاتی نظیر State Machine سفر، استریم لوکیشن رانندگان، محاسبهٔ ETA، Idempotency، Outbox، Observability و CI/CD تأکید دارد.

## نمای کلی معماری

```
┌────────┐        ┌────────────┐        ┌────────────┐
│ Client │ ─────► │ API Gateway│ ─────► │ Trip Service│
└────────┘        └────────────┘        └────────────┘
                       │  ▲                     │
                       │  │ gRPC                │ نردبان State Machine + Outbox
                       ▼  │                     ▼
                ┌──────────────┐        ┌──────────────┐
                │ Location/ETA │◄──────►│ NATS Message │
                │   Service    │        │     Bus      │
                └──────────────┘        └──────────────┘
                       │
                       ▼
                 PostgreSQL + Redis
```

- **API Gateway**: لایهٔ ورودی REST که احراز هویت JWT، Rate-Limit و مترک‌ها را مدیریت می‌کند.
- **Trip Service**: مدیریت چرخهٔ عمر سفر، مچینگ راننده، ثبت رویدادها و انتشار Outbox.
- **Location/ETA Service**: استریم موقعیت رانندگان با gRPC و محاسبهٔ ETA با استفاده از Redis و مدل‌های مسیریابی.
- **NATS**: برای انتشار رویدادهای دامنه‌ای و آگاهی سرویس‌ها از تغییر وضعیت سفر.
- **PostgreSQL + Redis**: ذخیرهٔ پایدار داده‌ها و کش موقعیت/حالت راننده.

## سرویس‌ها و دایرکتوری‌ها

| مسیر | توضیح |
|------|-------|
| `cmd/apigateway` | راه‌اندازی API Gateway با chi و middlewares احراز هویت/آبزروبیلیتی |
| `cmd/tripservice` | سرور HTTP برای مدیریت سفرها و webhook outbox worker |
| `cmd/locationservice` | سرور gRPC استریم موقعیت و REST ETA |
| `internal/trip` | لایه‌های handler/service/repository و منطق State Machine |
| `internal/eta` | محاسبهٔ ETA، دسترسی به Redis و مدل‌های فاصله |
| `internal/location` | مدیریت استریم gRPC و ذخیرهٔ لوکیشن |
| `pkg/outbox` | پیاده‌سازی الگوی Outbox برای انتشار رویدادها |
| `pkg/observability` | تنظیم zap، Prometheus و OpenTelemetry |
| `configs` | فایل‌های پیکربندی sqlc، migrate و نمونه env |
| `migrations` | اسکریپت‌های golang-migrate شامل اسکیمای اصلی |

## اجرای سریع در حالت توسعه

```bash
make mod
make sqlc
make migrate-up
make dev
```

مراحل بالا وابستگی‌های Go را همگام، کدهای sqlc را تولید، مایگریشن‌ها را اعمال و در نهایت تمام سرویس‌ها را در حالت توسعه با hot reload اجرا می‌کند. پس از بالا آمدن سرویس‌ها می‌توانید مستندات تعاملی Swagger را در آدرس [http://localhost:8088/docs](http://localhost:8088/docs) باز کنید و تمام مسیرها را مستقیماً از مرورگر فراخوانی نمایید. برای اجرای کامل، پیش‌نیازهای زیر نیاز است:

- Go 1.22+
- Docker و Docker Compose
- golang-migrate
- sqlc

## نکات برجسته

- **Idempotency Key Middleware**: تضمین می‌کند درخواست‌های ایجاد سفر یا تغییر وضعیت در حالت تکراری به نتایج قبلی پاسخ دهند.
- **State Machine Engine**: تغییر وضعیت‌ها با optimistic locking و ثبت رویداد در جدول `trip_events` انجام می‌شود.
- **Redis GEO Matching**: مختصات رانندگان در کلید `driver:locs` ذخیره و با `GEOSEARCH`، رزرو اتمیک و backoff نمایی راننده مناسب انتخاب می‌شود. مترک‌های `matching_time_seconds` و `assignment_attempts_total` رفتار سیستم را نشان می‌دهند.
- **Outbox Dispatcher Worker**: ورکری پس‌زمینه هر ۲۰۰ms صف `outbox` را با `FOR UPDATE SKIP LOCKED` می‌خواند، رویدادها را به NATS منتشر و پس از موفقیت `published=true` می‌کند. مترک‌های `outbox_publish_total`, `outbox_fail_total`, `outbox_lag_seconds` وضعیت صف را پایش می‌کنند.
- **Observability**: هر سرویس از zap برای لاگ ساختار‌یافته، Prometheus برای مترک‌ها و OpenTelemetry برای tracing استفاده می‌کند.
- **تست‌ها**: واحد و اینتگریشن با Testcontainers (Redis/Postgres/NATS) سناریوهای رزرو و بازیابی Outbox را پوشش می‌دهد.
- **Swagger/OpenAPI**: فایل `api/openapi.yaml` به صورت خودکار در Gateway سرو می‌شود و UI تعاملی Swagger از مسیر `/docs` در دسترس است.

## پیکربندی محیطی کلیدی

نمونهٔ کامل متغیرها در `configs/app.example.env` قرار دارد. مهم‌ترین موارد:

| متغیر | توضیح | مقدار پیش‌فرض |
|-------|-------|---------------|
| `REDIS_ADDR` | آدرس Redis برای GeoIndex و رزرو راننده | `redis:6379` |
| `MATCH_RADIUS_KM` | شعاع جست‌وجو به کیلومتر | `5` |
| `MATCH_TOPK` | سقف راننده بررسی‌شده در هر نوبت | `5` |
| `RESERVE_TTL_SEC` | TTL رزرو راننده در Redis | `10` |
| `MATCH_MAX_ATTEMPTS` | تعداد تلاش مجدد با backoff نمایی | `5` |
| `OUTBOX_POLL_MS` | بازهٔ اجرای worker (میلی‌ثانیه) | `200` |
| `OUTBOX_BATCH` | حداکثر رکورد در هر batch | `100` |
| `OUTBOX_RETRY_MAX` | سقف تلاش انتشار NATS | `5` |

## اجرای تست‌ها

تست‌های واحد و اینتگریشن (نیازمند Docker برای Testcontainers):

```bash
go test ./...
```

## مسیر توسعهٔ بعدی

- اتصال به موتور مسیریابی (OSRM/Valhalla) برای ETA دقیق.
- تکمیل داکیومنت Swagger و مثال‌های Postman.

