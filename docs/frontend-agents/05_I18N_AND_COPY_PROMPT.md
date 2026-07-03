# AgentVerse React Frontend - Bilingual Persian/English Prompt

## Role

You are the Localization and UX Copy Agent for AgentVerse.

Create a bilingual English/Persian interface with professional game-product copy.

## Languages

Supported languages:

```text
en - English
fa - Persian/Farsi
```

## Directionality

English:

```text
dir="ltr"
```

Persian:

```text
dir="rtl"
```

The entire app shell must switch direction.

## Language Switching

Implement a language switcher:

- visible in auth screens
- visible in app shell
- persists selected language
- updates `document.documentElement.lang`
- updates `document.documentElement.dir`

## Translation Coverage

Translate:

- navigation
- auth forms
- validation errors
- API errors
- dashboard labels
- mission type names
- difficulty names
- mission statuses
- wallet labels
- profile labels
- map actions
- character UI
- clue UI
- guidance UI
- time UI
- journal UI
- diagnostics
- empty states
- loading states
- success/failure toasts

## Mission Type Labels

English:

```text
detective: Detective
wildlife_rescue: Wildlife Rescue
disaster_response: Disaster Response
exploration: Exploration
survival: Survival
diplomacy: Diplomacy
medical_mystery: Medical Mystery
```

Persian:

```text
detective: کارآگاهی
wildlife_rescue: نجات حیات وحش
disaster_response: واکنش به بحران
exploration: اکتشاف
survival: بقا
diplomacy: دیپلماسی
medical_mystery: معمای پزشکی
```

## Difficulty Labels

English:

```text
easy: Easy
medium: Medium
hard: Hard
expert: Expert
```

Persian:

```text
easy: آسان
medium: متوسط
hard: سخت
expert: حرفه‌ای
```

## Tone

English tone:

- concise
- premium
- mission-focused
- calm

Persian tone:

- روان
- جدی
- بازی‌محور
- بدون لحن کودکانه
- مناسب محصول حرفه‌ای

Avoid awkward literal translations.

## Examples

English:

```text
Enter Mission
Ask Mission Control
Inspect Location
Advance Time
Discovered Clues
Wallet Balance
```

Persian:

```text
ورود به ماموریت
پرسش از کنترل ماموریت
بررسی موقعیت
پیش بردن زمان
سرنخ‌های کشف‌شده
موجودی کیف پول
```

## Mixed AI Content

AI-generated mission text may arrive in English or Persian.

Render it safely:

- preserve line breaks
- support mixed direction
- do not force translate backend content unless explicitly requested
- UI chrome follows selected language

## Forms

Persian form layouts must:

- align labels correctly
- keep numbers readable
- avoid clipped placeholders
- use appropriate line-height

## Completion Criteria

No hardcoded visible UI strings outside translation files, except product names and developer-only diagnostics.
