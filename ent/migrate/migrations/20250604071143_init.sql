-- Modify "invoices" table
ALTER TABLE "invoices" ADD COLUMN "original_apple_transaction_id" character varying NULL;
-- Modify "product_reservations" table
ALTER TABLE "product_reservations" DROP CONSTRAINT "product_reservations_invoices_reservations", DROP CONSTRAINT "product_reservations_products_reservations", DROP COLUMN "invoice_reservations", DROP COLUMN "product_reservations", ADD CONSTRAINT "product_reservations_invoices_reservations" FOREIGN KEY ("invoice_id") REFERENCES "invoices" ("id") ON UPDATE NO ACTION ON DELETE NO ACTION, ADD CONSTRAINT "product_reservations_products_reservations" FOREIGN KEY ("product_id") REFERENCES "products" ("id") ON UPDATE NO ACTION ON DELETE NO ACTION;
