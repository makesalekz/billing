-- Modify "invoices" table
ALTER TABLE "invoices" ADD COLUMN "is_revoked" boolean NOT NULL DEFAULT false, ADD COLUMN "revoked_at" timestamptz NULL, ADD COLUMN "is_revoked_processed" boolean NOT NULL DEFAULT false, ADD COLUMN "apple_store_transaction_id" character varying NULL;
