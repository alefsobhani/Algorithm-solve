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
make dev
```

این دستور سرویس‌ها را در حالت توسعه با hot reload اجرا می‌کند (air) و دیتابیس/Redis/NATS را با Docker Compose بالا می‌آورد. برای اجرای کامل، پیش‌نیازهای زیر نیاز است:

- Go 1.22+
- Docker و Docker Compose
- golang-migrate
- sqlc

## نکات برجسته

- **Idempotency Key Middleware**: تضمین می‌کند درخواست‌های ایجاد سفر یا تغییر وضعیت در حالت تکراری به نتایج قبلی پاسخ دهند.
- **State Machine Engine**: تغییر وضعیت‌ها با optimistic locking و ثبت رویداد در جدول `trip_events` انجام می‌شود.
- **Outbox Dispatcher**: رویدادهای دامنه‌ای در جدول `outbox` ذخیره و توسط worker به NATS منتشر می‌شوند.
- **Observability**: هر سرویس از zap برای لاگ ساختار‌یافته، Prometheus برای مترک‌ها و OpenTelemetry برای tracing استفاده می‌کند.
- **تست‌ها**: نمونه تست‌های واحد برای لایهٔ سرویس با استفاده از in-memory repository قرار دارد.

## مسیر توسعهٔ بعدی

- پیاده‌سازی کامل الگوریتم مچینگ با Redis GeoIndex و پشتیبانی از backoff.
- اتصال به موتور مسیریابی (OSRM/Valhalla) برای ETA دقیق.
- اضافه کردن پوشش تست Integration با Testcontainers.
- تکمیل داکیومنت Swagger و مثال‌های Postman.

