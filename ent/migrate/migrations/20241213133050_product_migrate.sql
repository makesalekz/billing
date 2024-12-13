INSERT INTO products
(id, app_id, "name", description, price, currency, is_active, is_limited, limited_till, "left", is_unique, unique_limit,
 is_expiring, expiring_time, created_at, updated_at)
values (11, 'calendaria', 'Qalai Premium Subscription', 'Qalai Premium Subscription', 7.99, 'USD', true, false, null, 0,
        false, 0, false, null, now(), now());

UPDATE bundles
SET product_id=11
WHERE id = 1;

UPDATE subscriptions
SET product_id=11
WHERE product_id = 10;

UPDATE invoices
SET product_id=11
WHERE product_id = 10;

DELETE
FROM products
where id = 10;