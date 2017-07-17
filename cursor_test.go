package fb

import (
	"io"
	"os"
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestNextRow(t *testing.T) {
	st := SuperTest{t}
	const SqlSelect = "SELECT * FROM RDB$DATABASE"

	os.Remove(TestFilename)

	conn, err := Create(TestConnectionString)
	if err != nil {
		t.Fatalf("Error creating database: %s", err)
	}
	defer conn.Drop()
	cursor, err := conn.Execute(SqlSelect)
	if err != nil {
		t.Fatalf("Error executing select statement: %s", err)
	}
	if cursor == nil {
		t.Fatal("Cursor should not be nil.")
	}
	if !cursor.Next() {
		t.Fatalf("Error in Next: %s", cursor.Err())
	}
	row := cursor.Row()
	st.Equal(4, len(row))
	st.Equal(nil, row[0])
	_, ok := row[1].(int16)
	if !ok {
		t.Errorf("Expected row[1] to be an int, got %v", reflect.TypeOf(row[1]))
	}
	st.Equal(nil, row[2])
	st.Equal("NONE", strings.TrimSpace(row[3].(string)))
}

var sqlSampleInsertData = `INSERT INTO TEST VALUES (
	1,
	1,
	'BINARY BLOB CONTENTS',
	123,
	1234,
	1234567890,
	123.24,
	123456789.24,
	'A',
	'ABCDEFGHIJKLMNOPQRSTUVWXYZ',
	'A',
	'ABCDEFGHIJKLMNOPQRSTUVWXYZ',
	'TEXT BLOB CONTENTS',
	'2013-10-10',
	'08:42:00',
	'2013-10-10 08:42:00',
	5.55,
	30303.33);`

func TestRowMap(t *testing.T) {
	st := SuperTest{t}
	os.Remove(TestFilename)

	conn, err := Create(TestConnectionString)
	if err != nil {
		t.Fatalf("Unexpected error creating database: %s", err)
	}
	defer conn.Drop()

	sqlInsert2 := "INSERT INTO TEST (ID, I32, F64) VALUES (2, null, null);"
	sqlSelect := "SELECT * FROM TEST;"
	dtExpected := time.Date(2013, 10, 10, 0, 0, 0, 0, conn.Location)
	tmExpected := time.Date(1970, 1, 1, 8, 42, 0, 0, conn.Location)
	tsExpected := time.Date(2013, 10, 10, 8, 42, 0, 0, conn.Location)

	if err = conn.ExecuteScript(sqlSampleSchema); err != nil {
		t.Fatalf("Error executing schema: %s", err)
	}

	if _, err = conn.Execute(sqlSampleInsertData); err != nil {
		t.Fatalf("Error executing insert: %s", err)
	}
	if _, err = conn.Execute(sqlInsert2); err != nil {
		t.Fatalf("Error executing insert: %s", err)
	}

	var cursor *Cursor
	if cursor, err = conn.Execute(sqlSelect); err != nil {
		t.Fatalf("Unexpected error in select: %s", err)
	}
	defer cursor.Close()

	if !cursor.Next() {
		t.Fatalf("Error in Next: %v", cursor.Err())
	}
	row := cursor.RowMap()

	st.Equal(int64(1), row["ID"])
	st.Equal(int32(1), row["FLAG"])
	st.Equal("BINARY BLOB CONTENTS", string(row["BINARY"].([]byte)))
	st.Equal(int32(123), row["I"])
	st.Equal(int32(1234), row["I32"])
	st.Equal(int64(1234567890), row["I64"])
	st.Equal(float32(123.24), row["F32"])
	st.Equal(123456789.24, row["F64"])
	st.Equal("A", row["C"])
	st.Equal("ABCDEFGHIJKLMNOPQRSTUVWXYZ", row["CS"])
	st.Equal("A", row["V"])
	st.Equal("ABCDEFGHIJKLMNOPQRSTUVWXYZ", row["VS"])
	st.Equal("TEXT BLOB CONTENTS", row["M"])
	st.Equal(dtExpected, row["DT"])
	st.Equal(tmExpected, row["TM"])
	st.Equal(tsExpected, row["TS"])
	st.Equal(5.55, row["N92"])
	st.Equal(30303.33, row["D92"])

	if !cursor.Next() {
		t.Fatalf("Error in Next: %v", cursor.Err())
	}
	row = cursor.RowMap()

	st.Nil(row["FLAG"])
	st.Nil(row["BINARY"])
	st.Nil(row["I"])
	st.Nil(row["I32"])
	st.Nil(row["I64"])
	st.Nil(row["F32"])
	st.Nil(row["F64"])
	st.Nil(row["C"])
	st.Nil(row["CS"])
	st.Nil(row["V"])
	st.Nil(row["VS"])
	st.Nil(row["M"])
	st.Nil(row["DT"])
	st.Nil(row["TM"])
	st.Nil(row["TS"])
}

func TestScan(t *testing.T) {
	st := SuperTest{t}
	os.Remove(TestFilename)

	conn, err := Create(TestConnectionString)
	if err != nil {
		t.Fatalf("Unexpected error creating database: %s", err)
	}
	defer conn.Drop()

	sqlInsert2 := "INSERT INTO TEST (ID, I32, F64) VALUES (2, null, null);"
	sqlSelect := "SELECT * FROM TEST;"
	dtExpected := time.Date(2013, 10, 10, 0, 0, 0, 0, conn.Location)
	tmExpected := time.Date(1970, 1, 1, 8, 42, 0, 0, conn.Location)
	tsExpected := time.Date(2013, 10, 10, 8, 42, 0, 0, conn.Location)

	if err = conn.ExecuteScript(sqlSampleSchema); err != nil {
		t.Fatalf("Error executing schema: %s", err)
	}

	if _, err = conn.Execute(sqlSampleInsertData); err != nil {
		t.Fatalf("Error executing insert: %s", err)
	}
	if _, err = conn.Execute(sqlInsert2); err != nil {
		t.Fatalf("Error executing insert: %s", err)
	}

	var cursor *Cursor
	if cursor, err = conn.Execute(sqlSelect); err != nil {
		t.Fatalf("Unexpected error in select: %s", err)
	}
	defer cursor.Close()

	if !cursor.Next() {
		t.Fatalf("Error in Next: %v", cursor.Err())
	}

	var (
		id     int64
		flag   bool
		binary []byte
		i      int
		i32    int32
		i64    int64
		f32    float32
		f64    float64
		c      string
		cs     string
		v      string
		vs     string
		m      string
		dt     time.Time
		tm     time.Time
		ts     time.Time
		n92    float64
		d92    float64
	)

	var (
		nflag   NullableBool
		nbinary NullableBytes
		ni      NullableInt32
		ni32    NullableInt32
		ni64    NullableInt64
		nf32    NullableFloat32
		nf64    NullableFloat64
		nc      NullableString
		ncs     NullableString
		nv      NullableString
		nvs     NullableString
		nm      NullableString
		ndt     NullableTime
		ntm     NullableTime
		nts     NullableTime
		nn92    NullableFloat64
		nd92    NullableFloat64
	)

	if err = cursor.Scan(&id, &flag, &binary, &i, &i32, &i64, &f32, &f64, &c, &cs, &v, &vs, &m, &dt, &tm, &ts, &n92, &d92); err != nil {
		t.Fatal(err)
	}

	st.Equal(int64(1), id)
	st.Equal(true, flag)
	st.Equal("BINARY BLOB CONTENTS", string(binary))
	st.Equal(123, i)
	st.Equal(int32(1234), i32)
	st.Equal(int64(1234567890), i64)
	st.Equal(float32(123.24), f32)
	st.Equal(123456789.24, f64)
	st.Equal("A", c)
	st.Equal("ABCDEFGHIJKLMNOPQRSTUVWXYZ", cs)
	st.Equal("A", v)
	st.Equal("ABCDEFGHIJKLMNOPQRSTUVWXYZ", vs)
	st.Equal("TEXT BLOB CONTENTS", m)
	st.Equal(dtExpected, dt)
	st.Equal(tmExpected, tm)
	st.Equal(tsExpected, ts)
	st.Equal(5.55, n92)
	st.Equal(30303.33, d92)

	if err = cursor.Scan(&id, &nflag, &nbinary, &ni, &ni32, &ni64, &nf32, &nf64, &nc, &ncs, &nv, &nvs, &nm, &ndt, &ntm, &nts, &nn92, &nd92); err != nil {
		t.Fatal(err)
	}

	st.False(nflag.Null)
	st.False(nbinary.Null)
	st.False(ni.Null)
	st.False(ni32.Null)
	st.False(ni64.Null)
	st.False(nf32.Null)
	st.False(nf64.Null)
	st.False(nc.Null)
	st.False(ncs.Null)
	st.False(nv.Null)
	st.False(nvs.Null)
	st.False(nm.Null)
	st.False(ndt.Null)
	st.False(ntm.Null)
	st.False(nts.Null)
	st.False(nn92.Null)
	st.False(nd92.Null)

	st.Equal(true, nflag.Value)
	st.Equal("BINARY BLOB CONTENTS", string(nbinary.Value))
	st.Equal(int32(123), ni.Value)
	st.Equal(int32(1234), ni32.Value)
	st.Equal(int64(1234567890), ni64.Value)
	st.Equal(float32(123.24), nf32.Value)
	st.Equal(123456789.24, nf64.Value)
	st.Equal("A", nc.Value)
	st.Equal("ABCDEFGHIJKLMNOPQRSTUVWXYZ", ncs.Value)
	st.Equal("A", nv.Value)
	st.Equal("ABCDEFGHIJKLMNOPQRSTUVWXYZ", nvs.Value)
	st.Equal("TEXT BLOB CONTENTS", nm.Value)
	st.Equal(dtExpected, ndt.Value)
	st.Equal(tmExpected, ntm.Value)
	st.Equal(tsExpected, nts.Value)
	st.Equal(5.55, nn92.Value)
	st.Equal(30303.33, nd92.Value)

	if !cursor.Next() {
		t.Fatalf("Error in Next: %v", cursor.Err())
	}
	if err = cursor.Scan(&id, &flag, &binary, &i, &i32, &i64, &f32, &f64, &c, &cs, &v, &vs, &m, &dt, &tm, &ts, &n92, &d92); err == nil {
		t.Fatal("Scan expected to fail")
	}

	if err = cursor.Scan(&id, &nflag, &nbinary, &ni, &ni32, &ni64, &nf32, &nf64, &nc, &ncs, &nv, &nvs, &nm, &ndt, &ntm, &nts, &nn92, &nd92); err != nil {
		t.Fatal(err)
	}
	st.True(nflag.Null)
	st.True(nbinary.Null)
	st.True(ni.Null)
	st.True(ni32.Null)
	st.True(ni64.Null)
	st.True(nf32.Null)
	st.True(nf64.Null)
	st.True(nc.Null)
	st.True(ncs.Null)
	st.True(nv.Null)
	st.True(nvs.Null)
	st.True(nm.Null)
	st.True(ndt.Null)
	st.True(ntm.Null)
	st.True(nts.Null)
	st.True(nn92.Null)
	st.True(nd92.Null)
}

func TestCursorFields(t *testing.T) {
	st := SuperTest{t}
	const SqlSelect = "SELECT * FROM RDB$DATABASE"

	os.Remove(TestFilename)

	conn, err := Create(TestConnectionString)
	if err != nil {
		t.Fatalf("Error creating database: %s", err)
	}
	defer conn.Drop()
	cursor, err := conn.Execute(SqlSelect)
	if err != nil {
		t.Fatalf("Error executing select statement: %s", err)
	}
	defer cursor.Close()

	cols := cursor.Columns
	st.Equal(4, len(cols))
	st.Equal("RDB$DESCRIPTION", cols[0].Name)
	st.Equal("RDB$RELATION_ID", cols[1].Name)
	st.Equal("RDB$SECURITY_CLASS", cols[2].Name)
	st.Equal("RDB$CHARACTER_SET_NAME", cols[3].Name)
}

func TestCursorFieldsLowercased(t *testing.T) {
	st := SuperTest{t}
	const SqlSelect = "SELECT * FROM RDB$DATABASE"

	os.Remove(TestFilename)

	conn, err := Create(TestConnectionString + "lowercase_names=true;")
	if err != nil {
		t.Fatalf("Error creating database: %s", err)
	}
	defer conn.Drop()
	cursor, err := conn.Execute(SqlSelect)
	if err != nil {
		t.Fatalf("Error executing select statement: %s", err)
	}
	defer cursor.Close()

	cols := cursor.Columns
	st.Equal(4, len(cols))
	st.Equal("rdb$description", cols[0].Name)
	st.Equal("rdb$relation_id", cols[1].Name)
	st.Equal("rdb$security_class", cols[2].Name)
	st.Equal("rdb$character_set_name", cols[3].Name)
}

func TestCursorFieldsMap(t *testing.T) {
	st := SuperTest{t}
	const SqlSelect = "SELECT * FROM RDB$DATABASE"

	os.Remove(TestFilename)

	conn, err := Create(TestConnectionString)
	if err != nil {
		t.Fatalf("Error creating database: %s", err)
	}
	defer conn.Drop()
	cursor, err := conn.Execute(SqlSelect)
	if err != nil {
		t.Fatalf("Error executing select statement: %s", err)
	}
	defer cursor.Close()

	cols := cursor.ColumnsMap
	st.Equal(4, len(cols))
	st.Equal(520, cols["RDB$DESCRIPTION"].TypeCode)
	st.Equal(500, cols["RDB$RELATION_ID"].TypeCode)
	st.Equal(452, cols["RDB$SECURITY_CLASS"].TypeCode)
	st.Equal(452, cols["RDB$CHARACTER_SET_NAME"].TypeCode)
}

func TestCursorFieldsWithAliasedFields(t *testing.T) {
	st := SuperTest{t}
	const SqlSelect = "SELECT RDB$DESCRIPTION DES, RDB$RELATION_ID REL, RDB$SECURITY_CLASS SEC, RDB$CHARACTER_SET_NAME FROM RDB$DATABASE"

	os.Remove(TestFilename)

	conn, err := Create(TestConnectionString)
	if err != nil {
		t.Fatalf("Error creating database: %s", err)
	}
	defer conn.Drop()
	cursor, err := conn.Execute(SqlSelect)
	if err != nil {
		t.Fatalf("Error executing select statement: %s", err)
	}
	defer cursor.Close()

	cols := cursor.Columns
	st.Equal(4, len(cols))
	st.Equal("DES", cols[0].Name)
	st.Equal("REL", cols[1].Name)
	st.Equal("SEC", cols[2].Name)
	st.Equal("RDB$CHARACTER_SET_NAME", cols[3].Name)
}

func TestNextAfterEnd(t *testing.T) {
	const SqlCreateGen = "create generator test_seq"
	const SqlSelectGen = "select gen_id(test_seq, 1) from rdb$database"

	os.Remove(TestFilename)

	conn, err := Create(TestConnectionString)
	if err != nil {
		t.Fatalf("Error creating database: %s", err)
	}
	defer conn.Drop()
	_, err = conn.Execute(SqlCreateGen)
	if err != nil {
		t.Fatalf("Error executing create statement: %s", err)
	}

	cursor, err := conn.Execute(SqlSelectGen)
	if err != nil {
		t.Fatalf("Error executing select statement: %s", err)
	}
	defer cursor.Close()

	if !cursor.Next() {
		t.Fatalf("Error in Next: %s", cursor.Err())
	}
	if cursor.Next() {
		t.Fatal("Next should not succeed.")
	}
	if cursor.Err() != io.EOF {
		t.Fatalf("Expecting io.EOF, got: %s", err)
	}
	if cursor.Next() {
		t.Fatal("Next should not succeed.")
	}
	err2, ok := cursor.Err().(*Error)
	if !ok {
		t.Fatalf("Expecting fb.Error, got: %s", reflect.TypeOf(cursor.Err()))
	}
	if err2.Message != "Cursor is past end of data." {
		t.Errorf("Unexpected error message: %s", err2.Message)
	}
}

func TestNextAfterEnd2(t *testing.T) {
	const SqlSelect = "select * from rdb$database"

	os.Remove(TestFilename)

	conn, err := Create(TestConnectionString)
	if err != nil {
		t.Fatalf("Error creating database: %s", err)
	}
	defer conn.Drop()

	cursor, err := conn.Execute(SqlSelect)
	if err != nil {
		t.Fatalf("Error executing select statement: %s", err)
	}
	defer cursor.Close()

	if !cursor.Next() {
		t.Fatalf("Error in Next: %s", cursor.Err())
	}
	if cursor.Next() {
		t.Fatal("Next should not succeed.")
	}
	if cursor.Err() != io.EOF {
		t.Fatalf("Expecting io.EOF, got: %s", err)
	}
	if cursor.Next() {
		t.Fatal("Next should not succeed.")
	}
	err2, ok := cursor.Err().(*Error)
	if !ok {
		t.Fatalf("Expecting fb.Error, got: %s", reflect.TypeOf(cursor.Err()))
	}
	if err2.Message != "Cursor is past end of data." {
		t.Errorf("Unexpected error message: %s", err2.Message)
	}
}

func TestCursorExecuteInsertDuplicate(t *testing.T) {
	const sqlInsert = "INSERT INTO TEST (ID) VALUES (0);"

	conn, err := Create(TestConnectionString)
	if err != nil {
		t.Fatalf("Unexpected error creating database: %s", err)
	}
	defer conn.Drop()

	if err = conn.ExecuteScript(sqlSampleSchema); err != nil {
		t.Fatalf("Error executing schema: %s", err)
	}

	if _, err = conn.Execute(sqlInsert); err != nil {
		t.Fatalf("Error executing insert: %s", err)
	}
	_, err = conn.Execute(sqlInsert)
	if err == nil {
		t.Fatal("Expected error executing insert")
	}
	if !strings.Contains(err.Error(), "duplicate column values") {
		t.Errorf("Expected to see 'duplicate column values', got: %s", err)
	}
}

// MBA 8.1s go1.1.2
func BenchmarkNextRow(b *testing.B) {
	const sqlSelect = "SELECT * FROM TEST;"

	b.StopTimer()
	os.Remove(TestFilename)

	conn, err := Create(TestConnectionString)
	if err != nil {
		b.Fatalf("Unexpected error creating database: %s", err)
	}
	defer conn.Drop()

	if err = conn.ExecuteScript(sqlSampleSchema); err != nil {
		b.Fatalf("Error executing schema: %s", err)
	}
	if err = insertGeneratedRows2(conn, 1000); err != nil {
		b.Fatalf("Error executing insert: %s", err)
	}

	b.StartTimer()
	for i := 0; i < b.N; i++ {
		cursor, err := conn.Execute(sqlSelect)
		if err != nil {
			b.Fatalf("Error executing select statement: %s", err)
		}
		for cursor.Next() {
		}
		cursor.Close()
	}
}
