package gott

import (
	"encoding/base64"
	"fmt"
	"io"
	"reflect"
	"time"
)

type Encoder struct {
	writer      *Writer
	types       map[*ttType][]string
	currentType *ttType
}

func NewEncoder(w io.Writer) *Encoder {
	return &Encoder{NewWriter(w), map[*ttType][]string{}, nil}
}
func encode(value interface{}) (result string, err error) {
	switch tv := value.(type) {
	case nil:
		result = ""
	case string:
		result = tv
	case float32, float64, int, int64:
		result = fmt.Sprintf("%v", tv)
	case time.Time:
		result = tv.Format(time.RFC3339)
	case []byte:
		result = base64.StdEncoding.EncodeToString(tv)
	default:
		err = fmt.Errorf("invalid type :%T", value)
	}
	return
}
func (enc *Encoder) findType(ty reflect.Type) *ttType {
	pkgPath := ty.PkgPath()
	tyName := ty.Name()
	for sty, _ := range enc.types {
		if sty.PkgPath == pkgPath && sty.Name == tyName {
			return sty
		}
	}
	return nil
}

//获取指定结构的属性名称和属性值,重名的自动去除
func getStructValues(obj reflect.Value) (fieldNames []string, fieldValues []interface{}, err error) {
	appendNoDup := func(name string, value interface{}) {
		bFound := false
		for _, vv := range fieldNames {
			if vv == name {
				bFound = true
				break
			}
		}
		if !bFound {
			fieldNames = append(fieldNames, name)
			fieldValues = append(fieldValues, value)
		}

		return
	}
	fieldNames = []string{}
	fieldValues = []interface{}{}
	ty := obj.Type()
	for i := 0; i < obj.NumField(); i++ {
		f := obj.Field(i)
		t := ty.Field(i)
		if !f.CanInterface() { //unexported
			continue
		}
		if t.Anonymous && f.Kind() == reflect.Struct {
			ns, vs, e := getStructValues(f)
			if err != nil {
				fieldNames, fieldValues, err = ns, vs, e
				return
			}
			if len(ns) > 0 {
				//排重后添加
				for si, sv := range ns {
					appendNoDup(sv, vs[si])
				}
			}
		} else {
			appendNoDup(t.Name, f.Interface())
		}
	}
	return
}

func (enc *Encoder) Encode(v interface{}) error {
	value := reflect.ValueOf(v)
	if value.Kind() == reflect.Ptr {
		value = value.Elem()
	}
	vtype := value.Type()
	if vtype.Kind() != reflect.Struct {
		return fmt.Errorf("param v must is ptr to struct")
	}
	columns, values, err := getStructValues(value)
	if err != nil {
		return err
	}

	encType := enc.findType(vtype)
	if encType == nil {
		//注册新类型
		encType = &ttType{
			PkgPath: vtype.PkgPath(),
			Name:    vtype.Name(),
		}
		line := append([]string{"*", encType.PkgPath, encType.Name}, columns...)
		fmts := make([]string, len(line))
		for i, _ := range line {
			fmts[i] = ""
		}
		fmts[0] = "`"
		if err := enc.writer.WriteWithFormat(line, fmts); err != nil {
			return err
		}
		enc.currentType = encType
		enc.types[enc.currentType] = columns
	} else if encType != enc.currentType {
		//引用类型
		line := []string{"@", encType.PkgPath, encType.Name}
		fmts := make([]string, len(line))
		for i, _ := range line {
			fmts[i] = ""
		}
		fmts[0] = "`"
		if err := enc.writer.WriteWithFormat(line, fmts); err != nil {
			return err
		}
		enc.currentType = encType
	}
	//写入数据
	line := []string{}
	for _, sv := range values {
		if str, err := encode(sv); err != nil {
			return err
		} else {
			line = append(line, str)
		}
	}
	if err := enc.writer.Write(line); err != nil {
		return err
	}
	if err := enc.writer.Flush(); err != nil {
		return err
	}
	return nil
}
