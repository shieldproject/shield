package db

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/jhunt/go-log"
)

type Store struct {
	UUID       string `json:"uuid"    mbus:"uuid"`
	TenantUUID string `json:"-"       mbus:"tenant_uuid"`
	Name       string `json:"name"    mbus:"name"`
	Summary    string `json:"summary" mbus:"summary"`
	Agent      string `json:"agent"   mbus:"agent"`
	Plugin     string `json:"plugin"  mbus:"plugin"`
	Global     bool   `json:"global"  mbus:"global"`
	Healthy    bool   `json:"healthy" mbus:"healthy"`

	DailyIncrease int64 `json:"daily_increase" mbus:"daily_increase"`
	StorageUsed   int64 `json:"storage_used"   mbus:"storage_used"`
	Threshold     int64 `json:"threshold"      mbus:"threshold"`
	ArchiveCount  int   `json:"archive_count"  mbus:"archive_count"`

	Config map[string]interface{} `json:"config,omitempty" mbus:"config"`

	LastTestTaskUUID string `json:"last_test_task_uuid"`
}

type StoreStats struct {
	DailyIncrease int64 `json:"daily_increase"`
	StorageUsed   int64 `json:"storage_used"`
	ArchiveCount  int   `json:"archive_count"`
}

func (store *Store) Configuration(db *DB, private bool) ([]ConfigItem, error) {
	if store.Config == nil {
		return nil, nil
	}

	meta, err := db.GetAgentPluginMetadata(store.Agent, store.Plugin)
	if meta == nil || err != nil {
		return nil, err
	}

	return DisplayableConfig("store", meta, store.Config, private), nil
}

type StoreFilter struct {
	UUID       string
	SkipUsed   bool
	SkipUnused bool
	SearchName string
	ForPlugin  string
	ForTenant  string
	ExactMatch bool
}

func (f *StoreFilter) Query() (string, []interface{}) {
	wheres := []string{}
	args := []interface{}{}

	if f.UUID != "" {
		if f.ExactMatch {
			wheres = append(wheres, "s.uuid = ?")
			args = append(args, f.UUID)
		} else {
			wheres = append(wheres, "s.uuid LIKE ? ESCAPE '/'")
			args = append(args, PatternPrefix(f.UUID))
		}
	}

	if f.SearchName != "" {
		if f.ExactMatch {
			wheres = append(wheres, "s.name = ?")
			args = append(args, f.SearchName)
		} else {
			wheres = append(wheres, "s.name LIKE ?")
			args = append(args, Pattern(f.SearchName))
		}
	}

	if len(wheres) == 0 {
		wheres = []string{"1"}
	} else if len(wheres) > 1 {
		wheres = []string{strings.Join(wheres, " OR ")}
	}

	if f.ForPlugin != "" {
		wheres = append(wheres, "s.plugin = ?")
		args = append(args, f.ForPlugin)
	}
	if f.ForTenant != "" {
		wheres = append(wheres, "s.tenant_uuid = ?")
		args = append(args, f.ForTenant)
	}

	if !f.SkipUsed && !f.SkipUnused {
		return `
		   SELECT s.uuid, s.name, s.summary, s.agent,
		          s.plugin, s.endpoint, s.tenant_uuid, -1 AS n,
		          s.daily_increase,
		          s.storage_used, s.archive_count, s.threshold,
		          s.healthy, s.last_test_task_uuid
		     FROM stores s
		    WHERE ` + strings.Join(wheres, " AND ") + `
		 ORDER BY s.name, s.uuid ASC`, args
	}

	having := `HAVING COUNT(j.uuid) = 0`
	if f.SkipUnused {
		having = `HAVING COUNT(j.uuid) > 0`
	}

	return `
	   SELECT DISTINCT s.uuid, s.name, s.summary, s.agent,
	                   s.plugin, s.endpoint, s.tenant_uuid, COUNT(j.uuid) AS n,
	                   s.daily_increase,
	                   s.storage_used, s.archive_count, s.threshold,
	                   s.healthy, s.last_test_task_uuid
	              FROM stores s
	         LEFT JOIN jobs j
	                ON j.store_uuid = s.uuid
	             WHERE ` + strings.Join(wheres, " AND ") + `
	             GROUP BY s.uuid
	             ` + having + `
	          ORDER BY s.name, s.uuid ASC`, args
}

func (db *DB) GetAllStores(filter *StoreFilter) ([]*Store, error) {
	if filter == nil {
		filter = &StoreFilter{}
	}

	l := []*Store{}
	query, args := filter.Query()
	r, err := db.Query(query, args...)
	if err != nil {
		return l, err
	}
	defer r.Close()

	for r.Next() {
		store := &Store{}

		var (
			rawconfig              []byte
			n                      int
			daily, used, threshold *int64
			archives               *int
			healthy                bool
			ltt                    sql.NullString
		)
		if err = r.Scan(&store.UUID, &store.Name, &store.Summary, &store.Agent,
			&store.Plugin, &rawconfig, &store.TenantUUID, &n, &daily,
			&used, &archives, &threshold,
			&healthy, &ltt); err != nil {
			return l, err
		}
		store.Healthy = healthy
		store.Global = store.TenantUUID == GlobalTenantUUID
		if ltt.Valid {
			store.LastTestTaskUUID = ltt.String
		}
		if daily != nil {
			store.DailyIncrease = *daily
		}
		if archives != nil {
			store.ArchiveCount = *archives
		}
		if used != nil {
			store.StorageUsed = *used
		}
		if threshold != nil {
			store.Threshold = *threshold
		}
		if rawconfig != nil {
			if err := json.Unmarshal(rawconfig, &store.Config); err != nil {
				log.Warnf("failed to parse storage system endpoint json '%s': %s", rawconfig, err)
			}
		}

		l = append(l, store)
	}

	return l, nil
}

func (db *DB) GetStore(id string) (*Store, error) {
	r, err := db.Query(`
	       SELECT s.uuid, s.name, s.summary, s.agent,
	              s.plugin, s.endpoint, s.tenant_uuid,
	              s.daily_increase,
	              s.storage_used, s.archive_count, s.threshold,
	              s.healthy, s.last_test_task_uuid

	         FROM stores s
	    LEFT JOIN jobs j ON j.store_uuid = s.uuid

	        WHERE s.uuid = ?`, id)
	if err != nil {
		return nil, err
	}
	defer r.Close()

	if !r.Next() {
		return nil, nil
	}

	store := &Store{}
	var (
		rawconfig              []byte
		daily, used, threshold *int64
		archives               *int
		healthy                bool
		ltt                    sql.NullString
	)
	if err = r.Scan(&store.UUID, &store.Name, &store.Summary, &store.Agent,
		&store.Plugin, &rawconfig, &store.TenantUUID, &daily,
		&used, &archives, &threshold,
		&healthy, &ltt); err != nil {
		return nil, err
	}
	store.Global = store.TenantUUID == GlobalTenantUUID
	store.Healthy = healthy
	if ltt.Valid {
		store.LastTestTaskUUID = ltt.String
	}
	if daily != nil {
		store.DailyIncrease = *daily
	}
	if archives != nil {
		store.ArchiveCount = *archives
	}
	if used != nil {
		store.StorageUsed = *used
	}
	if threshold != nil {
		store.Threshold = *threshold
	}
	if rawconfig != nil {
		if err := json.Unmarshal(rawconfig, &store.Config); err != nil {
			log.Warnf("failed to parse storage system endpoint json '%s': %s", rawconfig, err)
		}
	}

	return store, nil
}

func (db *DB) CreateStore(store *Store) (*Store, error) {
	rawconfig, err := json.Marshal(store.Config)
	if err != nil {
		return nil, fmt.Errorf("unable to marshal storage endpoint configs: %s", err)
	}

	store.UUID = RandomID()
	err = db.Exec(`
	   INSERT INTO stores (uuid, tenant_uuid, name, summary, agent,
	                       plugin, endpoint,
	                       threshold, healthy, last_test_task_uuid)
	               VALUES (?, ?, ?, ?, ?,
	                       ?, ?,
	                       ?, ?, ?)`,
		store.UUID, store.TenantUUID, store.Name, store.Summary, store.Agent,
		store.Plugin, string(rawconfig),
		store.Threshold, store.Healthy, store.LastTestTaskUUID)
	if err != nil {
		return nil, err
	}

	if store.TenantUUID == GlobalTenantUUID {
		db.sendCreateObjectEvent(store, "*")
	} else {
		db.sendCreateObjectEvent(store, "tenant:"+store.TenantUUID)
	}
	return store, nil
}

func (db *DB) UpdateStore(store *Store) error {
	rawconfig, err := json.Marshal(store.Config)
	if err != nil {
		return fmt.Errorf("unable to marshal storage endpoint configs: %s", err)
	}

	err = db.Exec(`
	   UPDATE stores
	      SET name                    = ?,
	          summary                 = ?,
	          agent                   = ?,
	          plugin                  = ?,
	          endpoint                = ?,
	          daily_increase          = ?,
	          archive_count           = ?,
	          storage_used            = ?,
	          threshold               = ?,
	          healthy                 = ?,
	          last_test_task_uuid     = ?
	    WHERE uuid = ?`,
		store.Name, store.Summary, store.Agent, store.Plugin,
		string(rawconfig),
		store.DailyIncrease, store.ArchiveCount, store.StorageUsed,
		store.Threshold, store.Healthy, store.LastTestTaskUUID,
		store.UUID)
	if err != nil {
		return err
	}

	update, err := db.GetStore(store.UUID)
	if err != nil {
		return err
	}
	if update == nil {
		return fmt.Errorf("unable to retrieve store %s after update", store.UUID)
	}

	db.sendUpdateObjectEvent(store, "tenant:"+store.TenantUUID)
	return nil
}

func (db *DB) DeleteStore(id string) (bool, error) {
	store, err := db.GetStore(id)
	if err != nil {
		return false, err
	}

	if store == nil {
		/* already deleted */
		return true, nil
	}

	r, err := db.Query(`SELECT COUNT(uuid) FROM jobs WHERE jobs.store_uuid = ?`, store.UUID)
	if err != nil {
		return false, err
	}
	defer r.Close()

	if !r.Next() {
		/* already deleted (temporal anomaly detected) */
		return true, nil
	}

	var numJobs int
	if err = r.Scan(&numJobs); err != nil {
		return false, err
	}
	if numJobs < 0 {
		return false, fmt.Errorf("Store %s is in used by %d (negative) Jobs", id, numJobs)
	}
	if numJobs > 0 {
		return false, nil
	}
	r.Close()

	err = db.Exec(`DELETE FROM stores WHERE uuid = ?`, store.UUID)
	if err != nil {
		return false, err
	}

	db.sendDeleteObjectEvent(store, "tenant:"+store.TenantUUID)
	return true, nil
}

func (store Store) ConfigJSON() (string, error) {
	b, err := json.Marshal(store.Config)
	if err != nil {
		return "", err
	}
	return string(b), err
}

func (db *DB) CleanStores() error {
	return db.Exec(`
	   DELETE FROM stores
	         WHERE uuid IN (SELECT uuid
	                          FROM stores s WHERE tenant_uuid = ''
	                           AND (SELECT COUNT(*)
	                                  FROM archives a
	                                 WHERE a.store_uuid = s.uuid
	                                   AND a.status != 'purged') = 0)`)
}
