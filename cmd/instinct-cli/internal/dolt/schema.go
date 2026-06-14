package dolt

// Schema returns DDL statements that create all tables managed by this package.
// Called by both production setup code (db.go) and tests.
func Schema() []string {
	return []string{
		`CREATE TABLE instincts (
			id                VARCHAR(64)   PRIMARY KEY,
			content           TEXT          NOT NULL,
			trigger_desc      TEXT          NOT NULL,
			domain            VARCHAR(128),
			source            ENUM('auto','manual') NOT NULL DEFAULT 'auto',
			scope             ENUM('project','global') NOT NULL DEFAULT 'project',
			project_id        VARCHAR(12)   NOT NULL,
			project_name      VARCHAR(256),
			observation_count INT           NOT NULL DEFAULT 0,
			created_at        TIMESTAMP     DEFAULT CURRENT_TIMESTAMP,
			updated_at        TIMESTAMP     DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE dedup_decisions (
			id              VARCHAR(64)   PRIMARY KEY,
			instinct_id_a   VARCHAR(64)   NOT NULL,
			instinct_id_b   VARCHAR(64)   NOT NULL,
			content_a       TEXT          NOT NULL,
			content_b       TEXT          NOT NULL,
			trigger_a       TEXT          NOT NULL,
			trigger_b       TEXT          NOT NULL,
			decision        ENUM('duplicate','distinct') NOT NULL,
			reasoning       TEXT,
			sim_bigram      DECIMAL(4,3),
			sim_trigram     DECIMAL(4,3),
			sim_overlap     DECIMAL(4,3),
			decided_by      ENUM('agent','human') NOT NULL DEFAULT 'agent',
			human_label     ENUM('correct','wrong'),
			source_branch_a VARCHAR(128),
			source_branch_b VARCHAR(128),
			winner_branch   VARCHAR(128),
			created_at      TIMESTAMP     DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE review_queue (
			instinct_id     VARCHAR(64)   PRIMARY KEY,
			content         TEXT          NOT NULL,
			trigger_desc    TEXT          NOT NULL,
			domain          VARCHAR(128),
			observation_count INT         NOT NULL DEFAULT 0,
			scope           ENUM('project','global') NOT NULL DEFAULT 'project',
			submitted_by    VARCHAR(256)  NOT NULL,
			submitted_at    TIMESTAMP     DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
		)`,
	}
}
