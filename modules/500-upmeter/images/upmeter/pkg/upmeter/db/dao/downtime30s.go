package dao

import (
	"fmt"

	log "github.com/sirupsen/logrus"

	"upmeter/pkg/checks"
	dbcontext "upmeter/pkg/upmeter/db/context"
)

const CreateTableDowntime30s_latest = `
CREATE TABLE IF NOT EXISTS downtime30s (
	timeslot INTEGER NOT NULL,
    success_seconds INTEGER NOT NULL,
    fail_seconds INTEGER NOT NULL,
    unknown_seconds INTEGER NOT NULL,
    nodata_seconds INTEGER NOT NULL,
    group_name TEXT NOT NULL,
    probe_name TEXT NOT NULL
)
`

const SelectDowntime30SecByTimeslot = `
SELECT
  rowid, timeslot, success_seconds, fail_seconds, unknown_seconds, nodata_seconds, group_name, probe_name
FROM downtime30s
WHERE
  timeslot = ?
`

const SelectDowntime30SecByTimeslotRange = `
SELECT
  rowid, timeslot, success_seconds, fail_seconds, unknown_seconds, nodata_seconds, group_name, probe_name
FROM downtime30s
WHERE
      timeslot >= ? AND timeslot <= ?
      AND group_name = ? AND probe_name = ?
`

const SelectDowntime30SecByTimeslotGroupProbe = `
SELECT
  rowid, timeslot, success_seconds, fail_seconds, unknown_seconds, nodata_seconds, group_name, probe_name
FROM downtime30s
WHERE
  timeslot = ? AND group_name = ? AND probe_name = ?
`

const SelectDowntime30SecGroupProbe = `
SELECT DISTINCT group_name, probe_name
FROM downtime30s
ORDER BY 1, 2
`

const SelectDowntime30SecStats = `
SELECT timeslot, count(timeslot)
FROM downtime30s
GROUP BY timeslot
`

const InsertDowntime30Sec = `
INSERT INTO downtime30s (timeslot, success_seconds, fail_seconds, unknown_seconds, nodata_seconds, group_name, probe_name)
VALUES
(?, ?, ?, ?, ?, ?, ?)
`

const UpdateDowntime30SecById = `
UPDATE downtime30s
SET
    success_seconds=?,
    fail_seconds=?,
    unknown_seconds=?,
    nodata_seconds=?
WHERE rowid=?
`

const DeleteDowntime30SecByEarlierTimestamp = `
DELETE FROM downtime30s
WHERE timeslot < ?
`

type Downtime30sDao struct {
	DbCtx    *dbcontext.DbContext
	ConnPool *dbcontext.ConnPool
	Table    string
}

func NewDowntime30sDao(dbCtx *dbcontext.DbContext) *Downtime30sDao {
	return &Downtime30sDao{
		DbCtx: dbCtx,
		Table: "downtime30s",
	}
}

type Downtime30sEntity struct {
	Rowid           int64
	DowntimeEpisode checks.DowntimeEpisode
}

func (d *Downtime30sDao) ListByTimestamp(tm int64) ([]Downtime30sEntity, error) {
	rows, err := d.DbCtx.StmtRunner().Query(SelectDowntime30SecByTimeslot, tm)
	if err != nil {
		return nil, fmt.Errorf("select for timestamp: %v", err)
	}
	defer rows.Close()

	var res = make([]Downtime30sEntity, 0)
	for rows.Next() {
		var entity = Downtime30sEntity{}
		err := rows.Scan(&entity.Rowid,
			&entity.DowntimeEpisode.TimeSlot,
			&entity.DowntimeEpisode.SuccessSeconds,
			&entity.DowntimeEpisode.FailSeconds,
			&entity.DowntimeEpisode.Unknown,
			&entity.DowntimeEpisode.NoData,
			&entity.DowntimeEpisode.ProbeRef.Group,
			&entity.DowntimeEpisode.ProbeRef.Probe)
		if err != nil {
			return nil, fmt.Errorf("row to Downtime30sEntity: %v", err)
		}
		res = append(res, entity)
	}

	return res, nil
}

func (d *Downtime30sDao) GetSimilar(downtime checks.DowntimeEpisode) (Downtime30sEntity, error) {
	rows, err := d.DbCtx.StmtRunner().Query(SelectDowntime30SecByTimeslotGroupProbe, downtime.TimeSlot, downtime.ProbeRef.Group, downtime.ProbeRef.Probe)
	if err != nil {
		return Downtime30sEntity{}, fmt.Errorf("select for timestamp: %v", err)
	}
	defer rows.Close()

	if !rows.Next() {
		// No entities found, return impossible rowid
		return Downtime30sEntity{Rowid: -1}, nil
	}

	var entity = Downtime30sEntity{}
	err = rows.Scan(&entity.Rowid,
		&entity.DowntimeEpisode.TimeSlot,
		&entity.DowntimeEpisode.SuccessSeconds,
		&entity.DowntimeEpisode.FailSeconds,
		&entity.DowntimeEpisode.Unknown,
		&entity.DowntimeEpisode.NoData,
		&entity.DowntimeEpisode.ProbeRef.Group,
		&entity.DowntimeEpisode.ProbeRef.Probe)
	if err != nil {
		return Downtime30sEntity{}, fmt.Errorf("row to Downtime30sEntity: %v", err)
	}

	// Assertion
	if rows.Next() {
		log.Errorf("Not consistent 30s data: more than one record selected for ts=%d, group='%s', probe='%s'", downtime.TimeSlot, downtime.ProbeRef.Group, downtime.ProbeRef.Probe)
	}

	return entity, nil
}

func (d *Downtime30sDao) ListForRange(start int64, end int64, group string, probe string) ([]Downtime30sEntity, error) {
	rows, err := d.DbCtx.StmtRunner().Query(SelectDowntime30SecByTimeslotRange, start, end, group, probe)
	if err != nil {
		return nil, fmt.Errorf("select for range: %v", err)
	}
	defer rows.Close()

	var res = make([]Downtime30sEntity, 0)
	for rows.Next() {
		var entity = Downtime30sEntity{}
		err := rows.Scan(&entity.Rowid,
			&entity.DowntimeEpisode.TimeSlot,
			&entity.DowntimeEpisode.SuccessSeconds,
			&entity.DowntimeEpisode.FailSeconds,
			&entity.DowntimeEpisode.Unknown,
			&entity.DowntimeEpisode.NoData,
			&entity.DowntimeEpisode.ProbeRef.Group,
			&entity.DowntimeEpisode.ProbeRef.Probe)
		if err != nil {
			return nil, fmt.Errorf("row to Downtime30sEntity: %v", err)
		}
		res = append(res, entity)
	}

	return res, nil
}

func (d *Downtime30sDao) ListGroupProbe() ([]checks.ProbeRef, error) {
	rows, err := d.DbCtx.StmtRunner().Query(SelectDowntime30SecGroupProbe)
	if err != nil {
		return nil, fmt.Errorf("select group and probe: %v", err)
	}
	defer rows.Close()

	var res = make([]checks.ProbeRef, 0)
	for rows.Next() {
		var ref = checks.ProbeRef{}
		err := rows.Scan(&ref.Group, &ref.Probe)
		if err != nil {
			return nil, fmt.Errorf("row to ProbeRef: %v", err)
		}
		res = append(res, ref)
	}

	return res, nil
}

func (d *Downtime30sDao) Stats() ([]string, error) {
	rows, err := d.DbCtx.StmtRunner().Query(SelectDowntime30SecStats)
	if err != nil {
		return nil, fmt.Errorf("select stats: %v", err)
	}
	defer rows.Close()

	var stats = []string{}
	for rows.Next() {
		var startTime int64
		var count int64
		rows.Scan(&startTime, &count)
		stats = append(stats, fmt.Sprintf("%d %d", startTime, count))
	}

	return stats, nil
}

func (d *Downtime30sDao) SaveBatch(downtimes []checks.DowntimeEpisode) error {
	for _, downtime := range downtimes {
		_, err := d.DbCtx.StmtRunner().Exec(InsertDowntime30Sec,
			downtime.TimeSlot,
			downtime.SuccessSeconds,
			downtime.FailSeconds,
			downtime.Unknown,
			downtime.NoData,
			downtime.ProbeRef.Group,
			downtime.ProbeRef.Probe)
		if err != nil {
			return err
		}
	}
	return nil
}

func (d *Downtime30sDao) Insert(downtime checks.DowntimeEpisode) error {
	_, err := d.DbCtx.StmtRunner().Exec(InsertDowntime30Sec,
		downtime.TimeSlot,
		downtime.SuccessSeconds,
		downtime.FailSeconds,
		downtime.Unknown,
		downtime.NoData,
		downtime.ProbeRef.Group,
		downtime.ProbeRef.Probe)
	return err
}

func (d *Downtime30sDao) Update(rowid int64, downtime checks.DowntimeEpisode) error {
	_, err := d.DbCtx.StmtRunner().Exec(UpdateDowntime30SecById,
		downtime.SuccessSeconds,
		downtime.FailSeconds,
		downtime.Unknown,
		downtime.NoData,
		rowid)
	if err != nil {
		return err
	}
	return nil
}

func (d *Downtime30sDao) DeleteEarlierThen(tm int64) error {
	_, err := d.DbCtx.StmtRunner().Exec(DeleteDowntime30SecByEarlierTimestamp, tm)
	if err != nil {
		return err
	}
	return nil
}
