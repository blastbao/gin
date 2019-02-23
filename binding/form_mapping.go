// Copyright 2014 Manu Martinez-Almeida.  All rights reserved.
// Use of this source code is governed by a MIT style
// license that can be found in the LICENSE file.

package binding

import (
	"errors"
	"reflect"
	"strconv"
	"strings"
	"time"
)





func mapUri(ptr interface{}, m map[string][]string) error {
	return mapFormByTag(ptr, m, "uri")
}

func mapForm(ptr interface{}, form map[string][]string) error {
	return mapFormByTag(ptr, form, "form")
}




func mapFormByTag(ptr interface{}, form map[string][]string, tag string) error {


	typ := reflect.TypeOf(ptr) .Elem()  //获取变量类型，返回reflect.Type类型 
	val := reflect.ValueOf(ptr).Elem()	//获取变量的值，返回reflect.Value类型 


	//遍历结构体属性
	for i := 0; i < typ.NumField(); i++ {

		typeField   := typ.Field(i)	 	//属性类型
		structField := val.Field(i)		//属性值
		if !structField.CanSet() { 		//是否可以修改其值，一个值必须是可以获得地址且不能通过访问结构的非导出字段获得，方可被修改
			continue
		}

		structFieldKind 	:= structField.Kind() 					//获取属性类别，返回一个常量 
		inputFieldName 		:= typeField.Tag.Get(tag)				//获取属性tag，比如tag="json"或者tag="yaml"


		////** 这块逻辑不用看，一般用不到 **/////
		inputFieldNameList 	:= strings.Split(inputFieldName, ",") 	//
		inputFieldName = inputFieldNameList[0]
		var defaultValue string
		if len(inputFieldNameList) > 1 {
			defaultList := strings.SplitN(inputFieldNameList[1], "=", 2)
			if defaultList[0] == "default" {
				defaultValue = defaultList[1]
			}
		}
		/////////////

		if inputFieldName == "" {
			//如果tag为空，直接用属性名做键
			inputFieldName = typeField.Name

			// if "form" tag is nil, we inspect if the field is a struct or struct pointer.
			// this would not make sense for JSON parsing but it does for a form since data is flatten
			
			//如果属性是结构体指针类型的，那么修正该熟悉类型为其指向的成员的类型。
			if structFieldKind == reflect.Ptr {
				if !structField.Elem().IsValid() {
					structField.Set(reflect.New(structField.Type().Elem()))
				}
				structField 	= structField.Elem()
				structFieldKind = structField.Kind()
			}

			//如果属性是结构体类型，那么递归～～～
			if structFieldKind == reflect.Struct {
				err := mapFormByTag(structField.Addr().Interface(), form, tag)
				if err != nil {
					return err
				}
				continue
			}

			//其他类型，不予处理
		}

		//从form参数表中取inputFieldName对应值
		inputValue, exists := form[inputFieldName]
		//若不存在，使用默认值
		if !exists {
			if defaultValue == "" {
				continue
			}
			inputValue = make([]string, 1)
			inputValue[0] = defaultValue
		}

		numElems := len(inputValue)
		//如果inputValue是个数组，且结构体属性是切片类型，那么:
		if structFieldKind == reflect.Slice && numElems > 0 {
			//获取切片数组的元素类型（sliceOf）
			sliceOf := structField.Type().Elem().Kind()
			//创建指定大小的切片
			slice 	:= reflect.MakeSlice(structField.Type(), numElems, numElems)
			//逐个元素赋值
			for i := 0; i < numElems; i++ {
				//根据元素类型，元素值，切片下标逐个赋值
				if err := setWithProperType(sliceOf, inputValue[i], slice.Index(i)); err != nil {
					return err
				}
			}
			//设置外层结构的属性值，这里的i是外层循环的i
			val.Field(i).Set(slice)
			continue
		}

		//如果结构体属性类型是时间类型，那么需要进行相应格式转换
		if _, isTime := structField.Interface().(time.Time); isTime {
			//根据元素类型，元素值，结构体属性字段进行赋值
			if err := setTimeField(inputValue[0], typeField, structField); err != nil {
				return err
			}
			continue
		}

		//根据元素类型，元素值，结构体属性字段进行赋值
		if err := setWithProperType(typeField.Type.Kind(), inputValue[0], structField); err != nil {
			return err
		}
	}
	return nil
}

func setWithProperType(valueKind reflect.Kind, val string, structField reflect.Value) error {
	switch valueKind {
	case reflect.Int:
		return setIntField(val, 0, structField)
	case reflect.Int8:
		return setIntField(val, 8, structField)
	case reflect.Int16:
		return setIntField(val, 16, structField)
	case reflect.Int32:
		return setIntField(val, 32, structField)
	case reflect.Int64:
		return setIntField(val, 64, structField)
	case reflect.Uint:
		return setUintField(val, 0, structField)
	case reflect.Uint8:
		return setUintField(val, 8, structField)
	case reflect.Uint16:
		return setUintField(val, 16, structField)
	case reflect.Uint32:
		return setUintField(val, 32, structField)
	case reflect.Uint64:
		return setUintField(val, 64, structField)
	case reflect.Bool:
		return setBoolField(val, structField)
	case reflect.Float32:
		return setFloatField(val, 32, structField)
	case reflect.Float64:
		return setFloatField(val, 64, structField)
	case reflect.String:
		structField.SetString(val)
	case reflect.Ptr:
		if !structField.Elem().IsValid() {
			structField.Set(reflect.New(structField.Type().Elem()))
		}
		structFieldElem := structField.Elem()
		return setWithProperType(structFieldElem.Kind(), val, structFieldElem)
	default:
		return errors.New("Unknown type")
	}
	return nil
}

func setIntField(val string, bitSize int, field reflect.Value) error {
	if val == "" {
		val = "0"
	}
	intVal, err := strconv.ParseInt(val, 10, bitSize)
	if err == nil {
		field.SetInt(intVal)
	}
	return err
}

func setUintField(val string, bitSize int, field reflect.Value) error {
	if val == "" {
		val = "0"
	}
	uintVal, err := strconv.ParseUint(val, 10, bitSize)
	if err == nil {
		field.SetUint(uintVal)
	}
	return err
}

func setBoolField(val string, field reflect.Value) error {
	if val == "" {
		val = "false"
	}
	boolVal, err := strconv.ParseBool(val)
	if err == nil {
		field.SetBool(boolVal)
	}
	return err
}

func setFloatField(val string, bitSize int, field reflect.Value) error {
	if val == "" {
		val = "0.0"
	}
	floatVal, err := strconv.ParseFloat(val, bitSize)
	if err == nil {
		field.SetFloat(floatVal)
	}
	return err
}

func setTimeField(val string, structField reflect.StructField, value reflect.Value) error {
	timeFormat := structField.Tag.Get("time_format")
	if timeFormat == "" {
		timeFormat = time.RFC3339
	}

	if val == "" {
		value.Set(reflect.ValueOf(time.Time{}))
		return nil
	}

	l := time.Local
	if isUTC, _ := strconv.ParseBool(structField.Tag.Get("time_utc")); isUTC {
		l = time.UTC
	}

	if locTag := structField.Tag.Get("time_location"); locTag != "" {
		loc, err := time.LoadLocation(locTag)
		if err != nil {
			return err
		}
		l = loc
	}

	t, err := time.ParseInLocation(timeFormat, val, l)
	if err != nil {
		return err
	}

	value.Set(reflect.ValueOf(t))
	return nil
}
