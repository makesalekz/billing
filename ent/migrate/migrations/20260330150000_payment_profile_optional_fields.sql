-- Make phone, email, user_token optional in payment_profiles (TTP doesn't always provide them)
ALTER TABLE "payment_profiles" ALTER COLUMN "phone" SET DEFAULT '';
ALTER TABLE "payment_profiles" ALTER COLUMN "phone" DROP NOT NULL;
ALTER TABLE "payment_profiles" ALTER COLUMN "email" SET DEFAULT '';
ALTER TABLE "payment_profiles" ALTER COLUMN "email" DROP NOT NULL;
ALTER TABLE "payment_profiles" ALTER COLUMN "user_token" SET DEFAULT '';
ALTER TABLE "payment_profiles" ALTER COLUMN "user_token" DROP NOT NULL;
