package migrations

import (
	"os"
	"time"

	log "github.com/sirupsen/logrus"

	"upmeter/pkg/upmeter/db/context"
)

const CreateVersionTable = `
CREATE TABLE IF NOT EXISTS _schema_version (
	timestamp INTEGER NOT NULL,
	version TEXT NOT NULL
)
`

const InsertVersion = `
INSERT INTO _schema_version (timestamp, version) VALUES (?, ?)
`

const SelectVersions = `
SELECT timestamp, version FROM _schema_version
`

var Migrator = NewMigratorService()

type MigratorService struct {
	DbCtx *context.DbContext
}

func NewMigratorService() *MigratorService {
	return &MigratorService{}
}

func (m *MigratorService) Apply(dbCtx *context.DbContext) error {
	if os.Getenv("SKIP_MIGRATIONS") == "yes" {
		return nil
	}

	m.DbCtx = dbCtx.Start()
	defer m.DbCtx.Stop()

	versions := m.getVersions()

	// Apply all migrations
	if _, ok := versions["V0000"]; !ok {
		V0000_Up(m)
	}

	if _, ok := versions["V0001"]; !ok {
		V0001_Up(m)
	}

	if _, ok := versions["V0002"]; !ok {
		V0002_Up(m)
	}

	return nil
}

func (m *MigratorService) getVersions() map[string]int64 {
	// Ensure version table exists
	_, err := m.DbCtx.StmtRunner().Exec(CreateVersionTable)
	if err != nil {
		log.Errorf("MIGRATE: Create version table: %v", err)
	}

	versions := map[string]int64{}

	rows, err := m.DbCtx.StmtRunner().Query(SelectVersions)

	for rows.Next() {
		var version string
		var timestamp int64
		err := rows.Scan(&timestamp, &version)
		if err != nil {
			log.Errorf("scan version table: %v", err)
			return map[string]int64{}
		}

		versions[version] = timestamp
	}
	return versions
}

func (m *MigratorService) saveApplied(version string) {
	_, err := m.DbCtx.StmtRunner().Exec(InsertVersion, time.Now().Unix(), version)
	if err != nil {
		log.Errorf("MIGRATE: save applied version %s: %v", version, err)
	}
}

func (m *MigratorService) applyActions(version string, actions []map[string]string) {
	var err error

	for _, action := range actions {
		_, err = m.DbCtx.StmtRunner().Exec(action["sql"])
		if err != nil {
			log.Errorf("MIGRATE %s: %s: %v", version, action["desc"], err)
			return
		}
	}

	m.saveApplied(version)
	log.Infof("MIGRATE %s: SUCCESS", version)
}
