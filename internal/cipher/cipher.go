// Package cipher is the only native database boundary. It never exposes native
// errors (which can include SQL or values), and owns all statement lifetimes.
package cipher

/*
#cgo CFLAGS: -DSQLITE_HAS_CODEC -DSQLITE_EXTRA_INIT=sqlcipher_extra_init -DSQLITE_EXTRA_SHUTDOWN=sqlcipher_extra_shutdown -DSQLITE_TEMP_STORE=3 -DSQLITE_OMIT_LOAD_EXTENSION -DSQLITE_DQS=0 -DSQLITE_THREADSAFE=1 -DSQLITE_DEFAULT_MEMSTATUS=0 -fvisibility=hidden
#cgo linux CFLAGS: -DSQLCIPHER_CRYPTO_OPENSSL
#cgo linux LDFLAGS: -lcrypto -lm -ldl -lpthread
#cgo darwin CFLAGS: -DSQLCIPHER_CRYPTO_CC -mmacosx-version-min=15.0
#cgo darwin LDFLAGS: -framework Security -framework CoreFoundation -mmacosx-version-min=15.0
#include "sqlite3.h"
#include <stdlib.h>
#include <string.h>
static int pass_bind(sqlite3_stmt *s, int i, const void *p, int n) {
  return sqlite3_bind_text(s, i, (const char*)p, n, SQLITE_TRANSIENT);
}
static void pass_clear(void *p, size_t n) { volatile unsigned char *q=p; while(n--) *q++=0; }
*/
import "C"
import (
	"errors"
	"math"
	"runtime"
	"strings"
	"unsafe"
)

var ErrDatabase = errors.New("database operation failed")
var ErrKey = errors.New("cannot unlock: wrong password, unsupported format, or damaged vault")
var ErrCipher = errors.New("verified SQLCipher 4.20.0 encryption is required")

type Value struct {
	Kind int
	Text string
	Int  int64
}

const (
	Null    = 0
	Text    = 1
	Integer = 2
)

func S(s string) Value { return Value{Kind: Text, Text: s} }
func I(n int64) Value  { return Value{Kind: Integer, Int: n} }
func N() Value         { return Value{} }
func Ptr(s *string) Value {
	if s == nil {
		return N()
	}
	return S(*s)
}
func (v Value) StringPtr() *string {
	if v.Kind != Text {
		return nil
	}
	s := v.Text
	return &s
}

type DB struct{ p *C.sqlite3 }

// Open never creates files. Callers exclusively create and secure fresh files.
func Open(path string, key []byte, profile int, readOnly bool) (*DB, error) {
	if len(key) == 0 || len(key) > 1024*1024 || strings.ContainsRune(path, 0) {
		return nil, ErrKey
	}
	cp := C.CString(path)
	defer C.free(unsafe.Pointer(cp))
	flags := C.int(C.SQLITE_OPEN_READWRITE | C.SQLITE_OPEN_FULLMUTEX)
	if readOnly {
		flags = C.SQLITE_OPEN_READONLY | C.SQLITE_OPEN_FULLMUTEX
	}
	d := &DB{}
	if C.sqlite3_open_v2(cp, &d.p, flags, nil) != C.SQLITE_OK {
		d.Close()
		return nil, ErrDatabase
	}
	ok := false
	defer func() {
		if !ok {
			d.Close()
		}
	}()
	rows, e := d.Query("PRAGMA cipher_version")
	if e != nil || len(rows) != 1 || len(rows[0]) != 1 || !strings.HasPrefix(rows[0][0].Text, "4.20.0 ") {
		return nil, ErrCipher
	}
	b := C.CBytes(key)
	defer func() { C.pass_clear(b, C.size_t(len(key))); C.free(b) }()
	if C.sqlite3_key(d.p, b, C.int(len(key))) != C.SQLITE_OK {
		return nil, ErrKey
	}
	runtime.KeepAlive(key)
	if profile != 3 && profile != 4 {
		return nil, ErrCipher
	}
	if _, e = d.Exec("PRAGMA cipher_compatibility = " + map[int]string{3: "3", 4: "4"}[profile]); e != nil {
		return nil, e
	}
	for _, sql := range []string{"PRAGMA cipher_memory_security=ON", "PRAGMA temp_store=MEMORY", "PRAGMA busy_timeout=2000", "PRAGMA trusted_schema=OFF"} {
		if _, e = d.Exec(sql); e != nil {
			return nil, e
		}
	}
	status, e := d.Query("PRAGMA cipher_status")
	if e != nil || len(status) != 1 || status[0][0].Kind != Text || status[0][0].Text != "1" {
		return nil, ErrCipher
	}
	// Read back the effective profile on every handle. Unknown pragmas or an
	// accidental default change must fail before any sensitive read or write.
	settings, e := d.Query("PRAGMA cipher_settings")
	expected := []string{
		"PRAGMA kdf_iter = 256000;", "PRAGMA cipher_page_size = 4096;",
		"PRAGMA cipher_use_hmac = 1;", "PRAGMA cipher_plaintext_header_size = 0;",
		"PRAGMA cipher_hmac_algorithm = HMAC_SHA512;", "PRAGMA cipher_kdf_algorithm = PBKDF2_HMAC_SHA512;",
	}
	if profile == 3 {
		expected[0] = "PRAGMA kdf_iter = 64000;"
		expected[1] = "PRAGMA cipher_page_size = 1024;"
		expected[4] = "PRAGMA cipher_hmac_algorithm = HMAC_SHA1;"
		expected[5] = "PRAGMA cipher_kdf_algorithm = PBKDF2_HMAC_SHA1;"
	}
	if e != nil || len(settings) != len(expected) {
		return nil, ErrCipher
	}
	for i, row := range settings {
		if len(row) != 1 || row[0].Kind != Text || row[0].Text != expected[i] {
			return nil, ErrCipher
		}
	}
	if _, e = d.Query("SELECT count(*) FROM sqlite_schema"); e != nil {
		return nil, ErrKey
	}
	if !readOnly {
		for _, sql := range []string{"PRAGMA journal_mode=DELETE", "PRAGMA synchronous=FULL", "PRAGMA fullfsync=ON", "PRAGMA foreign_keys=ON", "PRAGMA secure_delete=ON"} {
			if _, e = d.Exec(sql); e != nil {
				return nil, e
			}
		}
	}
	C.sqlite3_limit(d.p, C.SQLITE_LIMIT_LENGTH, 8*1024*1024)
	C.sqlite3_limit(d.p, C.SQLITE_LIMIT_SQL_LENGTH, 64*1024)
	ok = true
	return d, nil
}
func (d *DB) Close() error {
	if d == nil || d.p == nil {
		return nil
	}
	if C.sqlite3_close(d.p) != C.SQLITE_OK {
		return ErrDatabase
	}
	d.p = nil
	return nil
}
func (d *DB) Interrupt() {
	if d != nil && d.p != nil {
		C.sqlite3_interrupt(d.p)
	}
}
func (d *DB) statement(sql string, args []Value) (*C.sqlite3_stmt, error) {
	if d == nil || d.p == nil || len(sql) > math.MaxInt32 {
		return nil, ErrDatabase
	}
	cs := C.CString(sql)
	defer C.free(unsafe.Pointer(cs))
	var st *C.sqlite3_stmt
	if C.sqlite3_prepare_v2(d.p, cs, C.int(len(sql)), &st, nil) != C.SQLITE_OK {
		return nil, ErrDatabase
	}
	if st == nil {
		return nil, ErrDatabase
	}
	fail := func() (*C.sqlite3_stmt, error) { C.sqlite3_finalize(st); return nil, ErrDatabase }
	if int(C.sqlite3_bind_parameter_count(st)) != len(args) {
		return fail()
	}
	for i, v := range args {
		var rc C.int
		switch v.Kind {
		case Null:
			rc = C.sqlite3_bind_null(st, C.int(i+1))
		case Integer:
			rc = C.sqlite3_bind_int64(st, C.int(i+1), C.sqlite3_int64(v.Int))
		case Text:
			if len(v.Text) > 8*1024*1024 {
				return fail()
			}
			var b unsafe.Pointer
			if len(v.Text) == 0 {
				b = C.malloc(1)
			} else {
				b = C.CBytes([]byte(v.Text))
			}
			rc = C.pass_bind(st, C.int(i+1), b, C.int(len(v.Text)))
			C.pass_clear(b, C.size_t(len(v.Text)))
			C.free(b)
		default:
			return fail()
		}
		if rc != C.SQLITE_OK {
			return fail()
		}
	}
	return st, nil
}
func (d *DB) Exec(sql string, args ...Value) (int64, error) {
	st, e := d.statement(sql, args)
	if e != nil {
		return 0, e
	}
	defer C.sqlite3_finalize(st)
	for {
		rc := C.sqlite3_step(st)
		if rc == C.SQLITE_DONE {
			return int64(C.sqlite3_changes(d.p)), nil
		}
		if rc != C.SQLITE_ROW {
			return 0, ErrDatabase
		}
	}
}
func (d *DB) Query(sql string, args ...Value) ([][]Value, error) {
	st, e := d.statement(sql, args)
	if e != nil {
		return nil, e
	}
	defer C.sqlite3_finalize(st)
	rows := [][]Value{}
	for {
		rc := C.sqlite3_step(st)
		if rc == C.SQLITE_DONE {
			return rows, nil
		}
		if rc != C.SQLITE_ROW {
			return nil, ErrDatabase
		}
		row := make([]Value, int(C.sqlite3_column_count(st)))
		for i := range row {
			switch C.sqlite3_column_type(st, C.int(i)) {
			case C.SQLITE_NULL:
				row[i] = N()
			case C.SQLITE_INTEGER:
				row[i] = I(int64(C.sqlite3_column_int64(st, C.int(i))))
			case C.SQLITE_TEXT:
				p := C.sqlite3_column_text(st, C.int(i))
				n := C.sqlite3_column_bytes(st, C.int(i))
				row[i] = S(string(C.GoBytes(unsafe.Pointer(p), n)))
			default:
				return nil, ErrDatabase
			}
		}
		rows = append(rows, row)
		if len(rows) > 100000 {
			return nil, ErrDatabase
		}
	}
}
func (d *DB) Transaction(fn func() error) error {
	if _, e := d.Exec("BEGIN IMMEDIATE"); e != nil {
		return e
	}
	committed := false
	defer func() {
		if !committed {
			_, _ = d.Exec("ROLLBACK")
		}
	}()
	if e := fn(); e != nil {
		return e
	}
	if _, e := d.Exec("COMMIT"); e != nil {
		return e
	}
	committed = true
	return nil
}
func (d *DB) Integrity() error {
	r, e := d.Query("PRAGMA cipher_integrity_check")
	if e != nil || len(r) != 0 {
		return ErrDatabase
	}
	r, e = d.Query("PRAGMA integrity_check")
	if e != nil || len(r) != 1 || r[0][0].Text != "ok" {
		return ErrDatabase
	}
	return nil
}
