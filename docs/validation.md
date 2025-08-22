# Order Validation Rules

## 📋 Overview
Валидация заказов для микросервиса обработки заказов. Все проверки выполняются строго согласно бизнес-логике.

## 🎯 Main Order Validation

### Обязательные поля:
- `order_uid` - не пустой
- `track_number` - не пустой (после trim)
- `entry` - не пустой (после trim)
- `locale` - не пустой (после trim)
- `customer_id` - не пустой (после trim)
- `delivery_service` - не пустой (после trim)
- `shardkey` - не пустой (после trim)
- `oof_shard` - не пустой (после trim)

### Числовые ограничения:
- `sm_id` ≥ 0 (не отрицательный)

### Временные ограничения:
- `date_created` - обязательное поле
- `date_created` ≤ текущее время (не может быть в будущем)

## 🚚 Delivery Validation

### Обязательные поля:
- `name` - не пустой
- `phone` - не пустой
- `zip` - не пустой
- `city` - не пустой
- `address` - не пустой
- `region` - не пустой
- `email` - не пустой

### Форматные проверки:
- `email` - должен быть валидным email форматом

## 💳 Payment Validation

### Обязательные поля:
- `transaction` - не пустой
- `currency` - не пустой
- `provider` - не пустой
- `bank` - не пустой

### Числовые ограничения:
- `amount` ≥ 0
- `payment_dt` > 0 (положительный timestamp)
- `delivery_cost` ≥ 0
- `goods_total` ≥ 0
- `custom_fee` ≥ 0

## 🛍️ Items Validation

### Общие требования:
- Минимум 1 товар в заказе
- Все товары должны проходить индивидуальную валидацию

### Per Item Validation:

#### Обязательные поля:
- `track_number` - не пустой
- `rid` - не пустой
- `name` - не пустой
- `size` - не пустой
- `brand` - не пустой

#### Числовые ограничения:
- `chrt_id` > 0 (положительный)
- `price` ≥ 0
- `sale` ∈ [0, 100] (включительно)
- `total_price` ≥ 0
- `nm_id` > 0 (положительный)
- `status` ≥ 0

## 🚨 Error Handling

### Формат ошибок:
- Каждая ошибка содержит название поля и причину
- Вложенные структуры имеют префиксы (`delivery validation failed: `, `item %d: `)
- Все ошибки на английском языке

### Примеры ошибок:
```
"order_uid is required"
"delivery validation failed: email has invalid format"
"item 2: sale must be between 0 and 100"
"payment validation failed: amount must be non-negative"
```

## ✅ Validation Flow

1. **Main Order** → базовые поля заказа
2. **Delivery** → информация о доставке
3. **Payment** → платежная информация
4. **Items** → проверка каждого товара

Если любая проверка fails - валидация немедленно прерывается и возвращается ошибка.

## 🛡️ Security Notes

- Все строковые поля проходят `strings.TrimSpace()` перед проверкой
- Email валидация через стандартный `net/mail.ParseAddress`
- Нет проверок на максимальные длины (только на наличие)
- Все числовые проверки включают граничные значения (≥ 0, > 0)