package main

import (
	"fmt"
	"io/ioutil"
)

var bytes []byte
var byteIndex int = 0


type LineAttribute struct {
	a int // u2
	b int // u4
	c int // u2
}

type ExceptionTable struct {
	start_pc int // u2
	end_pc int // u2
	handler_pc int //u2
	catch_type int // u2
}

type CodeAttribute struct {
	attribute_name_index int // u2
	attribute_length int // u4
	max_stqck int // u2
	max_locals int // u2
	code_length int // u4
	code []byte
	exception_table_length int // u2
	exception_tables []ExceptionTable
	attributes_count int // u2
	attribute_infos []AttributeInfo
	//body []byte
}

type AttributeInfo struct {
	attribute_name_index int // u2
	attribute_length int // u4
	body []byte
}

type MethodInfo struct {
	access_flags     int
	name_index       int
	descriptor_index int
	attributes_count int
	ai    []CodeAttribute
}

type ConstantNameAndType struct {
	tag byte
	first int
	second int
}

type ConstantUTF8 struct {
	tag byte
	len int
	content string
}

type ConstantClass struct {
	tag byte
	s int
}


type ConstantString struct {
	tag byte
	s int
}

type ConstantMethodfRef struct {
	tag byte
	first int
	second int
}

func readCafebabe() []byte {
	byteIndex += 4
	return bytes[0:4]
}

func readU2() int {
	left := bytes[byteIndex]
	right := bytes[byteIndex+1]
	byteIndex += 2

	return int(int(left) * 256 + int(right))
}

func readU4() int {
	b1 := bytes[byteIndex]
	b2 := bytes[byteIndex+1]
	b3 := bytes[byteIndex+2]
	b4 := bytes[byteIndex+3]
	byteIndex += 4

	return int(int(b1) * 256 * 256 * 256 + int(b2) * 256 * 256 + int(b3) * 256 + int(b4))
}

func readBytes(n int) []byte {
	r := bytes[byteIndex:byteIndex+n]
	byteIndex += n
	return r
}
func readByte() byte {
	b := bytes[byteIndex]
	byteIndex++
	return b
}

func readAttribute() LineAttribute {
	at := LineAttribute{
		a: readU2(),
		b: readU4(),
		c: readU2(),
	}
	return at
}

func readAttributeInfo() AttributeInfo {
	a := AttributeInfo{
		attribute_name_index: readU2(),
		attribute_length: readU4(),
	}
	a.body = readBytes(a.attribute_length)
	return a
}

func readExceptionTable() {
	readU2()
	readU2()
	readU2()
	readU2()
}

func readCodeAttribute() CodeAttribute {
	a := CodeAttribute{
		attribute_name_index: readU2(),
		attribute_length: readU4(),
		max_stqck:readU2(),
		max_locals:readU2(),
		code_length:readU4(),
	}
	a.code = readBytes(a.code_length)
	a.exception_table_length = readU2()
	for i:=0;i<a.exception_table_length;i++ {
		readExceptionTable()
	}
	a.attributes_count = readU2()
	for i:=0;i<a.attributes_count;i++ {
		readAttributeInfo()
	}
	return a
}

func readMethodInfo() MethodInfo {
	methodInfo := MethodInfo{
		access_flags:     readU2(),
		name_index:       readU2(),
		descriptor_index: readU2(),
		attributes_count: readU2(),
	}
	var cas []CodeAttribute
	for i:=0;i<methodInfo.attributes_count; i++ {
		ca := readCodeAttribute()
		cas = append(cas, ca)
	}
	methodInfo.ai = cas

	return methodInfo
}
func main() {
	var err error
	bytes, err = ioutil.ReadFile("HelloWorld.class")
	if err != nil {
		panic(err)
	}

	cafebabe := readCafebabe()
	for _, char := range cafebabe {
		fmt.Printf("%x ", char)
	}
	major_version := readU2()
	minor_version := readU2()
	constant_pool_count := readU2()
	fmt.Printf("major = %d, minior = %d\n", major_version, minor_version)
	fmt.Printf("constant_pool_count = %d\n", constant_pool_count)
	var entries []interface{}
	entries = append(entries, nil)
	for i:=0; i< constant_pool_count -1 ; i++ {
		tag := readByte()
		fmt.Printf("[i=%d] tag=%02X\n", i, tag)
		var e interface{}
		switch tag {
		case 0x0a, 0x09:
			e = &ConstantMethodfRef{
				first:readU2(),
				second:readU2(),
				tag:tag,
			}
		case 0x08:
			e = &ConstantString{
				s: readU2(),
				tag:tag,
			}
		case 0x07:
			e = &ConstantClass{
				s: readU2(),
				tag:tag,
			}
		case 0x01:
			ln := readU2()
			e = &ConstantUTF8{
				tag:tag,
				len: ln,
				content: string(readBytes(ln)),
			}
		case 0x0c:
			e = &ConstantNameAndType{
				first:readU2(),
				second:readU2(),
				tag:tag,
			}
		default:
			panic("unknown tag")
		}
		//e.tag = tag
		entries = append(entries, e)
	}

	fmt.Printf("Entries=%d\n", len(entries) -1)
	for i, e := range entries {
		fmt.Printf("[%d] Entry=%#v\n", i, e)
	}

	access_flags := readU2()
	this_class := readU2()
	super_class := readU2()
	interface_count := readU2()
	//interfaces := readU2()
	fields_count := readU2()
	methods_count := readU2()
	fmt.Printf("access_flags=%d\n", access_flags)
	fmt.Printf("this_class=%d\n", this_class)
	fmt.Printf("super_class=%d\n", super_class)
	fmt.Printf("interface_count=%d\n", interface_count)
	//fmt.Printf("interfaces=%d\n", interfaces)
	fmt.Printf("fields_count=%d\n", fields_count)
	fmt.Printf("methods_count=%d\n", methods_count)

	for i:=0;i<methods_count;i++ {
		methodInfo := readMethodInfo()
		entry := getFromCPool(entries, methodInfo.name_index)
		cutf8, ok := entry.(*ConstantUTF8)
		if !ok {
			panic("not ConstantUTF8")
		}
		fmt.Printf("methodInfo '%s'=%v\n", cutf8.content, methodInfo)
	}
	attributes_count := readU2()
	fmt.Printf("attributes_count=%d\n", attributes_count)
	attr := readAttributeInfo()
	fmt.Printf("attribute=%v\n", attr)
	if len(bytes) == byteIndex {
		fmt.Printf("__EOF__\n")
	}
}

func getFromCPool(entries []interface{}, i int) interface{} {
	return entries[i]
}
