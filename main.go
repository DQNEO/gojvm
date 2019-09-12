package main

import (
	"fmt"
	"io/ioutil"
)

var bytes []byte
var byteIndex int = 0

type ExceptionTable struct {
	start_pc   u2
	end_pc     u2
	handler_pc u2
	catch_type u2
}

type CodeAttribute struct {
	attribute_name_index   u2
	attribute_length       u4
	max_stqck              u2
	max_locals             u2
	code_length            u4
	code                   []byte
	exception_table_length u2
	exception_tables       []ExceptionTable
	attributes_count       u2
	attribute_infos        []AttributeInfo
}

type AttributeInfo struct {
	attribute_name_index u2
	attribute_length     u4
	body                 []byte
}

type MethodInfo struct {
	access_flags     u2
	name_index       u2
	descriptor_index u2
	attributes_count u2
	ai               []CodeAttribute
}

type CONSTANT_Class_info struct {
	tag        u1
	name_index u2
}

type CONSTANT_Fieldref_info struct {
	tag                 u1
	class_index         u2
	name_and_type_index u2
}

type CONSTANT_Methodref_info struct {
	tag                 u1
	class_index         u2
	name_and_type_index u2
}

type CONSTANT_String_info struct {
	tag          u1
	string_index u2
}

type CONSTANT_NameAndType_info struct {
	tag              u1
	name_index       u2
	descriptor_index u2
}

type CONSTANT_Utf8_info struct {
	tag    u1
	length u2
	bytes  []byte
}

func readCafebabe() [4]byte {
	byteIndex += 4
	return [4]byte{bytes[0], bytes[1], bytes[2], bytes[3]}
}

type u1 byte
type u2 uint16
type u4 uint32

func readU2() u2 {
	left := bytes[byteIndex]
	right := bytes[byteIndex+1]
	byteIndex += 2

	return u2(u2(left)*256 + u2(right))
}
func readU4() u4 {
	b1 := bytes[byteIndex]
	b2 := bytes[byteIndex+1]
	b3 := bytes[byteIndex+2]
	b4 := bytes[byteIndex+3]
	byteIndex += 4

	return u4(u4(b1)*256*256*256 + u4(b2)*256*256 + u4(b3)*256 + u4(b4))
}

func readBytes(n int) []byte {
	r := bytes[byteIndex : byteIndex+n]
	byteIndex += n
	return r
}
func readByte() u1 {
	b := bytes[byteIndex]
	byteIndex++
	return u1(b)
}

func readAttributeInfo() AttributeInfo {
	a := AttributeInfo{
		attribute_name_index: readU2(),
		attribute_length:     readU4(),
	}
	a.body = readBytes(int(a.attribute_length))
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
		attribute_length:     readU4(),
		max_stqck:            readU2(),
		max_locals:           readU2(),
		code_length:          readU4(),
	}
	a.code = readBytes(int(a.code_length))
	a.exception_table_length = readU2()
	for i := u2(0); i < a.exception_table_length; i++ {
		readExceptionTable()
	}
	a.attributes_count = readU2()
	for i := u2(0); i < a.attributes_count; i++ {
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
	for i := u2(0); i < methodInfo.attributes_count; i++ {
		ca := readCodeAttribute()
		cas = append(cas, ca)
	}
	methodInfo.ai = cas

	return methodInfo
}

type ConstantPool []interface{}

// https://docs.oracle.com/javase/specs/jvms/se11/html/jvms-4.html#jvms-4.1
type ClassFile struct {
	magic               [4]byte
	minor_version       u2
	major_version       u2
	constant_pool_count u2
	constant_pool       ConstantPool
	access_flags        u2
	this_class          u2
	super_class         u2
	interface_count     u2
	fields_count        u2
	methods_count       u2
	methods             []MethodInfo
	attributes_count    u2
	attributes          []AttributeInfo
}

func parseClassFile(filename string) *ClassFile {
	var err error
	bytes, err = ioutil.ReadFile(filename)
	if err != nil {
		panic(err)
	}

	magic := readCafebabe()
	minor_version := readU2()
	major_version := readU2()
	constant_pool_count := readU2()

	var constant_pool []interface{}
	constant_pool = append(constant_pool, nil)
	for i := u2(0); i < constant_pool_count-1; i++ {
		tag := readByte()
		//fmt.Printf("[i=%d] tag=%02X\n", i, tag)
		var e interface{}
		switch tag {
		case 0x09:
			e = &CONSTANT_Fieldref_info{
				class_index:         readU2(),
				name_and_type_index: readU2(),
				tag:                 tag,
			}
		case 0x0a:
			e = &CONSTANT_Methodref_info{
				class_index:         readU2(),
				name_and_type_index: readU2(),
				tag:                 tag,
			}
		case 0x08:
			e = &CONSTANT_String_info{
				string_index: readU2(),
				tag:          tag,
			}
		case 0x07:
			e = &CONSTANT_Class_info{
				name_index: readU2(),
				tag:        tag,
			}
		case 0x01:
			ln := readU2()
			e = &CONSTANT_Utf8_info{
				tag:    tag,
				length: ln,
				bytes:  readBytes(int(ln)),
			}
		case 0x0c:
			e = &CONSTANT_NameAndType_info{
				name_index:       readU2(),
				descriptor_index: readU2(),
				tag:              tag,
			}
		default:
			panic("unknown tag")
		}
		//e.tag = tag
		constant_pool = append(constant_pool, e)
	}

	access_flags := readU2()
	this_class := readU2()
	super_class := readU2()
	interface_count := readU2()
	fields_count := readU2()
	methods_count := readU2()

	var methods []MethodInfo = make([]MethodInfo, methods_count)
	for i := u2(0); i < methods_count; i++ {
		methodInfo := readMethodInfo()
		methods[i] = methodInfo
	}
	attributes_count := readU2()
	var attributes []AttributeInfo
	for i := u2(0); i < attributes_count; i++ {
		attr := readAttributeInfo()
		attributes = append(attributes, attr)
	}
	if len(bytes) == byteIndex {
		fmt.Printf("__EOF__\n")
	}

	return &ClassFile{
		magic:               magic,
		minor_version:       minor_version,
		major_version:       major_version,
		constant_pool_count: constant_pool_count,
		constant_pool:       constant_pool,
		access_flags:        access_flags,
		this_class:          this_class,
		super_class:         super_class,
		interface_count:     interface_count,
		fields_count:        fields_count,
		methods_count:       methods_count,
		methods:             methods,
		attributes_count:    attributes_count,
		attributes:          attributes,
	}
}

func debugConstantPool(cp []interface{}) {
	for i, c := range cp {
		if i == 0 {
			continue
		}
		fmt.Printf(" #%02d = ",i)
		switch c.(type) {
		case *CONSTANT_Fieldref_info:
			cf := c.(*CONSTANT_Fieldref_info)
			fmt.Printf("Fieldref\t#%d.#%d\n",
				cf.class_index, cf.name_and_type_index)
		case *CONSTANT_Methodref_info:
			cm := c.(*CONSTANT_Methodref_info)
			fmt.Printf("Methodref\t#%d.#%d\n",
				cm.class_index, cm.name_and_type_index)
		case *CONSTANT_Class_info:
			fmt.Printf("Class\t%d\n", c.(*CONSTANT_Class_info).name_index)
		case *CONSTANT_String_info:
			fmt.Printf("String\t%d\n", c.(*CONSTANT_String_info).string_index)
		case *CONSTANT_NameAndType_info:
			cn := c.(*CONSTANT_NameAndType_info)
			fmt.Printf("NameAndType\t#%d:#%d\n", cn.name_index, cn.descriptor_index)
		case *CONSTANT_Utf8_info:
			fmt.Printf("Utf8\t%s\n", c.(*CONSTANT_Utf8_info).bytes)
		default:
			panic("Unknown constant pool")
		}
	}

}

func (cp ConstantPool) get(i u2) interface{} {
	return cp[i]
}

func (cp ConstantPool) getClassInfo(id u2) *CONSTANT_Class_info {
	entry := cp.get(id)
	ci, ok := entry.(*CONSTANT_Class_info)
	if !ok {
		panic("type mismatch")
	}
	return ci
}


func (cp ConstantPool) getUTF8Byttes(id u2) []byte {
	entry := cp.get(id)
	utf8, ok := entry.(*CONSTANT_Utf8_info)
	if !ok {
		panic("type mismatch")
	}
	return utf8.bytes
}

func debugClassFile(cf *ClassFile) {
	for _, char := range cf.magic {
		fmt.Printf("%x ", char)
	}

	fmt.Printf("\n")
	fmt.Printf("major_version = %d, minior_version = %d\n", cf.major_version, cf.minor_version)
	fmt.Printf("access_flags=%d\n", cf.access_flags)
	ci := cf.constant_pool.getClassInfo(cf.this_class)
	fmt.Printf("class %s\n", cf.constant_pool.getUTF8Byttes(ci.name_index))
	fmt.Printf("  super_class=%d\n", cf.super_class)

	fmt.Printf("Constant pool:\n")
	debugConstantPool(cf.constant_pool)

	fmt.Printf("interface_count=%d\n", cf.interface_count)
	//fmt.Printf("interfaces=%d\n", interfaces)
	fmt.Printf("fields_count=%d\n", cf.fields_count)
	fmt.Printf("methods_count=%d\n", cf.methods_count)

	for _, methodInfo := range cf.methods{
		methodName := cf.constant_pool.getUTF8Byttes(methodInfo.name_index)
		fmt.Printf(" %s:\n", methodName)
		for _, ca  := range methodInfo.ai {
			for _, c := range ca.code {
				fmt.Printf(" %x", c)
			}
		}
		fmt.Printf("\n")
	}
	fmt.Printf("attributes_count=%d\n", cf.attributes_count)
	fmt.Printf("attribute=%v\n", cf.attributes[0])
}

func main() {
	cf := parseClassFile("HelloWorld.class")
	debugClassFile(cf)
}

