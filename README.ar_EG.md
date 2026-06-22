[English](/README.md) | [فارسی](/README.fa_IR.md) | [العربية](/README.ar_EG.md) | [中文](/README.zh_CN.md) | [Español](/README.es_ES.md) | [Русский](/README.ru_RU.md) | [Türkçe](/README.tr_TR.md)

<p align="center">
  <picture>
    <source media="(prefers-color-scheme: dark)" srcset="./media/3x-ui-dark.png">
    <img alt="3x-ui-multiport" src="./media/3x-ui-light.png">
  </picture>
</p>

<p align="center">
  <a href="https://github.com/exhxx-tg/3x-ui-multiport/releases"><img src="https://img.shields.io/github/v/release/exhxx-tg/3x-ui-multiport" alt="Release"></a>
  <a href="https://github.com/exhxx-tg/3x-ui-multiport/actions"><img src="https://img.shields.io/github/actions/workflow/status/exhxx-tg/3x-ui-multiport/release.yml.svg" alt="Build"></a>
  <a href="#"><img src="https://img.shields.io/github/go-mod/go-version/exhxx-tg/3x-ui-multiport.svg" alt="GO Version"></a>
  <a href="https://github.com/exhxx-tg/3x-ui-multiport/releases/latest"><img src="https://img.shields.io/github/downloads/exhxx-tg/3x-ui-multiport/total.svg" alt="Downloads"></a>
  <a href="https://www.gnu.org/licenses/gpl-3.0.en.html"><img src="https://img.shields.io/badge/license-GPL%20V3-blue.svg?longCache=true" alt="License"></a>
  <a href="https://pkg.go.dev/github.com/exhxx-tg/3x-ui-multiport"><img src="https://pkg.go.dev/badge/github.com/exhxx-tg/3x-ui-multiport.svg" alt="Go Reference"></a>
  <a href="https://goreportcard.com/report/github.com/exhxx-tg/3x-ui-multiport"><img src="https://goreportcard.com/badge/github.com/exhxx-tg/3x-ui-multiport" alt="Go Report Card"></a>
</p>

**X-UI PRO (3x-ui-multiport)** هي لوحة تحكم ويب متقدمة ومفتوحة المصدر لإدارة **13 بروتوكول** موحّد — تجمع بين [Xray-core](https://github.com/XTLS/Xray-core) وخدمات VPN المستقلة وطبقات التمويه في واجهة إدارة واحدة. توفّر واجهة نظيفة ومتعددة اللغات لنشر وتكوين ومراقبة وتأمين بروتوكولات الوكيل وVPN — من خادم VPS واحد إلى عمليات النشر متعددة العقد.

تم بناء X-UI PRO كنسخة مطوّرة من 3X-UI، وتضيف 8 بروتوكولات إضافية، وأمانًا على مستوى المؤسسات، ومراقبة شاملة، وأدوات DevOps جاهزة للإنتاج.

> [!IMPORTANT]
> هذا المشروع مخصص للاستخدام الشخصي فقط. يرجى عدم استخدامه لأغراض غير قانونية أو في بيئة إنتاجية.

## نظام الـ 13 بروتوكول

### 🔹 البروتوكولات الأساسية (Xray-native)
| البروتوكول | الوصف | المصدر |
|---|---|---|
| **VMess** | وكيل شبيه بـ Socks5 مع تشفير | Xray-core |
| **VLESS** | بديل VMess خفيف بدون تشفير | Xray-core |
| **Trojan** | بروتوكول مبني على TLS يقلد HTTPS | Xray-core |
| **Shadowsocks** | Socks5 بسيط مع تشفير | Xray-core |
| **Hysteria** | بروتوكول UDP محسّن للسرعة | Xray-core |

### 🔹 الخدمات المستقلة
| الخدمة | الوصف | المصدر |
|---|---|---|
| **OpenVPN** | VPN قياسي (TCP/UDP) | OpenVPN |
| **WireGuard** | VPN حديث يعمل في النواة | WireGuard |
| **Dropbear** | خادم SSH خفيف الوزن | Dropbear |

### 🔹 طبقات التمويه (Transport Wrappers)
| الطبقة | الوصف | متوافقة مع |
|---|---|---|
| **WebSocket** | نفق WebSocket HTTP | VMess, VLESS, SS, Trojan |
| **TLS/HTTPS** | نفق مشفر بـ TLS | VMess, VLESS, SS, Trojan |
| **HTTP/2** | نفق متعدد الإرسال HTTP/2 | VLESS, Trojan |
| **gRPC** | تغليف بروتوكول gRPC | VLESS, Trojan |
| **Naive** | نفق HTTP CONNECT | جميع البروتوكولات |

## الميزات

- **13 بروتوكول** — 5 أساسية (VMess, VLESS, Trojan, Shadowsocks, Hysteria) + 3 خدمات مستقلة (OpenVPN, WireGuard, Dropbear) + 5 طبقات تمويه (WebSocket, TLS, HTTP/2, gRPC, Naive)
- **وسائل نقل وأمان حديثة** — TCP، mKCP، WebSocket، gRPC، HTTPUpgrade، XHTTP، مؤمَّنة بـ TLS و XTLS و REALITY
- **Fallback** — تقديم عدة بروتوكولات على منفذ واحد
- **إدارة لكل عميل** — حصص الترافيك، تواريخ انتهاء، حدود IP، حالة اتصال مباشرة
- **إحصائيات ترافيك** — لكل اتصال وارد وكل عميل وكل اتصال صادر
- **دعم العقد المتعددة** — إدارة عبر عدة خوادم من لوحة واحدة
- **أمان متكامل** — RBAC، 2FA، سجل التدقيق، التحكم بعناوين IP، تحديد المعدل
- **مراقبة شاملة** — فحوصات الصحة، مقاييس الأداء، قواعد التنبيه، Prometheus
- **أداء محسّن** — تجمّع العمال، التخزين المؤقت، تجمّع اتصالات DB
- **أداة CLI** — إدارة كاملة للبروتوكولات من الطرفية
- **Kubernetes** — ملفات K8s كاملة
- **13 لغة واجهة** مع سمات داكنة وفاتحة

## البدء السريع

### تثبيت سطر واحد (Bare Metal)
```bash
bash <(curl -Ls https://raw.githubusercontent.com/exhxx-tg/3x-ui-multiport/main/install.sh)
```

### Docker (موصى به)
```bash
docker run -d \
  --name x-ui \
  --cap-add=NET_ADMIN \
  --cap-add=NET_RAW \
  -p 2053:2053 \
  -v x-ui-db:/etc/x-ui \
  ghcr.io/exhxx-tg/3x-ui-multiport:latest
```

### Docker Compose
```bash
git clone https://github.com/exhxx-tg/3x-ui-multiport.git
cd 3x-ui-multiport
docker compose up -d
```

### Kubernetes
```bash
kubectl apply -k deploy/k8s/
```

للحصول على الوثائق الكاملة، يرجى زيارة [docs/](docs/).

## المنصات المدعومة

**أنظمة التشغيل:** Ubuntu، Debian، Armbian، Fedora، CentOS، RHEL، AlmaLinux، Rocky Linux، Oracle Linux، Amazon Linux، Virtuozzo، Arch، Manjaro، Parch، openSUSE، Alpine و Windows.

**المعماريات:** `amd64` · `386` · `arm64` (aarch64) · `armv7` · `armv6` · `armv5` · `s390x`.

## اللغات المدعومة

English · فارسی · العربية · 中文（简体） · 中文（繁體） · Español · Русский · Українська · Türkçe · Tiếng Việt · 日本語 · Bahasa Indonesia · Português (Brasil)

## المساهمة

المساهمات مرحب بها. يرجى قراءة [دليل المساهمة](/CONTRIBUTING.md).

## شكر خاص

- [alireza0](https://github.com/alireza0/)
- [MHSanaei](https://github.com/MHSanaei/) — مشروع 3X-UI الأصلي

## دعم المشروع

**إذا كان هذا المشروع مفيدًا لك، أعطه**:star2:

[![Stargazers over time](https://starchart.cc/exhxx-tg/3x-ui-multiport.svg?variant=adaptive)](https://starchart.cc/exhxx-tg/3x-ui-multiport)
