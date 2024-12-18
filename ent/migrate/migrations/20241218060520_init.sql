-- Modify "products" table
ALTER TABLE "products" ADD COLUMN "metadata" character varying NULL;

INSERT INTO products
(id, app_id, "name", description, price, currency, is_active, is_limited, limited_till, "left", is_unique, unique_limit,
 is_expiring, expiring_time, created_at, updated_at, metadata)
values (12, 'calendaria', 'Qalai Premium Trial', 'Qalai Premium Trial', 0, 'USD', true, false, null, 0,
        true, 1, false, null, now(), now(),'trial=True');

INSERT INTO bundles
(id, amount, item_id, product_id, created_at, updated_at)
values (2, 1, 1, 12, now(), now());