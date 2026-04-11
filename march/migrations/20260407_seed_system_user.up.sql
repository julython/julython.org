-- Fixed UUID for automated analysis_metrics.updated_by (webhook L1 scans).
INSERT INTO users (id, name, username, role)
VALUES (
  '00000000-0000-0000-0000-000000000001',
  'Julython System',
  'julython-system',
  'admin'
)
ON CONFLICT (id) DO NOTHING;
