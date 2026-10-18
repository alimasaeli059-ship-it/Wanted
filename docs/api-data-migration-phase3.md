# مرحلهٔ اجرایی سوم: آماده‌سازی API، پایگاه‌داده و مهاجرت کنترل‌شده

> **وضعیت این سند: طراحی و ممیزی مستنداتی؛ پیاده‌سازی عملیاتی انجام نشده است.** در workspace قابل مشاهده هیچ runtime بک‌اند، پیکربندی استقرار، PostgreSQL، API، ابزار migration یا test runner وجود ندارد. هیچ endpoint، schema اجرایی، دیتابیس یا اتصال ساختگی ایجاد نشده و داده‌ای از مرورگر مهاجرت نکرده است.

## ۱. نتیجهٔ بررسی محیط و شواهد

فهرست فایل‌های workspace در زمان این مرحله فقط `README.md`، `index.html`، `styles.css` و `app.js` را نشان می‌دهد. مستند مرحلهٔ دوم در `docs/security-auth-phase2.md` است. فایل‌های پیکربندی اجرایی نظیر `package.json`، `go.mod`، `pyproject.toml`، `Dockerfile`، Compose، تنظیمات استقرار/CI، migration، schema پایگاه‌داده، یا تنظیمات اتصال به سرور در فهرست نیستند. README نیز اجرای مستقیم فرانت‌اند استاتیک را مستند کرده است.

نتیجهٔ محدود به شواهد workspace: **ساخت و اجرای backend/API/PostgreSQL در این محیط تأییدشدنی نیست.** دسترسی بیرونی احتمالی به سرویس‌های حساب یا workspace را نمی‌توان از این فایل‌ها استنباط کرد. بنابراین موارد زیر در این مرحله اجرا نشدند: ایجاد migration، ساخت یا راه‌اندازی دیتابیس، API، اتصال رابط به API، import/export واقعی از مرورگر، و تست واحد/API/integration.

برای آغاز پیاده‌سازی واقعی لازم است تیم/مالک محیطی قابل اجرا فراهم کند که شامل این موارد باشد:

1. workspace یا مخزن دارای backend runtime تأییدشده، نسخهٔ پشتیبانی‌شده، روش اجرای محلی/CI و امکان نگهداری وابستگی‌ها؛ انتخاب runtime و framework باید با زیرساخت و مهارت تیم نهایی شود، نه بر اساس حدس.
2. PostgreSQL قابل دسترس از backend، با نسخهٔ پشتیبانی‌شده، TLS، حساب‌های کم‌دسترسی، ایجاد/اعمال migration و backup/restore آزمایشی. secret اتصال فقط در محیط امن سرور/secret manager باشد.
3. محیط‌های جداگانهٔ development و staging، دامنه/API origin، DNS و TLS؛ دسترسی شبکهٔ ورودی و خروجی، firewall، دسترسی webhook، محدودسازی CORS و سیاست proxy باید مستند شوند.
4. secret management و فرایند rotation، مدیریت لاگ/مانیتورینگ و شناسهٔ درخواست، مدیریت خطا، و دسترسی محدود برای افراد مجاز.
5. پیش از production: مالک محصول، tenant مقصد، نقش bootstrap مالک، ماتریس مجوز نهایی، سیاست حفظ/حذف داده، شرایط مهاجرت و مسئول تأیید عملیاتی باید مشخص باشند.

هیچ سرویس، VPS، دامنه یا دیتابیس خریداری یا متصل نشده است. صرفاً داشتن runtime یا اتصال دیتابیس نیز بدون staging، مدیریت secret و آزمون جداسازی tenant برای انتشار کافی نیست.

## ۲. ممیزی ساختار واقعی localStorage

### شیوهٔ ذخیره‌سازی که از کد قابل مشاهده است

در `app.js` داده‌های برنامه از helperهای `localRead` و `localSave` خوانده/نوشته می‌شوند. کلیدهای دامنه عمدتاً با الگوی `hoshyar:<business-id>:<نام-کلید>` ساخته می‌شوند. مقدار انتخابگر کسب‌وکار در کلید جداگانهٔ `hoshyar_active_business` است و اگر موجود نباشد، کد مقدار پیش‌فرض `demo-business-a` را به‌کار می‌گیرد. این انتخابگر، اثبات هویت یا مالکیت نیست.

فهرست زیر **کلیدها و شکل مورد انتظار در کد** را نشان می‌دهد؛ این workspace به محتوای localStorage واقعی مرورگر کاربر دسترسی ندارد، بنابراین وجود، تعداد، صحت یا نسخهٔ رکوردهای هیچ مرورگری را تأیید نمی‌کند. اکثر کلیدهای دامنه JSON array هستند، اما ممکن است در مرورگر مقدار خراب، ناقص، قدیمی یا قابل دستکاری وجود داشته باشد.

| کلید scoped (زیر `hoshyar:<business-id>:`) | شکل/معنای قابل مشاهده در کد | ملاحظهٔ مهاجرت |
|---|---|---|
| `hoshyar_users` | آرایهٔ کاربرهای محلی با فیلدهایی مانند `id`, `username`, `email`, `mobile`, `role`, `status`, `registrationDecision`, `createdAt` و برخی تنظیمات بازاریاب | هویت، نقش، وضعیت و تأیید مرورگری معتبر نیستند؛ به‌صورت حساب فعال یا عضویت مورد اعتماد وارد نشوند. اطلاعات تماس شخصی است. |
| `hoshyar_customers` | آرایهٔ رکوردهای CRM؛ حداقل نام و تلفن در منطق خواندن | مالکیت، رضایت، تکراری‌بودن و کیفیت تماس نامعلوم است؛ هر رکورد نیازمند preview و mapping صریح است. |
| `hoshyar_orders` | آرایهٔ سفارش‌های محلی، با `id` و جزئیات فرم/وضعیت | سفارش محلی اثبات خرید یا پرداخت نیست؛ وضعیت مالی از آن استنباط نشود. |
| `hoshyar_appointments` | آرایهٔ نوبت‌ها با شناسه و اطلاعات زمان/خدمت/مشتری | رزرو محلی، تأیید یا عدم‌تداخل سمت سرور را اثبات نمی‌کند؛ زمان و timezone باید بازبینی شود. |
| `calls` | آرایهٔ تماس؛ کد رکورد دستی با `id`, `source`, `contact`, `date`, `time`, `status`, `duration`, `note`, `createdAt` و فیلدهای provider می‌سازد | رکورد دستی یا source برابر provider، بدون شواهد/اعتبارسنجی سرویس تماس واقعی نیست. یادداشت و شماره حساس‌اند؛ صوت/متن معتبر سرویس از این کلید اثبات نمی‌شود. |
| `hoshyar_support_tickets` | آرایهٔ تیکت‌ها و پیام‌ها؛ خواندن، فیلدهای نمایشی و شناسه را نرمال می‌کند | نویسنده، زمان و پاسخ محلی قابل جعل‌اند؛ تیکت به حساب واقعی نسبت داده نشود. |
| `hoshyar_business` | شیء تنظیمات کسب‌وکار، از جمله `workingHours` و داده‌های پروفایل | هویت کسب‌وکار مقصد اثبات نشده؛ صرفاً محتوای پیشنهادی قابل بازبینی است. |
| `hoshyar_business_catalog` | آرایهٔ خدمات/محصولات؛ اشیای catalog | قیمت و شرایط محلی، قیمت تراکنش یا تعرفهٔ مصوب محسوب نمی‌شود. |
| `hoshyar_plans` | آرایهٔ پلن‌ها با فیلدهایی مانند نوع، نام، قیمت، دوره، محدودیت، قابلیت و فعال‌بودن | محتوای محلی منبع قیمت/اشتراک معتبر نیست؛ مالک باید تعرفه را جداگانه تأیید کند. |
| `hoshyar_subscriptions` | آرایهٔ تخصیص‌های محلی با ارجاع به `userId`/`planId`، وضعیت و تاریخ‌ها | هیچ وضعیت فعال یا حق دسترسی از این رکورد قابل اعتماد نیست. |
| `hoshyar_expenses` | آرایهٔ هزینه با `id`, `title`, `category`, `amount`, `date`, `kind`, `source`, `verificationStatus`, `description`, `createdAt` | ثبت دستی/تخمینی صورتحساب تأییدشده یا بدهی واقعی نیست؛ ارقام باید مجزا و تأیید شوند. |
| `hoshyar_marketer_trial_prospects` | آرایهٔ سرنخ با `id`, `marketerId`, `name`, `allowance`, `createdAt` | نسبت‌دادن بازاریاب و سهمیهٔ مرورگری قابل اعتماد نیست. |
| `hoshyar_marketer_trial_usage` | آرایهٔ رویداد مصرف محلی با `id`, `marketerId`, `prospectId`, `occurredAt`, `source`, `action` | دفترکل مصرف معتبر نیست؛ هیچ سهمیهٔ سروری را پر یا کم نکند. |
| `hoshyar_professional_marketing_packages` | آرایهٔ تنظیمات بسته‌های ۲۰/۵۰/۱۰۰ موردی، قیمت و فیلدهای `definition`, `deliverable`, `ownerConfirmed` | تعریف «مورد»/خروجی و تأیید محلی تا تصویب مستقل مالک قطعی نیست؛ تغییر/مهاجرت به محصول مصوب ممنوع تا تصمیم. |
| `hoshyar_professional_marketing_orders` | آرایهٔ درخواست بسته با شناسه، جزئیات/قیمت snapshot و وضعیت درخواست | درخواست محلی نه پرداخت است و نه تعهد تحویل؛ هر دو وضعیت باید جداگانه بررسی شوند. |
| `hoshyar_admin_notifications` | آرایهٔ اعلان محلی، مشتق از رخدادهای اقدام مرورگری و دارای `read` | اعلان‌ها پیام یا رخداد معتبر سرور نیستند؛ وضعیت خواندن فقط محلی است. |
| `hoshyar_admin_events` | آرایهٔ رویدادهای محلی، با `id`, `type`, `title`, `text`, `occurredAt`, `relatedId`, `result`, `source: browser_local` | audit امنیتی نیست؛ عامل احراز‌شده، زمان معتبر سرور و تغییرناپذیری ندارد. به audit سرور import نشود. |

دو کلید غیر دامنه‌ای نیز در کد دیده می‌شود: `hoshyar_active_business` (انتخابگر نمایشی بالا) و `hoshyar:<business-id>:__isolation_test` (دادهٔ موقت آزمون مرورگری). هیچ‌کدام دادهٔ کسب‌وکار قابل انتقال نیستند. تست browser-local ممکن است انتخابگر را تغییر دهد؛ نتیجهٔ آن کنترل tenant در سرور را اثبات نمی‌کند.

این inventory از call siteها و فیلدهای مورد استفاده/ساخته‌شده در `app.js` استخراج شده، نه از نمونهٔ دادهٔ واقعی. شکل‌های قدیمی یا ناشناخته ممکن است وجود داشته باشند؛ importer آینده باید schema ناشناخته را رد/قرنطینه کند، نه حدس بزند یا بی‌صدا اصلاح کند.

## ۳. طرح مدل رابطه‌ای پیشنهادی

این مدل، طراحی اولیه برای PostgreSQL است و migration اجرایی نیست. جدول‌های کسب‌وکاری tenant-scoped باید به `business_id` متصل باشند. شناسهٔ پایدار تصادفی (UUID یا شناسهٔ معادل غیرقابل حدس)، timestamp استاندارد UTC، FK، unique/indexهای لازم و سیاست آرشیو/حذف برای هر موجودیت مشخص شود. نقش مؤثر از عضویت سروری و نشست احراز‌شده استخراج می‌شود.

- `businesses`: tenant و تنظیمات.
- `users`, `registration_requests`, `sessions`, `credential_tokens`: هویت، درخواست در انتظار تأیید مالک، نشست‌های ابطال‌پذیر و توکن‌های موقت؛ رمز فقط password hash، نشست/token فقط hash ذخیره شود.
- `business_memberships`: ارتباط چندبه‌چند user/business، نقش مجاز (`owner`, `admin`, `marketer`, `customer`) و وضعیت؛ unique روی `(business_id,user_id)`؛ قواعد حفظ آخرین مالک در transaction.
- `customers`, `customer_interactions`, `followups`: CRM و پیگیری؛ مالکیت tenant و دسترسی نقش در هر عملیات.
- `catalog_items`, `orders`, `order_items`: قیمت/تعریف catalog و snapshot سفارش؛ وضعیت سفارش از وضعیت پرداخت/تحویل مستقل باشد.
- `appointments`: مشتری/خدمت، بازهٔ زمانی و وضعیت؛ جلوگیری از تعارض رزرو در دیتابیس و transaction، نه فقط validation رابط.
- `calls`: شناسهٔ provider، زمان، وضعیت اعتبارسنجی، metadata کمینه؛ صوت/transcript در storage امن جدا با رضایت و retention، نه ستون یا log پیش‌فرض.
- `support_tickets`, `support_messages`: تیکت، نویسندهٔ معتبر و پیام؛ کنترل مالکیت هر تیکت در query.
- `plans`, `subscriptions`: کاتالوگ و تخصیص؛ تغییر پلن جدید نباید snapshot تاریخی قرارداد/پرداخت را عوض کند.
- `payments`, `payment_events`, `cost_entries`, `usage_records`: وضعیت تراکنش و رویداد idempotent، هزینهٔ تأییدشده از دستی/تخمینی جدا، منبع و واحد/دورهٔ مصرف.
- `marketer_prospects`, `trial_quota_ledger`: سرنخ منتسب به بازاریاب و دفترکل مصرف. ثبت مصرف/کاهش سهمیه اتمیک، tenant-scoped و idempotent باشد.
- `professional_package_definitions`, `professional_package_orders`: تعریف versioned با actor/زمان تأیید و snapshot تعریف/خروجی/قیمت درخواست؛ وضعیت سفارش، پرداخت و fulfillment مستقل. تا تصمیم مالک، definition `unapproved` بماند.
- `notifications`, `audit_events`: اعلان per-user و رویداد ممیزی افزایشی؛ مجوز خواندن محدود و مسیر ویرایش/حذف برای کاربر معمولی نداشته باشد.
- `integration_accounts`: فقط provider، tenant، وضعیت، metadata غیرمحرمانه و ارجاع secret؛ secret واقعی خارج از جدول عمومی و خارج از frontend نگهداری شود.

محدودیت‌های ضروری: uniqueهای متناسب با tenant، FKهای restrict/cascade سنجیده، indexهای `(business_id, created_at)` و کلیدهای جست‌وجوی پرتکرار، check constraint وضعیت/مبلغ، و تراکنش برای تصمیم ثبت‌نام، رزرو، پرداخت/fulfillment و مصرف سهمیه. مبلغ از کلاینت پذیرفته نشود. Webhook تکراری با شناسهٔ منبع و idempotency key بدون تکرار اثر پردازش شود. دسترسی مالی در query/service سمت سرور محدود شود، نه با مخفی‌کردن رابط.

## ۴. قرارداد API پیشنهادی برای پیاده‌سازی آینده

هیچ‌یک از endpointهای زیر اکنون وجود ندارد. قرارداد نسخهٔ اولیه JSON زیر `/api/v1` پیشنهاد می‌شود؛ هر درخواست correlation/request ID، validation سمت سرور، پاسخ خطای یکنواخت و کنترل auth/tenant/permission داشته باشد. مسیرهای tenant از نشست مجاز استخراج شوند؛ وجود `business_id` در URL به‌تنهایی مجوز نیست.

| دامنه | مسیرهای اولیه | شرط اصلی |
|---|---|---|
| هویت/عضویت | `POST /auth/registrations`, `POST /auth/login`, `POST /auth/logout`, `GET /auth/session`, `GET /me/businesses` | ثبت‌نام `pending_owner_approval`؛ session cookie امن/CSRF؛ rate limit و پاسخ عمومی برای جلوگیری از user enumeration. |
| تأیید مالک/حساب | `GET /owner/registration-requests`, `POST /owner/registration-requests/{id}/decision`, `PATCH /owner/users/{id}/status`, `PATCH /owner/memberships/{id}/role` | فقط مالک مجاز؛ transition مجاز، audit، ابطال نشست پس از disable/block یا تغییر حساس. |
| مشتری/CRM | `GET/POST /customers`, `GET/PATCH/DELETE /customers/{id}`, `/customers/{id}/interactions`, `/customers/{id}/followups` | tenant و object ownership در هر query؛ ثبت actor سرور. |
| سفارش/نوبت | `GET/POST /orders`, `GET/PATCH /orders/{id}`, `GET/POST /appointments`, `PATCH /appointments/{id}` | مجوز per-operation، validation وضعیت، رقابت رزرو در transaction؛ سفارش ≠ پرداخت. |
| تماس/تیکت | `GET /calls`, `POST /provider-webhooks/calls`, `GET/POST /support/tickets`, `POST /support/tickets/{id}/messages` | webhook با اعتبارسنجی امضا/زمان/idempotency؛ مشتری فقط تیکت/تماس مجاز خود را می‌بیند. |
| پلن/اشتراک | `GET /plans`, `POST/PATCH /owner/plans`, `GET /subscriptions`, `POST /owner/subscriptions` | تغییر کاتالوگ و تخصیص محدود؛ entitlement از سرور محاسبه شود، نه مقدار درخواستی کلاینت. |
| پرداخت/هزینه | `POST /payments/checkout`, `POST /webhooks/payments/{provider}`, `GET /payments`, `GET /costs` | قیمت/سفارش از DB؛ verify سمت سرور؛ محرمانگی مالی، callback idempotent و audit. |
| دمو/بستهٔ بازاریابی | `GET/POST /marketer/prospects`, `POST /marketer/prospects/{id}/trial-uses`, `GET /professional-packages`, `POST /professional-package-orders` | scope سرنخ برای بازاریاب، سهمیه در transaction؛ تعریف بسته فقط پس از تأیید ثبت‌شدهٔ مالک؛ درخواست ≠ پرداخت/تحویل. |
| اعلان/audit | `GET /notifications`, `PATCH /notifications/{id}`, `GET /audit-events` | مجوز tenant/user؛ audit فقط append؛ پاسخ نباید راز یا دادهٔ غیرضروری افشا کند. |

وضعیت‌های خطا حداقل شامل `400` بدساختار، `401` نشست نامعتبر، `403` مجوز ناکافی، `404` ناموجود/خارج از دامنه، `409` تعارض یا duplicate، `422` اعتبارسنجی و `429` rate-limit باشد. هیچ stack trace، query، secret یا اطلاعاتی برای تشخیص وجود حساب در پاسخ عمومی قرار نگیرد.

## ۵. نقش‌ها و کنترل‌های سروری

ماتریس اولیهٔ مرحلهٔ دوم در `docs/security-auth-phase2.md` مرجع طراحی احراز هویت است. نقش canonical پیشنهادی این نقشه `owner`, `admin`, `marketer`, `customer` است. بعضی بخش‌های قدیمی README واژهٔ `manager` را به‌کار می‌برند؛ این معادل قطعی `admin` فرض نشود تا مالک آن را تأیید کند. نقش‌ها از payload مرورگر پذیرفته نمی‌شوند. برای همهٔ queryها tenant از نشست و عضویت فعال می‌آید؛ object-level authorization باید جداگانه بررسی شود.

- **owner:** مدیریت اعضا/تنظیمات tenant و تصمیم درخواست عضویت؛ دسترسی مالی/تعریف بسته فقط در محدودهٔ کسب‌وکار و با audit. جلوگیری از حذف یا تنزل آخرین مالک.
- **admin:** عملیات CRM، سفارش، نوبت و تیکت در دامنهٔ مجاز؛ دسترسی مالی، تغییر مالک، credential و تأیید بسته‌ها به‌طور پیش‌فرض ممنوع مگر مالک مجوز صریح بدهد.
- **marketer:** فقط سرنخ‌ها و سهمیهٔ منتسب؛ بدون مشاهدهٔ CRM کامل، اطلاعات مالی، سهمیهٔ دیگران یا تغییر نقش/اشتراک.
- **customer:** فقط پروفایل، سفارش، نوبت، تیکت، اشتراک و پرداخت خودش طبق سیاست محصول؛ هیچ دسترسی cross-customer یا دادهٔ داخلی.

هر API تغییر‌دهنده باید authentication، tenant membership، permission، اعتبارسنجی schema و state transition را سمت سرور اجرا کند. بررسی مالکیت شناسه باید در هر درخواست انجام شود تا IDOR ممکن نباشد. rate limit برای ثبت‌نام/ورود/reset و webhook abuse، CSRF/Origin برای cookie session، CORS محدود، TLS، secret manager، ثبت audit پالایش‌شده و ابطال نشست لازم‌اند.

## ۶. مهاجرت نسخه‌دار و قابل بازیابی — طرح، نه اجرا

مهاجرت تا فراهم‌شدن احراز هویت واقعی، انتخاب tenant مقصد، پایگاه‌داده و تأیید صریح مالک **ممنوع و اجرا نشده است**. پیشنهاد نسخهٔ اولیهٔ فرایند:

1. **Inventory/export فقط با رضایت:** کاربر export خام از مرورگر انتخابی بگیرد؛ manifest شامل شناسهٔ تصادفی export، زمان UTC، نسخهٔ exporter، schema version، کلیدهای موجود، تعداد رکورد، اندازه و checksum باشد. از localStorage کپی نشود مگر کاربر این مرورگر/پروفایل را صریحاً انتخاب کرده باشد. export رمز/credential را شامل نکند؛ فایل حساس محلی تلقی شود.
2. **عدم تغییر منبع:** نسخهٔ مبدأ read-only بماند. هیچ حذف، اصلاح، deduplication یا normalize خاموش انجام نشود. تبدیل‌های قابل برگشت در گزارش mapping ثبت شوند؛ نسخهٔ خام و checksum برای rollback حفظ شود.
3. **انتخاب مقصد و preview:** کاربر پس از ورود، کسب‌وکار مقصد را از membership معتبر server انتخاب کند، سپس کلیدها، نوع/تعداد، داده‌های ناقص، duplicateهای احتمالی و فیلدهای تبدیل/عدم انتقال را ببیند و برای همان مقصد تأیید جدا بدهد. انتخابگر `hoshyar_active_business` مبنای tenant نیست.
4. **stage/quarantine:** import ابتدا در جداول staging با `import_id`, source key, source record id/hash, schema version و نتیجهٔ validation ذخیره شود. هیچ رکورد staging تا commit نهایی برای عملیات محصول قابل مشاهده نباشد.
5. **نسخه و idempotency:** قرارداد import دارای `schema_version` و `import_id` ثابت باشد. unique constraint روی `(business_id, import_id, source_key, source_record_id)` یا کلید hash معادل، retry را idempotent کند. تبدیل هر نسخه به نسخهٔ جاری migration صریح و آزمون‌پذیر داشته باشد؛ نسخهٔ ناشناخته رد شود.
6. **گزارش/تصمیم رکورد:** وضعیت هر رکورد `accepted`, `needs_review`, `rejected`, `duplicate_candidate` و دلیل صریح ثبت شود. رکورد ناقص یا تکراری خودکار ادغام/حذف نشود؛ مالک/کاربر آن را بررسی کند. پاسخ/خطاها شامل شمارنده و شناسه باشند، نه دادهٔ حساس اضافی.
7. **تفکیک provenance:** CRM، تنظیمات، سفارش، نوبت، پلن و درخواست‌های محلی فقط با برچسب `imported_unverified` و مبدأ نگهداری شوند. درخواست کاربر برای حساب، role/status، پرداخت، هزینهٔ تأییدشده، رویداد ممیزی، تماس provider، مصرف سهمیه و تأیید بسته از مرورگر به‌عنوان حقیقت سرور پذیرفته نشوند. موارد غیرقابل انتقال در preview با دلیل مشخص شوند.
8. **تأیید و commit:** قبل از commit، خلاصهٔ دقیق و immutable از مقصد/تعداد/نگاشت/ردها نمایش و تأیید صریح گرفته شود. عملیات مجاز در transaction، با audit actor/import ID و idempotency اجرا شود. شکست باید transaction را rollback کند؛ پاک‌سازی مجاز فقط دادهٔ staging همان import را هدف بگیرد.
9. **بازیابی/حذف مبدأ:** پس از commit، reconcile تعداد و checksum و خروجی گزارش‌شده انجام شود. دادهٔ localStorage فقط با درخواست صریح و پس از export قابل بازیابی و تأیید موفقیت، در جریان جداگانه پاک شود؛ پاک‌کردن در این مرحلهٔ طراحی انجام نمی‌شود.
10. **پایلوت محدود:** ابتدا tenant آزمایشی و subset کم‌خطر، سپس تست بازیابی export/import و conflict، بعد داده‌های حساس فقط با رضایت و retention مصوب. تعمیم به همهٔ کاربران تنها پس از approval و rollback drill.

**خطرهای خاص داده:** `hoshyar_users` هویت/نقش/تصمیم سروری نمی‌سازد؛ `hoshyar_subscriptions` entitlement نمی‌دهد؛ `hoshyar_expenses` و درخواست‌های بسته پرداخت را ثابت نمی‌کنند؛ مصرف demo سهمیهٔ واقعی نیست؛ اعلان/رویداد مرورگر audit نیست. برای هرکدام، عدم انتقال به منبع حقیقت پیش‌فرض است.

## ۷. اتصال تدریجی رابط پس از API واقعی

تا وقتی API واقعی و تست‌شده در staging در دسترس نیست، frontend و حالت local نباید حذف یا به endpoint ساختگی متصل شوند. وقتی آماده شد:

1. یک adapter مشخص API با base URL محیطی/پیکربندی deployment و قرارداد پاسخ نسخه‌دار اضافه شود؛ secret هرگز به JS نرود.
2. قابلیت‌ها به‌ترتیب مستقل و پشت feature flag کنترل‌شدهٔ deployment متصل شوند؛ آغاز کار با session و دادهٔ کم‌خطر، سپس CRM، سفارش/نوبت، تیکت و سایر دامنه‌ها.
3. UI وضعیت‌های `سرور`, `محلی/واردنشده`, `در انتظار sync`, `خطا` را جدا و صریح نمایش دهد. fallback بی‌صدا از خطای server به local ممنوع باشد.
4. هیچ رکورد محلی از API پاسخ داده‌شده به‌عنوان دادهٔ server جا زده نشود. همگام‌سازی/conflict باید قابل مشاهده، idempotent و قابل بازبینی باشد.
5. local mode برای دورهٔ سازگاری باقی بماند، اما دقیقاً با برچسب محلی و بدون ادعای نقش، پرداخت، سهمیه یا audit امن؛ زمان و مسیر حذف/ادامهٔ آن پس از اندازه‌گیری و تصمیم مالک جداگانه تعیین شود.

در این مرحله این adapter یا feature flag به کد UI اضافه نشده است؛ اضافه‌کردن نقطهٔ اتصال بدون backend/contract قابل آزمون، خطر ایجاد رفتار نادرست دارد.

## ۸. آزمون‌های پذیرش پیش از اعلام آمادگی

هیچ‌یک از آزمون‌های این فهرست در این workspace اجرا نشده‌اند؛ این‌ها معیار کار آینده‌اند:

- **Migration/unit:** اعتبارسنجی نسخه‌ها، تبدیل schema، duplicate/conflict classification، idempotency و rollback transaction؛ دادهٔ malformed/unknown بدون حذف خام حفظ/قرنطینه شود.
- **API integration:** ثبت‌نام pending، auth/session/logout/expiry، مجوز نقش در هر endpoint، validation، error mapping، rate limit و webhook تکراری/نامعتبر.
- **tenant/security:** دو tenant A/B و چهار نقش؛ تلاش دسترسی به شناسهٔ resource tenant دیگر، دستکاری `business_id`/`user_id`/`role`، دسترسی مالی customer/marketer، دستکاری سهمیه و منع تغییر entitlement.
- **پایگاه‌داده:** قیدهای FK/unique، تراکنش رزرو هم‌زمان، آخرین owner، مصرف سهمیه هم‌زمان و پرداخت/fulfillment idempotent.
- **مهاجرت end-to-end:** preview، رضایت مقصد، import تکراری، رکورد ناقص/تکراری، audit، checksum/reconciliation، backup، restore و عدم تغییر مبدأ.
- **عملیاتی:** TLS/cookie/CSRF/CORS، secret scan، لاگ پالایش‌شده، backup restore drill، محدودسازی دسترسی DB و پایش خطا/health.

`check_project` فقط بررسی ایستای سازگاری فایل‌های فرانت‌اند است؛ جایگزین هیچ‌کدام از تست‌های API، database، مهاجرت یا امنیت عملی نیست.

## ۹. تصمیم‌ها و موانع باقی‌مانده

پیش از پیاده‌سازی، مالک/تیم باید این موارد را تأیید کند:

1. محیط backend واقعی، runtime/version، روش deploy، PostgreSQL و نسخه، مسئول patch و محل میزبانی؛ مجوز/دسترسی شبکه و staging.
2. domain و API origin، TLS، CORS، webhook ingress/egress و secret manager؛ هیچ credential در چت، مخزن یا فرانت‌اند قرار نگیرد.
3. معنای نهایی `admin` در برابر `manager`، ماتریس permission هر endpoint، bootstrap اولین owner، تأیید عضویت و MFA/نشست.
4. سیاست tenant و مالکیت داده، consent و retention/delete برای CRM، تماس، تیکت و اطلاعات مالی؛ RPO/RTO و backup/restore مسئول‌دار.
5. اولویت دامنه‌های API و قرارداد دقیق وضعیت‌های سفارش، نوبت، پرداخت، اشتراک و fulfillment؛ سیاست ارز/مبالغ و مجوز مالی.
6. تعریف «مورد»، خروجی و قیمت/تعهد بسته‌های بازاریابی؛ `ownerConfirmed` در localStorage تأیید مالک سروری محسوب نمی‌شود.
7. تعیین اینکه مهاجرت برای کدام مرورگر/پروفایل و کدام کسب‌وکار لازم است، چه داده‌ای مجاز به انتقال است، چه کسانی تأیید می‌کنند و چه مدت export نگهداری می‌شود.

تا رفع موارد زیر، API، دیتابیس، اتصال UI و مهاجرت **قابل انجام یا ایمن برای انتشار نیستند**: runtime قابل اجرا، PostgreSQL، staging، secret management، سیاست مجوز مصوب و انتخاب tenant. هیچ ادعای آمادگی عملیاتی یا فروش از این سند نتیجه نمی‌شود.

## ۱۰. خروجی واقعی این مرحله و نقشهٔ راه

- تکمیل این سند معماری، قرارداد مقدماتی، inventory کلیدهای کد و روش مهاجرت versioned؛ تغییر UI یا داده انجام نشد.
- با توجه به نبود زیرساخت، **مرحلهٔ API/DB/migration عملیاتی نشده است**؛ endpoint، schema اجرایی، migration یا import وجود ندارد.
- چهار گام عملیاتی نقشهٔ راه همچنان باز هستند: (۱) فراهم‌کردن/تأیید محیط backend، DB، staging و تصمیم‌های مالک؛ (۲) پیاده‌سازی هویت، schema پایه، tenant isolation و RBAC و آزمون امنیتی؛ (۳) اجرای API دامنه‌ها و مهاجرت کنترل‌شده پس از تأیید، با آزمون integration؛ (۴) اتصال AI/تلفن/پرداخت در sandbox و staging، backup restore و تأیید انتشار. مستندسازی فعلی هیچ‌یک را به‌عنوان اجراشده علامت نمی‌زند.
