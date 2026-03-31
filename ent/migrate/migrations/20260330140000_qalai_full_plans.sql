-- Full Qalai plans: display items, Starter product, all bundles

-- =============================================
-- 1. Display items (features for pricing page)
-- =============================================

INSERT INTO "items" ("created_at", "updated_at", "name", "description")
VALUES
  (NOW(), NOW(), 'AI чат с ассистентом', 'Общайтесь с AI ассистентом по любым вопросам'),
  (NOW(), NOW(), 'Документы в базе знаний', 'Загружайте документы для AI анализа'),
  (NOW(), NOW(), 'Загрузка файлов', 'Максимальный размер загружаемого файла (MB)'),
  (NOW(), NOW(), 'AI агенты', 'Создавайте кастомных AI ассистентов с навыками'),
  (NOW(), NOW(), 'Запись и резюме встреч', 'Автоматическая запись, транскрипция и саммари'),
  (NOW(), NOW(), 'Граф знаний (KAG)', 'Извлечение сущностей, связей, визуализация знаний'),
  (NOW(), NOW(), 'Автоматизации', 'Создавайте автоматические процессы и workflows'),
  (NOW(), NOW(), 'Приглашение в команду', 'Приглашайте коллег в рабочее пространство'),
  (NOW(), NOW(), 'Общее рабочее пространство', 'Совместная работа над проектами и документами'),
  (NOW(), NOW(), 'Приоритетная поддержка', 'Быстрый ответ и выделенный менеджер');

-- Save display item IDs into temp table for bundle creation
CREATE TEMP TABLE _display_items AS
SELECT id, name FROM items WHERE topic_name IS NULL ORDER BY id;

-- =============================================
-- 2. Starter product (free, no subscription)
-- =============================================

INSERT INTO "products" ("app_id", "name", "description", "price", "currency", "is_active", "is_limited", "left", "is_unique", "unique_limit", "is_expiring", "payment_model", "product_period", "created_at", "updated_at")
VALUES (
  'knowledge',
  'Qalai Starter',
  'Начните бесплатно — AI чат и база знаний',
  0, 'KZT', true, false, 0, false, 0, false,
  'ONE_TIME', 'unlimited',
  NOW(), NOW()
);

-- =============================================
-- 3. Delete old bundles for products 1-4 and add full bundles
-- =============================================

DELETE FROM "bundles" WHERE product_id IN (1, 2, 3, 4);

-- Starter bundles (product = last inserted = Starter)
INSERT INTO "bundles" ("created_at", "updated_at", "amount", "item_id", "product_id")
SELECT NOW(), NOW(), b.amount, di.id, p.id
FROM (VALUES
  ('AI чат с ассистентом', 1),
  ('Документы в базе знаний', 20),
  ('Загрузка файлов', 100)
) AS b(item_name, amount)
JOIN _display_items di ON di.name = b.item_name
CROSS JOIN products p WHERE p.name = 'Qalai Starter';

-- Pro Monthly bundles (product_id = 1)
INSERT INTO "bundles" ("created_at", "updated_at", "amount", "item_id", "product_id")
VALUES (NOW(), NOW(), 1, 1, 1);  -- RBAC trigger: Qalai Premium Features

INSERT INTO "bundles" ("created_at", "updated_at", "amount", "item_id", "product_id")
SELECT NOW(), NOW(), b.amount, di.id, 1
FROM (VALUES
  ('AI чат с ассистентом', 1),
  ('Документы в базе знаний', -1),
  ('Загрузка файлов', 500),
  ('AI агенты', -1),
  ('Запись и резюме встреч', -1),
  ('Граф знаний (KAG)', 1),
  ('Автоматизации', -1)
) AS b(item_name, amount)
JOIN _display_items di ON di.name = b.item_name;

-- Pro Yearly bundles (product_id = 2) — same features as Monthly
INSERT INTO "bundles" ("created_at", "updated_at", "amount", "item_id", "product_id")
VALUES (NOW(), NOW(), 1, 1, 2);  -- RBAC trigger

INSERT INTO "bundles" ("created_at", "updated_at", "amount", "item_id", "product_id")
SELECT NOW(), NOW(), b.amount, di.id, 2
FROM (VALUES
  ('AI чат с ассистентом', 1),
  ('Документы в базе знаний', -1),
  ('Загрузка файлов', 500),
  ('AI агенты', -1),
  ('Запись и резюме встреч', -1),
  ('Граф знаний (KAG)', 1),
  ('Автоматизации', -1)
) AS b(item_name, amount)
JOIN _display_items di ON di.name = b.item_name;

-- Business Monthly bundles (product_id = 3)
INSERT INTO "bundles" ("created_at", "updated_at", "amount", "item_id", "product_id")
VALUES
  (NOW(), NOW(), 1, 1, 3),  -- RBAC trigger: Premium
  (NOW(), NOW(), 1, 2, 3);  -- RBAC trigger: Business

INSERT INTO "bundles" ("created_at", "updated_at", "amount", "item_id", "product_id")
SELECT NOW(), NOW(), b.amount, di.id, 3
FROM (VALUES
  ('AI чат с ассистентом', 1),
  ('Документы в базе знаний', -1),
  ('Загрузка файлов', -1),
  ('AI агенты', -1),
  ('Запись и резюме встреч', -1),
  ('Граф знаний (KAG)', 1),
  ('Автоматизации', -1),
  ('Приглашение в команду', -1),
  ('Общее рабочее пространство', 1),
  ('Приоритетная поддержка', 1)
) AS b(item_name, amount)
JOIN _display_items di ON di.name = b.item_name;

-- Business Yearly bundles (product_id = 4) — same features as Monthly
INSERT INTO "bundles" ("created_at", "updated_at", "amount", "item_id", "product_id")
VALUES
  (NOW(), NOW(), 1, 1, 4),  -- RBAC trigger: Premium
  (NOW(), NOW(), 1, 2, 4);  -- RBAC trigger: Business

INSERT INTO "bundles" ("created_at", "updated_at", "amount", "item_id", "product_id")
SELECT NOW(), NOW(), b.amount, di.id, 4
FROM (VALUES
  ('AI чат с ассистентом', 1),
  ('Документы в базе знаний', -1),
  ('Загрузка файлов', -1),
  ('AI агенты', -1),
  ('Запись и резюме встреч', -1),
  ('Граф знаний (KAG)', 1),
  ('Автоматизации', -1),
  ('Приглашение в команду', -1),
  ('Общее рабочее пространство', 1),
  ('Приоритетная поддержка', 1)
) AS b(item_name, amount)
JOIN _display_items di ON di.name = b.item_name;

DROP TABLE _display_items;
