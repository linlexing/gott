package gott

import (
	"bytes"
	"fmt"
	"reflect"
	"testing"
	"time"
)

func Test1(t *testing.T) {
	bys := &bytes.Buffer{}
	cols := []string{"c1", "c2", "c3", "c4", "c5"}
	tys := []string{INT, STR, FLOAT, TIME, BLOB}
	enc := NewEncoder(bys, cols, tys)
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
