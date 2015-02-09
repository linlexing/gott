package gott

import (
	"bytes"
	"reflect"
	"testing"
)

func testData() [][]string {
	return [][]string{{"c1", "c2", "c3"},
		{"#adfadfa"},
		{"#adfab ad"},
		{"1", "^^22^^", "23.32"},
		{"2", `foo\nbar`, "^2^foo\nbar\ntoo^1^^\n\naother line^2^\n"}}
}
func TestRW(t *testing.T) {
	buf := testData()
	trueBuf := [][]string{buf[0], buf[3], buf[4]}
	src := bytes.NewBuffer(nil)
	w := NewWriter(src)
	if err := w.WriteAll(buf); err != nil {
		t.Fatal(err)
	}
	r := NewReader(src)
	r.Comment = '#'
	if lines, err := r.ReadAll(); err != nil {
		t.Fatal(err)
	} else if !reflect.DeepEqual(trueBuf, lines) {
		t.Fatalf("not equ,\n%#v\n%#v", buf, lines)
	}
}

func BenchmarkWriter(b *testing.B) {
	src := bytes.NewBuffer(nil)
	w := NewWriter(src)
	for i := 0; i < b.N; i++ {
		w.Write(testData()[4])
	}
}
func BenchmarkReader(b *testing.B) {
	src := bytes.NewBuffer(nil)
	w := NewWriter(src)
	r := NewReader(src)
	for i := 0; i < b.N; i++ {
		w.Write(testData()[4])
	}
	for i := 0; i < b.N; i++ {
		r.Read()
	}
}

/*
func Test1(t *testing.T) {
	bys := &bytes.Buffer{}
	cols := []string{"c1", "c2", "c3", "c4", "c5"}
	tys := []string{INT, STR, FLOAT, TIME, BLOB}
	enc := NewReader(bys)
	row1 := []interface{}{nil, "1", 12.90, time.Now().Truncate(time.Second), []byte{1, 2, 3}}
	row2 := []interface{}{2, "foo\nbar", nil, nil, nil}
	row3 := []interface{}{3, "^1^^`foo\nbar`", nil, nil, nil}
	if err := enc.Encode(row1); err != nil {
		t.Fatal(err)
	}
	if err := enc.Encode(row2); err != nil {
		t.Fatal(err)
	}
	if err := enc.Encode(row3); err != nil {
		t.Fatal(err)
	}
	dec := NewDecoder(bys)
	if cs, err := dec.Columns(); err != nil {
		t.Fatal(err)
	} else if !reflect.DeepEqual(cols, cs) {
		t.Fatalf("columns not equ,%v", cs)
	}
	if ts, err := dec.Types(); err != nil {
		t.Fatal(err)
	} else if !reflect.DeepEqual(tys, ts) {
		t.Fatalf("columns not equ,%v", ts)
	}
	if ts, err := dec.Decode(); err != nil {
		t.Fatal(err)
	} else if !reflect.DeepEqual(row1, ts) {
		t.Fatalf("columns not equ,%v != %v", ts, row1)
	}
	if ts, err := dec.Decode(); err != nil {
		t.Fatal(err)
	} else if !reflect.DeepEqual(row2, ts) {
		t.Fatalf("columns not equ,%v != %v", ts, row2)
	}
	if ts, err := dec.Decode(); err != nil {
		t.Fatal(err)
	} else if !reflect.DeepEqual(row3, ts) {
		t.Fatalf("columns not equ,%v != %v", ts, row3)
	}
	fmt.Print(bys.String())
}
*/
