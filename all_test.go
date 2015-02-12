package gott

import (
	"bytes"
	"fmt"
	"reflect"
	"testing"
	"time"
)

func testData() [][]string {
	return [][]string{{"c1", "c2", "c3"},
		{"#adfadfa"},
		{"#adfab ad"},
		{"1", "^^22^^", "23.32"},
		{"2", "foo\nbar", "^2^f`oo\nbar\ntoo^1^^\n\naother line^2^\n"}}
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
func TestFormat(t *testing.T) {
	buf := []string{"foo", "bar", "too"}
	formt := []string{"", "`", "^1^"}
	src := bytes.NewBuffer(nil)
	w := NewWriter(src)
	if err := w.WriteWithFormat(buf, formt); err != nil {
		t.Fatal(err)
	}
	if err := w.Flush(); err != nil {
		t.Fatal(err)
	}
	r := NewReader(src)
	if values, formats, err := r.ReadWithFormat(); err != nil {
		t.Fatal(err)
	} else if !reflect.DeepEqual(values, buf) || !reflect.DeepEqual(formats, formt) {
		t.Fatal("not equ")
	}

}

type TestData struct {
	inlineStr
	T名称  string
	T年龄  int
	T体重  float64
	档案   string //不会被更新
	T档案  []byte
	Time time.Time
	Null string
	Dup  string
}
type inlineStr struct {
	Addr string
	As   string
	Dup  []byte
}

func TestEncoderDecoder(t *testing.T) {
	data := &TestData{
		inlineStr: inlineStr{
			Addr: "addr 1",
			As:   "as",
			Dup:  []byte{1, 2, 3},
		},
		T名称:  "名称",
		T年龄:  12,
		T体重:  12.23,
		档案:   "no write",
		T档案:  []byte{45, 44, 43},
		Time: time.Now().Truncate(time.Second),
		Dup:  "123",
	}
	buf := &bytes.Buffer{}
	enc := NewEncoder(buf)
	if err := enc.Encode(data); err != nil {
		t.Fatal(err)
	}
	dec := NewDecoder(buf)
	outData := new(TestData)
	if err := dec.Decode(outData); err != nil {
		t.Fatal(err)
	}
	if reflect.DeepEqual(outData, data) {
		t.Fatal("equ")
	}
	outData.Dup = "123"
	outData.档案 = "no write"
	if !reflect.DeepEqual(outData, data) {
		fmt.Printf("%#v\n%#v", data, outData)
		t.Fatal("not equ")
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
