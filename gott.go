// Package csv reads and writes tab-separated values (TT) files.
//
// A tt file contains zero or more records of one or more fields per record.
// Each record is separated by the newline character. The final record may
// optionally be followed by a newline character.
//
//  field1,field2,field3
// White space is considered part of a field.
//
// Carriage returns before newline characters are silently removed.
//
// Blank lines are ignored.  A line with only whitespace characters (excluding
// the ending newline character) is not considered a blank line.
//
// Fields which start and stop with the quote character ` are called
// quoted-fields.
// The beginning and ending quote are not part of the field.
//
// The source:
//
//  normal string,`quoted-field`
// results in the fields
//
//  {"normal string", "quoted-field"}
// Within a character ^ followed by a second character ^ is considered a single
// quote.
//
//  ^^the "word" is true^^,^foo^a "quoted-field"^foo^
// results in
//
//  {`the "word" is true`, `a "quoted-field"`}
// Newlines and commas may be included in a quoted-field
//
//  ^^`Multi-line
//  field`^^,`comma
// is ,`
// results in
//
//  {"`Multi-line\nfield`", "comma\nis ,"}
package gott

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"
)

type ParseError struct {
	Line   int   // Line where the error occurred
	Column int   // Column (rune index) where the error occurred
	Err    error // The actual error
}

var (
	ErrQuote      = errors.New("extraneous ` in field")
	ErrUpQuote    = errors.New("extraneous ^ in field")
	ErrNotUpQuote = errors.New("extraneous ^ not end field")
	ErrFieldCount = errors.New("wrong number of fields in line")
)

func (e *ParseError) Error() string {
	return fmt.Sprintf("line:%d,column:%d parse error:%s", e.Line, e.Column, e.Err)
}

// A Reader reads records from a TT-encoded file.
//
// As returned by NewReader. The exported fields can be changed to customize
// the details before the first call to Read or ReadAll.
//
// Comma is the field delimiter. It defaults to '\t'.
//
// Comment, if not 0, is the comment character. Lines beginning with the Comment
// character are ignored.
//
// If FieldsPerRecord is positive, Read requires each record to have the given
// number of fields. If FieldsPerRecord is 0, Read sets it to the number of
// fields in the first record, so that future records must have the same field
// count. If FieldsPerRecord is negative, no check is made and records may have
// a variable number of fields.
type Reader struct {
	Comma           rune // field delimiter (set to '\t' by NewReader)
	Comment         rune // comment character for start of line
	FieldsPerRecord int  // number of expected fields per record
	r               *bufio.Reader
}

//NewReader returns a new Reader that reads from r.
func NewReader(r io.Reader) *Reader {
	return &Reader{
		Comma: '\t',
		r:     bufio.NewReader(r),
	}
}

//Read reads one record from r. The record is a slice of strings with each
// string representing one field.
func (r *Reader) Read() (record []string, err error) {
	for {
		record, err = r.parseRecord()
		if record != nil {
			break
		}
		if err != nil {
			return nil, err
		}
	}
	if r.FieldsPerRecord > 0 {
		if len(record) != r.FieldsPerRecord {
			return record, fmt.Errorf("error field count:%d,must is :%d,data:%#v", len(record), r.FieldsPerRecord, record)
		}
	} else if r.FieldsPerRecord == 0 {
		r.FieldsPerRecord = len(record)
	}
	return record, nil
}

// ReadAll reads all the remaining records from r.
// Each record is a slice of fields.
// A successful call returns err == nil, not err == EOF. Because ReadAll is
// defined to read until EOF, it does not treat end of file as an error to be
// reported.
func (r *Reader) ReadAll() (records [][]string, err error) {
	for {
		record, err := r.Read()
		if err == io.EOF {
			return records, nil
		}
		if err != nil {
			return nil, err
		}
		records = append(records, record)
	}
}

func (dec *Reader) parseRecord() ([]string, error) {
	result := []string{}
	oneField := &bytes.Buffer{}
	for {
		if r, _, err := dec.r.ReadRune(); err != nil {
			if err == io.EOF {
				//如果首字符是EOF，则返回nil

				if len(result) == 0 && oneField.Len() == 0 {
					result = nil
				} else {
					result = append(result, oneField.String())
				}
			}
			return result, err
		} else {
			switch r {
			case dec.Comment:
				//如果首字符是注释符号，则返回nil
				if len(result) == 0 && oneField.Len() == 0 {
					_, err := dec.r.ReadString('\n')
					return nil, err
				}
			case dec.Comma:
				result = append(result, oneField.String())
				oneField.Reset()
			case '\r':
				//\r被丢弃
				if nextC, err := dec.r.Peek(1); err != nil {
					return nil, err
				} else if nextC[0] != '\n' {
					if _, err := oneField.WriteRune(r); err != nil {
						return nil, err
					}
				}
			case '\n':
				//如果首字符是\n，则返回nil
				if len(result) == 0 && oneField.Len() == 0 {
					result = nil
				} else {

					result = append(result, oneField.String())
				}
				return result, nil
			case '`':
				if oneField.Len() != 0 {
					return result, fmt.Errorf("the line :%v,%s format error,\"`\"at the data", result, oneField)
				}
				if str, err := dec.r.ReadString('`'); err != nil {
					return result, fmt.Errorf("the line :%v,%s format error,\"`\" not end", result, str)
				} else {
					//取出最后的`符号
					result = append(result, str[:len(str)-1])
				}
				if r1, _, err := dec.r.ReadRune(); err != nil || r1 != dec.Comma {
					return result, err
				}
			case '^':
				if oneField.Len() != 0 {
					return result, fmt.Errorf("the line :%v,%s format error,\"^\"at the data", result, oneField)
				}
				if str, err := dec.r.ReadString('^'); err != nil {
					return result, fmt.Errorf("the line :%v,%s format error,\"^\" not end", result, str)
				} else {
					id := "^" + str
					if oneField, err := read(dec.r, []byte(id)); err != nil {
						return result, fmt.Errorf("the line :%v,%s format error,%s not end", result, oneField, id)
					} else {
						result = append(result, string(oneField))
					}
				}
				if r1, _, err := dec.r.ReadRune(); err != nil || r1 != dec.Comma {
					return result, err
				}
			default:
				if _, err := oneField.WriteRune(r); err != nil {
					return nil, err
				}
			}
		}
	}
}

type reader interface {
	ReadString(delim byte) (line string, err error)
}

func read(r reader, delim []byte) (line []byte, err error) {
	for {
		s := ""
		s, err = r.ReadString(delim[len(delim)-1])
		if err != nil {
			return
		}
		line = append(line, []byte(s)...)
		if bytes.HasSuffix(line, delim) {
			return line[:len(line)-len(delim)], nil
		}
	}
}

// A Writer writes records to a TT encoded file.
//
// As returned by NewWriter, a Writer writes records terminated by a
// newline and uses '\t' as the field delimiter.  The exported fields can be
// changed to customize the details before the first call to Write or WriteAll.
//
// Comma is the field delimiter.
type Writer struct {
	Comma rune
	w     *bufio.Writer
}

// NewWriter returns a new Writer that writes to w.
func NewWriter(w io.Writer) *Writer {
	return &Writer{
		w:     bufio.NewWriter(w),
		Comma: '\t',
	}
}
func (w *Writer) encodeString(str string) string {
	notSign := true
	notSpec := true
	dolaId := []string{}
	oneId := bytes.Buffer{}
	idLen := -1
	if str == "" {
		return str
	}
	//如果是` ^开头，则取消非特殊化标识，因为解析时会错误识别为多行字符串
	if str[0] == '^' {
		notSpec = false
	}
	for _, c := range str {
		if idLen > -1 {
			idLen++
			//只捕获前9位，如果没有^出现，则复位
			if idLen > 9 {
				idLen = -1
			}
		}
		switch c {
		case '`':
			notSign = false
		case '\r', '\n', w.Comma:
			notSpec = false
		case '^':
			//如果捕获ID已经开始，则增加ID
			if idLen > -1 {
				dolaId = append(dolaId, oneId.String())
			}
			//开始捕获ID
			oneId.Truncate(0)
			idLen = 0

		default:
			if idLen > -1 {
				oneId.WriteRune(c)
			}
		}
	}
	//如果没有含有歧义的字符，则是原样输出
	if notSpec {
		return str
	}
	//如果没有出现上单引号，则返回以其包括的字符串
	if notSign {
		return "`" + str + "`"
	}
	//确定一个分割字符串
	dem := ""
	has := false
	for _, oneId := range dolaId {
		if oneId == dem {
			has = true
			break
		}
	}
	if has {
		for i := 1; i < 1000; i++ {
			dem = strconv.Itoa(i)
			has = false
			for _, oneId := range dolaId {
				if oneId == dem {
					has = true
					break
				}
			}
			if !has {
				break
			}
		}
	}
	if has {
		panic("can't gen the ^??^ id")
	} else {
		return "^" + dem + "^" + str + "^" + dem + "^"
	}
}

// Writer writes a single TT record to w along with any necessary quoting.
// A record is a slice of strings with each string being one field.
func (w *Writer) Write(record []string) (err error) {

	if w.w == nil {
		return fmt.Errorf("must call NewWriter init the class")
	}
	if record == nil || len(record) == 0 {
		return fmt.Errorf("the record is nil")
	}
	wdata := make([]string, len(record))
	for i, v := range record {
		wdata[i] = w.encodeString(v)
	}

	if _, err := w.w.Write([]byte(strings.Join(wdata, string(w.Comma)) + "\n")); err != nil {
		return err
	}
	return nil
}

// Flush writes any buffered data to the underlying io.Writer.
// To check if an error occurred during the Flush, call Error.
func (w *Writer) Flush() {
	w.w.Flush()
}

// Error reports any error that has occurred during a previous Write or Flush.
func (w *Writer) Error() error {
	_, err := w.w.Write(nil)
	return err
}

// WriteAll writes multiple TT records to w using Write and then calls Flush.
func (w *Writer) WriteAll(records [][]string) (err error) {
	for _, record := range records {
		err = w.Write(record)
		if err != nil {
			return err
		}
	}
	return w.w.Flush()
}
