-- TipTopPay migration: schema changes + Qalai AI products seed

-- 1. Add ttp_subscription_id column to invoices
ALTER TABLE "invoices" ADD COLUMN IF NOT EXISTS "ttp_subscription_id" character varying NULL;

-- 2. payment_provider is character varying, no ALTER TYPE needed
-- TIP_TOP_PAYMENT is a new value handled by application code

-- =============================================
-- SEED DATA: Qalai AI Items, Products, Bundles
-- =============================================

-- Items (NATS topics for RBAC integration)
INSERT INTO "items" ("id", "created_at", "updated_at", "name", "description", "topic_name")
VALUES
  (1, NOW(), NOW(), 'Qalai Pro Features', 'Unlocks Pro features: agents, recaps, KAG, automations, PMS', 'qalai-premium-features'),
  (2, NOW(), NOW(), 'Qalai Business Features', 'Unlocks Business features: team invites, shared workspace, priority support', 'qalai-business-features')
ON CONFLICT ("id") DO NOTHING;

-- Reset sequence after manual ID insert
SELECT setval(pg_get_serial_sequence('items', 'id'), GREATEST((SELECT MAX(id) FROM items), 2));

-- Products: Pro Monthly
INSERT INTO "products" ("id", "app_id", "name", "description", "price", "currency", "is_active", "is_limited", "left", "is_unique", "unique_limit", "is_expiring", "payment_model", "product_period", "created_at", "updated_at")
VALUES (
  1, 'knowledge',
  'Qalai Pro Monthly',
  'All AI features for individual use. Agents, recaps, KAG, automations, PMS. Billed monthly.',
  7990, 'KZT', true, false, 0, false, 0, false,
  'RECURRENT', 'month',
  NOW(), NOW()
) ON CONFLICT ("id") DO NOTHING;

-- Products: Pro Yearly (2 months free: 10 * 7990 = 79900)
INSERT INTO "products" ("id", "app_id", "name", "description", "price", "currency", "is_active", "is_limited", "left", "is_unique", "unique_limit", "is_expiring", "payment_model", "product_period", "created_at", "updated_at")
VALUES (
  2, 'knowledge',
  'Qalai Pro Yearly',
  'All AI features for individual use. Agents, recaps, KAG, automations, PMS. Billed yearly (2 months free).',
  79900, 'KZT', true, false, 0, false, 0, false,
  'RECURRENT', 'year',
  NOW(), NOW()
) ON CONFLICT ("id") DO NOTHING;

-- Products: Business Monthly
INSERT INTO "products" ("id", "app_id", "name", "description", "price", "currency", "is_active", "is_limited", "left", "is_unique", "unique_limit", "is_expiring", "payment_model", "product_period", "created_at", "updated_at")
VALUES (
  3, 'knowledge',
  'Qalai Business Monthly',
  'Full platform for teams. Everything in Pro + team invites, shared workspace, priority support. Billed monthly.',
  19990, 'KZT', true, false, 0, false, 0, false,
  'RECURRENT', 'month',
  NOW(), NOW()
) ON CONFLICT ("id") DO NOTHING;

-- Products: Business Yearly (2 months free: 10 * 19990 = 199900)
INSERT INTO "products" ("id", "app_id", "name", "description", "price", "currency", "is_active", "is_limited", "left", "is_unique", "unique_limit", "is_expiring", "payment_model", "product_period", "created_at", "updated_at")
VALUES (
  4, 'knowledge',
  'Qalai Business Yearly',
  'Full platform for teams. Everything in Pro + team invites, shared workspace, priority support. Billed yearly (2 months free).',
  199900, 'KZT', true, false, 0, false, 0, false,
  'RECURRENT', 'year',
  NOW(), NOW()
) ON CONFLICT ("id") DO NOTHING;

-- Reset product sequence
SELECT setval(pg_get_serial_sequence('products', 'id'), GREATEST((SELECT MAX(id) FROM products), 4));

-- Bundles: Pro products → Pro item
INSERT INTO "bundles" ("created_at", "updated_at", "amount", "item_id", "product_id")
VALUES
  (NOW(), NOW(), 1, 1, 1),  -- Pro Monthly → Qalai Pro Features
  (NOW(), NOW(), 1, 1, 2)   -- Pro Yearly  → Qalai Pro Features
ON CONFLICT DO NOTHING;

-- Bundles: Business products → Pro item + Business item
INSERT INTO "bundles" ("created_at", "updated_at", "amount", "item_id", "product_id")
VALUES
  (NOW(), NOW(), 1, 1, 3),  -- Business Monthly → Qalai Pro Features
  (NOW(), NOW(), 1, 2, 3),  -- Business Monthly → Qalai Business Features
  (NOW(), NOW(), 1, 1, 4),  -- Business Yearly  → Qalai Pro Features
  (NOW(), NOW(), 1, 2, 4)   -- Business Yearly  → Qalai Business Features
ON CONFLICT DO NOTHING;
