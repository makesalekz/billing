ALTER SEQUENCE "items_id_seq" RESTART WITH 1000;
ALTER SEQUENCE "products_id_seq" RESTART WITH 1000;

INSERT INTO items
    (id, "name", description, topic_name, created_at, updated_at)
VALUES (1, 'Qalai Premium Features', 'Qalai Premium Features', 'qalai-premium-features', now(), now());

INSERT INTO products
(id, app_id, "name", description, price, currency, is_active, is_limited, limited_till, "left", is_unique, unique_limit,
 is_expiring, expiring_time, created_at, updated_at)
values (10, 'calendaria', 'Qalai Premium Subscription', 'Qalai Premium Subscription', 4.99, 'USD', true, false, null, 0,
        false, 0, false, null, now(), now());

INSERT INTO bundles
    (id, amount, item_id, product_id, created_at, updated_at)
values (1, 1, 1, 10, now(), now());