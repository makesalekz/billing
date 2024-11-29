ALTER SEQUENCE "items_id_seq" RESTART WITH 1000;
ALTER SEQUENCE "products_id_seq" RESTART WITH 1000;

INSERT INTO items
    (id,"name", description, topic_name)
VALUES
    (1,'Qalai Premium Features','Qalai Premium Features','qalai-premium-features');

INSERT INTO products
    (id, app_id, "name", description,price, currency, is_active, is_limited, limited_till, "left", is_unique, unique_limit, is_expiring, expiring_time)
values
    (10,'calendaria','Qalai Premium Subscription', 'Qalai Premium Subscription', 4.99, 'USD', true, false, null, null, false, null, false, null);

INSERT INTO bundles
    (id, amount, item_id, product_id)
values
    (1,1,1,10);