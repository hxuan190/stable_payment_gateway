-- Drop travel_rule_data table and related objects
DROP INDEX IF EXISTS idx_travel_rule_payer_wallet;
DROP INDEX IF EXISTS idx_travel_rule_merchant_country;
DROP INDEX IF EXISTS idx_travel_rule_payer_country;
DROP INDEX IF EXISTS idx_travel_rule_created_at;
DROP INDEX IF EXISTS idx_travel_rule_payment_id;

DROP TABLE IF EXISTS travel_rule_data;
