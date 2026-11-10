package vault

import (
	"encoding/hex"
	"errors"
	"reflect"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/ssemakov/pass-rework/internal/cipher"
)

var ErrFormat = errors.New("This storage format is not supported")
var ErrLegacy = errors.New("This is a V1 storage file. Migrate it to use it with PassV2")
var ErrConflict = errors.New("entry changed; reload it before saving")
var ErrInput = errors.New("invalid entry or password")

const columns = "id,name,login,password,url,created_at,updated_at"
const createItems = "CREATE TABLE login_items(id INTEGER PRIMARY KEY AUTOINCREMENT, name varchar(255), login varchar(255), password varchar(255), url varchar(255), created_at INTEGER, updated_at INTEGER)"

const createVersions = "CREATE TABLE item_versions(id INTEGER PRIMARY KEY REFERENCES login_items(id) ON DELETE CASCADE,revision INTEGER NOT NULL CHECK(revision>0))"
const createMeta = "CREATE TABLE pass_meta(format TEXT NOT NULL,uuid TEXT NOT NULL)"

type Entry struct {
	ID          string  `json:"id"`
	Name        *string `json:"name"`
	Login       *string `json:"login"`
	Password    *string `json:"password,omitempty"`
	URL         *string `json:"url"`
	CreatedAt   *string `json:"createdAt"`
	UpdatedAt   *string `json:"updatedAt"`
	Revision    string  `json:"revision"`
	HasPassword bool    `json:"hasPassword"`
}
type Input struct {
	Name     *string `json:"name"`
	Login    *string `json:"login"`
	Password *string `json:"password"`
	URL      *string `json:"url"`
}
type Patch struct {
	Name            *string `json:"name"`
	Login           *string `json:"login"`
	URL             *string `json:"url"`
	Password        *string `json:"password"`
	ReplacePassword bool    `json:"replacePassword"`
	Revision        string  `json:"revision"`
}

func (i Input) validate() error {
	for _, s := range []*string{i.Name, i.Login, i.Password, i.URL} {
		if s != nil && (len(*s) > 1024*1024 || !utf8.ValidString(*s)) {
			return ErrInput
		}
	}
	return nil
}
func integer(s string) (int64, error) {
	n, e := strconv.ParseInt(s, 10, 64)
	if e != nil || strconv.FormatInt(n, 10) != s {
		return 0, ErrInput
	}
	return n, nil
}
func asIntPtr(v cipher.Value) *string {
	if v.Kind == cipher.Null {
		return nil
	}
	s := strconv.FormatInt(v.Int, 10)
	return &s
}
func fromRow(r []cipher.Value) Entry {
	return Entry{strconv.FormatInt(r[0].Int, 10), r[1].StringPtr(), r[2].StringPtr(), r[3].StringPtr(), r[4].StringPtr(), asIntPtr(r[5]), asIntPtr(r[6]), strconv.FormatInt(r[7].Int, 10), r[3].Kind == cipher.Text && len(r[3].Text) > 0}
}
func initialise(d *cipher.DB) error {
	return d.Transaction(func() error {
		for _, s := range []string{createItems, createVersions, createMeta, "PRAGMA user_version=1"} {
			if _, e := d.Exec(s); e != nil {
				return e
			}
		}
		_, e := d.Exec("INSERT INTO pass_meta VALUES(?,?)", cipher.S("pass-rework-v1"), cipher.S(randomID()))
		return e
	})
}
func schema(d *cipher.DB, current bool) error {
	rows, e := d.Query("SELECT type,name,tbl_name,sql FROM sqlite_schema WHERE name != 'sqlite_sequence' ORDER BY name")
	if e != nil {
		return e
	}
	expected := []string{"login_items"}
	if current {
		expected = []string{"item_versions", "login_items", "pass_meta"}
	}
	if len(rows) != len(expected) {
		return ErrFormat
	}
	for i, r := range rows {
		if r[0].Text != "table" || r[1].Text != expected[i] || r[2].Text != expected[i] {
			return ErrFormat
		}
	}
	if current {
		definitions := []string{createVersions, createItems, createMeta}
		for i, r := range rows {
			if r[3].Kind != cipher.Text || r[3].Text != definitions[i] {
				return ErrFormat
			}
		}
		violations, e := d.Query("PRAGMA foreign_key_check")
		if e != nil || len(violations) != 0 {
			return ErrFormat
		}
	}
	fields := []string{"id", "name", "login", "password", "url", "created_at", "updated_at"}
	info, e := d.Query("PRAGMA table_xinfo(login_items)")
	if e != nil || len(info) != 7 {
		return ErrFormat
	}
	for i, f := range fields {
		if info[i][1].Text != f {
			return ErrFormat
		}
		typ := strings.ToUpper(info[i][2].Text)
		if i == 0 || i >= 5 {
			if typ != "INTEGER" {
				return ErrFormat
			}
		} else if typ != "VARCHAR(255)" && typ != "TEXT" {
			return ErrFormat
		}
	}
	if info[0][5].Int != 1 {
		return ErrFormat
	}
	// SQLite's dynamic typing is checked for every value, never silently coerced.
	data, e := d.Query("SELECT " + columns + " FROM login_items ORDER BY id")
	if e != nil {
		return e
	}
	for _, r := range data {
		if r[0].Kind != cipher.Integer {
			return ErrFormat
		}
		for i := 1; i < 7; i++ {
			if r[i].Kind == cipher.Null {
				continue
			}
			if i <= 4 && (r[i].Kind != cipher.Text || !utf8.ValidString(r[i].Text)) || i >= 5 && r[i].Kind != cipher.Integer {
				return ErrFormat
			}
		}
	}
	if current {
		extras, e := d.Query("SELECT count(*) FROM item_versions LEFT JOIN login_items USING(id) WHERE login_items.id IS NULL")
		if e != nil || extras[0][0].Int != 0 {
			return ErrFormat
		}
		v, e := d.Query("PRAGMA user_version")
		if e != nil || len(v) != 1 || v[0][0].Int != 1 {
			return ErrFormat
		}
		m, e := d.Query("SELECT format,uuid FROM pass_meta")
		if e != nil || len(m) != 1 || m[0][0].Text != "pass-rework-v1" || len(m[0][1].Text) != 32 {
			return ErrFormat
		}
		if _, e = hex.DecodeString(m[0][1].Text); e != nil {
			return ErrFormat
		}
		v, e = d.Query("SELECT count(*) FROM login_items LEFT JOIN item_versions USING(id) WHERE revision IS NULL OR typeof(revision)!='integer' OR revision<1")
		if e != nil || v[0][0].Int != 0 {
			return ErrFormat
		}
	}
	return nil
}
func list(d *cipher.DB, q string) ([]Entry, error) {
	if len(q) > 4096 {
		return nil, ErrInput
	}
	// Presence alone reproduces the legacy empty-field UI without prefetching
	// any password. BLOB length preserves embedded-NUL text semantics.
	rows, e := d.Query("SELECT i.id,i.name,i.login,NULL,i.url,i.created_at,i.updated_at,v.revision,i.password IS NOT NULL AND length(CAST(i.password AS BLOB))>0 FROM login_items i JOIN item_versions v USING(id) WHERE instr(lower(coalesce(i.name,'')),lower(?))>0 OR instr(lower(coalesce(i.url,'')),lower(?))>0 OR instr(lower(coalesce(i.login,'')),lower(?))>0 ORDER BY coalesce(i.name,''),i.id", cipher.S(q), cipher.S(q), cipher.S(q))
	if e != nil {
		return nil, e
	}
	out := make([]Entry, 0, len(rows))
	for _, r := range rows {
		entry := fromRow(r)
		entry.HasPassword = r[8].Int != 0
		out = append(out, entry)
	}
	return out, nil
}
func get(d *cipher.DB, id string) (Entry, error) {
	n, e := integer(id)
	if e != nil {
		return Entry{}, e
	}
	r, e := d.Query("SELECT i.id,i.name,i.login,i.password,i.url,i.created_at,i.updated_at,v.revision FROM login_items i JOIN item_versions v USING(id) WHERE i.id=?", cipher.I(n))
	if e != nil {
		return Entry{}, e
	}
	if len(r) != 1 {
		return Entry{}, ErrConflict
	}
	return fromRow(r[0]), nil
}
func create(d *cipher.DB, i Input) (Entry, error) {
	if e := i.validate(); e != nil {
		return Entry{}, e
	}
	var id string
	e := d.Transaction(func() error {
		now := time.Now().UnixMilli()
		if _, e := d.Exec("INSERT INTO login_items(name,login,password,url,created_at,updated_at) VALUES(?,?,?,?,?,?)", cipher.Ptr(i.Name), cipher.Ptr(i.Login), cipher.Ptr(i.Password), cipher.Ptr(i.URL), cipher.I(now), cipher.I(now)); e != nil {
			return e
		}
		r, e := d.Query("SELECT last_insert_rowid()")
		if e != nil {
			return e
		}
		n := r[0][0].Int
		id = strconv.FormatInt(n, 10)
		_, e = d.Exec("INSERT INTO item_versions VALUES(?,1)", cipher.I(n))
		return e
	})
	if e != nil {
		return Entry{}, e
	}
	return get(d, id)
}
func update(d *cipher.DB, id string, p Patch) (Entry, error) {
	i := Input{p.Name, p.Login, p.Password, p.URL}
	if e := i.validate(); e != nil {
		return Entry{}, e
	}
	n, e := integer(id)
	if e != nil {
		return Entry{}, e
	}
	rev, e := integer(p.Revision)
	if e != nil || rev < 1 || rev == 1<<63-1 {
		return Entry{}, ErrInput
	}
	e = d.Transaction(func() error {
		changed, e := d.Exec("UPDATE item_versions SET revision=revision+1 WHERE id=? AND revision=?", cipher.I(n), cipher.I(rev))
		if e != nil {
			return e
		}
		if changed != 1 {
			return ErrConflict
		}
		sql := "UPDATE login_items SET name=?,login=?,url=?,updated_at=? WHERE id=?"
		args := []cipher.Value{cipher.Ptr(p.Name), cipher.Ptr(p.Login), cipher.Ptr(p.URL), cipher.I(time.Now().UnixMilli()), cipher.I(n)}
		if p.ReplacePassword {
			sql = "UPDATE login_items SET name=?,login=?,url=?,updated_at=?,password=? WHERE id=?"
			args = []cipher.Value{cipher.Ptr(p.Name), cipher.Ptr(p.Login), cipher.Ptr(p.URL), cipher.I(time.Now().UnixMilli()), cipher.Ptr(p.Password), cipher.I(n)}
		}
		_, e = d.Exec(sql, args...)
		return e
	})
	if e != nil {
		return Entry{}, e
	}
	return get(d, id)
}
func remove(d *cipher.DB, id, revision string) error {
	n, e := integer(id)
	if e != nil {
		return e
	}
	rev, e := integer(revision)
	if e != nil || rev < 1 {
		return ErrInput
	}
	return d.Transaction(func() error {
		c, e := d.Exec("DELETE FROM login_items WHERE id=? AND EXISTS(SELECT 1 FROM item_versions WHERE id=? AND revision=?)", cipher.I(n), cipher.I(n), cipher.I(rev))
		if e != nil {
			return e
		}
		if c != 1 {
			return ErrConflict
		}
		return nil
	})
}
func sequence(d *cipher.DB) ([]cipher.Value, error) {
	r, e := d.Query("SELECT seq FROM sqlite_sequence WHERE name='login_items'")
	if e != nil {
		return nil, e
	}
	if len(r) == 0 {
		return nil, nil
	}
	if len(r) != 1 || r[0][0].Kind != cipher.Integer {
		return nil, ErrFormat
	}
	return r[0], nil
}
func compare(a, b *cipher.DB, current bool) error {
	for _, s := range []string{"SELECT " + columns + " FROM login_items ORDER BY id", "SELECT seq FROM sqlite_sequence WHERE name='login_items'"} {
		x, e := a.Query(s)
		if e != nil {
			return e
		}
		y, e := b.Query(s)
		if e != nil || !reflect.DeepEqual(x, y) {
			return errors.New("every-entry verification failed")
		}
	}
	if current {
		for _, s := range []string{"SELECT id,revision FROM item_versions ORDER BY id", "SELECT format,uuid FROM pass_meta"} {
			x, e := a.Query(s)
			if e != nil {
				return e
			}
			y, e := b.Query(s)
			if e != nil || !reflect.DeepEqual(x, y) {
				return ErrFormat
			}
		}
	}
	return nil
}
