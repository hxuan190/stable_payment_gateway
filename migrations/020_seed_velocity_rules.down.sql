-- Rollback velocity rule seeding
DELETE FROM aml_rules WHERE id IN (
    'VEL_BASIC_001',
    'VEL_SPIKE_001',
    'VEL_VOLUME_001',
    'VEL_HFT_001',
    'VEL_CASHOUT_001'
);

