CREATE TABLE analysis_metrics (
  id           uuid         NOT NULL,
  project_id   uuid         NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
  metric_type  text         NOT NULL,
  level        smallint     NOT NULL DEFAULT 0,
  score        smallint     NOT NULL DEFAULT 0,
  data         jsonb        NOT NULL DEFAULT '{}',
  sha          text         NOT NULL DEFAULT '',
  updated_at   timestamptz  NOT NULL DEFAULT now(),
  updated_by   uuid         NOT NULL REFERENCES users(id),

  CONSTRAINT pk_analysis_metrics        PRIMARY KEY (id),
  CONSTRAINT uq_analysis_metrics        UNIQUE (project_id, metric_type),
  CONSTRAINT chk_analysis_metrics_level CHECK (level BETWEEN 0 AND 3),
  CONSTRAINT chk_analysis_metrics_score CHECK (score BETWEEN 0 AND 10)
);

CREATE INDEX ix_analysis_metrics_project ON analysis_metrics (project_id);
CREATE INDEX ix_analysis_metrics_type    ON analysis_metrics (metric_type);