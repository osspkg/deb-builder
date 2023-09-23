/*
 *  Copyright (c) 2021-2023 Mikhail Knyazhev <markus621@gmail.com>. All rights reserved.
 *  Use of this source code is governed by a BSD-3-Clause license that can be found in the LICENSE file.
 */

package packages

import (
	"bytes"
	"fmt"
	"reflect"
	"strconv"
)

var (
	separator = byte(':')
	lineend   = byte('\n')
	linebreak = byte(' ')
	rawKey    = "_"
)

func decode(b []byte, in interface{}) error {
	keys := make(map[string]string)
	elements := reflect.ValueOf(in).Elem()

	for i := 0; i < elements.NumField(); i++ {
		tagField := elements.Type().Field(i).Tag.Get("key")
		keys[tagField] = ""
	}
	keys[rawKey] = ""

	key := make([]byte, 0, 256)
	value := make([]byte, 0, 256)
	index := 0
	state := 0

	writeData := func() {
		kk := string(key)
		vv := string(value)

		if len(key) == 0 && len(value) == 0 {
			return
		}

		if _, ok := keys[kk]; ok {
			keys[kk] = vv
		} else {
			if len(keys[rawKey]) > 0 {
				keys[rawKey] += string(lineend)
			}
			keys[rawKey] += fmt.Sprintf("%s: %v", kk, vv)
		}

		key, value = key[:0], value[:0]
		state = 0
	}

	for i := 0; i < len(b); i++ {
		next := b[i]
		if index == 0 {
			if next == linebreak {
				if len(key) == 0 {
					return fmt.Errorf("invalid file format")
				}
				value = append(value, lineend)
				state = 1
			} else {
				writeData()
			}
		}

		// key
		if state == 0 {
			if next == linebreak && index == 0 {
				return fmt.Errorf("invalid file format")
			}
			if next == lineend && index != 0 {
				return fmt.Errorf("invalid file format")
			}
			if next == separator && index != 0 {
				state = 1
				i++
				continue
			} else {
				key = append(key, next)
			}
		}

		// only value
		if state == 1 {
			if next == lineend {
				state = 0
				index = 0
				continue
			} else {
				value = append(value, next)
			}
		}

		index++
	}

	writeData()

	for i := 0; i < elements.NumField(); i++ {
		field := elements.Field(i)
		typeField := elements.Type().Field(i)
		tagField := typeField.Tag.Get("key")

		if vv, ok := keys[tagField]; ok && len(vv) > 0 {
			switch kind := field.Kind(); kind {
			case reflect.String:
				elements.Field(i).SetString(vv)
			case reflect.Int, reflect.Int64:
				num, err := strconv.ParseInt(vv, 10, 64)
				if err != nil {
					return err
				}
				elements.Field(i).SetInt(num)
			default:
				return fmt.Errorf("type is not supported: %s %v", tagField, kind)
			}
		}
	}

	return nil
}

func encode(in interface{}) ([]byte, error) {
	buf := &bytes.Buffer{}
	elements := reflect.ValueOf(in).Elem()

	for i := 0; i < elements.NumField(); i++ {
		result := ""
		field := elements.Field(i)
		tagField := elements.Type().Field(i).Tag.Get("key")

		if field.IsZero() {
			continue
		}

		if tagField != "_" {
			result = fmt.Sprintf("%s: %v\n", tagField, field.Interface())
		} else {
			result = field.Interface().(string)
		}

		if _, err := buf.WriteString(result); err != nil {
			return nil, err
		}
	}

	return buf.Bytes(), nil
}
