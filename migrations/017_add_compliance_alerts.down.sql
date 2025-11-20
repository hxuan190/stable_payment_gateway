-- Drop compliance_alerts table and related objects

DROP TRIGGER IF EXISTS compliance_alerts_updated_at ON compliance_alerts;
DROP FUNCTION IF EXISTS update_compliance_alerts_updated_at();
DROP TABLE IF EXISTS compliance_alerts;
