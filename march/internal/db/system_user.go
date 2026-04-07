package db

import "github.com/google/uuid"

// SystemUserID is the seeded user row used for webhook-driven analysis_metrics.updated_by.
var SystemUserID = uuid.MustParse("00000000-0000-0000-0000-000000000001")
