package gott

import (
	"encoding/base64"
	"fmt"
	"io"
	"reflect"
	"strconv"
	"time"
)

type ttType struct {
	PkgPath string
	Name    string
}

// 解析类型化的TT文件，只能序列化结构类型，支持嵌入的结构(非指针嵌入)文,
// 写入时，先注册类型（如果没有注册过），然后写入属性，注册类型用特殊的符号*,
// 如果已经注册过，则是引用类型，用@符号，注册或引用后，后续的数据行就是该类型的数据。
// 具体的文件格式如下：
//	`*`	gott	TFoo	One	Two		--注册类型gott.TFoo，其有两个属性One Two，后面的数据行全部采用TFoo的格式
//	fooV11	fooV12					--foo的实例
//	fooV21	fooV22					--foo的实例
//	`*`	other	TBaro	One	Two	Three	--注册类型other.TBar，其有三个属性One Two Three，后面的数据行全部采用TBar的格式
//	barV11	barV12	barV13				--bar的实例
//	barV11	barV12	barV13				--bar的实例
//	`@`	gott	TFoo				--引用类型TBar，必须在前面注册过，后面的数据行全部采用TFoo的格式
//	fooV31	fooV32					--foo的实例
type Decoder struct {
	reader      *Reader
	types       map[*ttType][]string
	currentType *ttType
}

var (
	ErrTypeLine     = fmt.Errorf("type line error")
	ErrNotFoundProp = fmt.Errorf("not found the prop")
)

func NewDecoder(r io.Reader) *Decoder {
	return &Decoder{NewReader(r), map[*ttType][]string{}, nil}

}

func findPropByName(obj reflect.Type, name string) ([]int, error) {
	for i := 0; i < obj.NumField(); i++ {
		f := obj.Field(i)
		if f.Anonymous && f.Type.Kind() == reflect.Struct {
			result, err := findPropByName(f.Type, name)
			if err != nil {
				return nil, err
			}
			if result != nil {
				return append([]int{i}, result...), err
			}
		} else {
			if f.Name == name {
				return []int{i}, nil
			}
		}
	}
	return nil, nil
}
func decode(encValue string, value reflect.Value) error {
	switch value.Interface().(type) {
	case string:
		value.Set(reflect.ValueOf(encValue))
	case float32:
		if encValue == "" {
			value.Set(reflect.ValueOf(float32(0)))
		} else {

			if f, err := strconv.ParseFloat(encValue, 32); err != nil {
				return err
			} else {
				value.Set(reflect.ValueOf(f))
			}
		}
	case float64:
		if encValue == "" {
			value.Set(reflect.ValueOf(float64(0)))
		} else {

			if f, err := strconv.ParseFloat(encValue, 64); err != nil {
				return err
			} else {
				value.Set(reflect.ValueOf(f))
			}
		}
	case int:
		if encValue == "" {
			value.Set(reflect.ValueOf(int(0)))
		} else {
			if f, err := strconv.Atoi(encValue); err != nil {
				return err
			} else {
				value.Set(reflect.ValueOf(f))
			}
		}

	case int64:
		if encValue == "" {
			value.Set(reflect.ValueOf(int64(0)))
		} else {
			if f, err := strconv.ParseInt(encValue, 10, 64); err != nil {
				return err
			} else {
				value.Set(reflect.ValueOf(f))
			}
		}
	case time.Time:
		if encValue == "" {
			value.Set(reflect.ValueOf(time.Time{}))
		} else {
			if f, err := time.Parse(time.RFC3339, encValue); err != nil {
				return err
			} else {
				value.Set(reflect.ValueOf(f))
			}
		}
	case []byte:
		if encValue == "" {
			value.Set(reflect.ValueOf(nil))
		} else {
			bys, err := base64.StdEncoding.DecodeString(encValue)
			if err != nil {
				return err
			}
			value.Set(reflect.ValueOf(bys))
		}
	default:
		return fmt.Errorf("invalid type :%T", value.Interface())
	}
	return nil
}
func (t *Decoder) Decode(v interface{}) error {
	vtype := reflect.TypeOf(v)
	value := reflect.ValueOf(v)
	if vtype.Kind() != reflect.Ptr {
		return fmt.Errorf("param v must is ptr to struct")
	}
	vtype = vtype.Elem()
	value = value.Elem()
	if vtype.Kind() != reflect.Struct {
		return fmt.Errorf("param v must is ptr to struct")
	}

	var values, formats []string
	for {
		var err error
		values, formats, err = t.reader.ReadWithFormat()
		if err != nil {
			return err
		}
		//空行忽略，继续
		if values != nil {
			if values[0] == "*" && formats[0] == "`" {
				//注册类型
				t.currentType = &ttType{values[1], values[2]}
				t.types[t.currentType] = values[3:]
			} else if values[0] == "@" && formats[0] == "`" {
				//引用类型
				var foundType *ttType = nil
				for tK, _ := range t.types {
					if tK.PkgPath == values[1] && tK.Name == values[2] {
						foundType = tK
						break
					}
				}
				if foundType != nil {
					t.currentType = foundType
				} else {
					return fmt.Errorf("the type %s.%s not found", values[1], values[2])
				}
			} else {
				break //读取到数据
			}
		}
	}

	if t.currentType == nil {
		return fmt.Errorf("current type is empty")
	}

	typeColumns := t.types[t.currentType]
	if len(values) != len(typeColumns) {
		return fmt.Errorf("the value %#v length not equ type prop name :%#v", values, typeColumns)
	}
	for i, fieldStringValue := range values {
		idx, err := findPropByName(vtype, typeColumns[i])
		if err != nil {
			return err
		}
		if idx == nil {
			return fmt.Errorf("can't find the prop:%s at type %T", typeColumns[i], v)
		}
		if err := decode(fieldStringValue, value.FieldByIndex(idx)); err != nil {
			return err
		}
	}
	return nil
}
