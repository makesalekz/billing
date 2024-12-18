-- Modify "invoices" table
ALTER TABLE "invoices" ADD COLUMN "is_trial" boolean NOT NULL DEFAULT false;
