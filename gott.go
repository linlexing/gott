package gott

import (
	"bufio"
	"bytes"
	"encoding/base64"
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"
)

const (
	INT   = "INT"
	STR   = "STR"
	FLOAT = "FLOAT"
	TIME  = "TIME"
	BLOB  = "BLOB"
)

type Decoder struct {
	reader  *bufio.Reader
	columns []string
	types   []string
}

func NewDecoder(r io.Reader) *Decoder {

	return &Decoder{reader: bufio.NewReader(r)}
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
func (dec *Decoder) readLine() ([]string, error) {
	result := []string{}
	oneField := &bytes.Buffer{}
	for {
		if r, _, err := dec.reader.ReadRune(); err != nil {
			if err == io.EOF {
				result = append(result, oneField.String())
			}
			return result, err
		} else {
			switch r {
			case '\t':
				result = append(result, oneField.String())
				oneField.Reset()
			case '\r':
			//\r被丢弃
			case '\n':
				result = append(result, oneField.String())
				return result, nil
			case '`':
				if oneField.Len() != 0 {
					return result, fmt.Errorf("the line :%v,%s format error,\"`\"at the data", result, oneField)
				}
				if str, err := dec.reader.ReadString('`'); err != nil {
					return result, fmt.Errorf("the line :%v,%s format error,\"`\" not end", result, str)
				} else {
					//取出最后的`符号
					result = append(result, str[:len(str)-1])
				}
			case '^':
				if oneField.Len() != 0 {
					return result, fmt.Errorf("the line :%v,%s format error,\"^\"at the data", result, oneField)
				}
				if str, err := dec.reader.ReadString('^'); err != nil {
					return result, fmt.Errorf("the line :%v,%s format error,\"^\" not end", result, str)
				} else {
					id := "^" + str
					if oneField, err := read(dec.reader, []byte(id)); err != nil {
						return result, fmt.Errorf("the line :%v,%s format error,%s not end", result, oneField, id)
					} else {
						//取出最后的id
						result = append(result, str[:len(str)-len(id)])
					}
				}
			default:
				if _, err := oneField.WriteRune(r); err != nil {
					return nil, err
				}
			}
		}
	}
}
func (dec *Decoder) readHeader() (err error) {
	if dec.columns == nil {
		dec.columns, err = dec.readLine()
	}
	if err != nil {
		return
	}
	if dec.types == nil {
		dec.types, err = dec.readLine()
	}
	return
}
func (dec *Decoder) typeStr(record []string) (result []interface{}, err error) {
	if record == nil || len(record) == 0 {
		panic("the record is nil")
	}
	if len(record) != len(dec.columns) {
		return nil, fmt.Errorf("record:%v length is %d,column number is %d", record, len(record), len(dec.columns))
	}
	result = []interface{}{}
	for i, v := range record {
		if v == "" {
			result = append(result, nil)
		} else {
			var tmp interface{}
			switch dec.types[i] {
			case INT:
				tmp, err = strconv.ParseInt(v, 10, 64)
			case STR:
				tmp = v
			case FLOAT:
				tmp, err = strconv.ParseFloat(v, 64)
			case TIME:
				tmp, err = time.Parse(time.RFC3339, v)
			case BLOB:
				tmp, err = base64.StdEncoding.DecodeString(v)
			default:
				panic(fmt.Errorf("the type %s invalid", dec.types[i]))
			}
			if err != nil {
				return
			}
			result = append(result, tmp)
		}
	}
	return
}

//从输入流中取出一条记录的值，e必须为指向[]interface{}的指针，
//自动跳过第一、第二行，空行
func (dec *Decoder) Decode() ([]interface{}, error) {
	if dec.columns == nil {
		if err := dec.readHeader(); err != nil {
			return nil, err
		}
	}
	if line, err := dec.readLine(); err != nil {
		if err == io.EOF {
			tarray, _ := dec.typeStr(line)
			return tarray, err
		} else {
			return nil, err
		}
	} else {
		return dec.typeStr(line)
	}
}

//读取并返回输入流中第一行的内容，各列的名称
func (dec *Decoder) Columns() (result []string, err error) {
	if dec.columns == nil {
		err = dec.readHeader()
	}
	result = dec.columns
	return
}

//读取并返回输入流中第二行的内容，各列的数据类型
func (dec *Decoder) Types() (result []string, err error) {
	if dec.columns == nil {
		err = dec.readHeader()
	}
	result = dec.types
	return
}

type Encoder struct {
	writer  io.Writer
	columns []string
	types   []string
}

//用指定的字段名、数据类型初始化一个流，调用完成后，第一行、第二行的内容已经写入
func NewEncoder(w io.Writer, columns []string, types []string) *Encoder {
	if len(columns) == 0 {
		panic("must have column")
	}
	if len(columns) != len(types) {
		panic("the column number and type number must equ")
	}
	for _, s := range columns {
		if strings.ContainsAny(s, "\r\n\t") {
			panic("the column name can't contains \\r \\n \\t")
		}
	}
	for _, i := range types {
		switch i {
		case "INT", "STR", "TIME", "BLOB", "FLOAT":
		default:
			panic(fmt.Errorf("the type %d invalid", i))
		}
	}

	if _, err := w.Write([]byte(strings.Join(columns, "\t") + "\n")); err != nil {
		panic(err)
	}
	if _, err := w.Write([]byte(strings.Join(types, "\t") + "\n")); err != nil {
		panic(err)
	}
	return &Encoder{w, columns, types}
}
func encodeString(str string) string {
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
		case '\r', '\n', '\t':
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

//写入一条记录的值，e为[]interface{}的slice，如果数据类型和个数与
//预设的不一致，将返回错误
func (enc *Encoder) Encode(data []interface{}) error {
	if enc.writer == nil {
		return fmt.Errorf("must call NewEncoder init the class")
	}
	if len(data) != len(enc.columns) {
		return fmt.Errorf("the data length %d not equ column number:%d", len(data), len(enc.columns))
	}
	wdata := []string{}
	for i, v := range data {
		t := ""
		fieldValue := ""
		switch v := v.(type) {
		case nil:
		case string:
			t = STR
			fieldValue = encodeString(v)
		case []byte:
			t = BLOB
			fieldValue = base64.StdEncoding.EncodeToString(v)
		case int, int64, int16, byte:
			t = INT
			fieldValue = fmt.Sprintf("%d", v)
		case float32, float64:
			t = FLOAT
			fieldValue = fmt.Sprintf("%f", v)
		case time.Time:
			fieldValue = v.Format(time.RFC3339)
		default:
			return fmt.Errorf("the type %T invalid", v)
		}
		if t != "" && t != enc.types[i] {
			return fmt.Errorf("the column:%s type is %s,the data type is :%T", enc.columns[i], enc.types[i], v)
		}
		wdata = append(wdata, fieldValue)

	}

	if _, err := enc.writer.Write([]byte(strings.Join(wdata, "\t") + "\n")); err != nil {
		return err
	}
	return nil
}
