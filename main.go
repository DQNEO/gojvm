package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"strings"
)

var cpool ConstantPool
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

type ConstantPoolEntry interface {
	Type() string
	String() string
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

func (c *CONSTANT_Class_info) Type() string { return "Class" }
func (c *CONSTANT_Fieldref_info) Type() string { return "Fieldref" }
func (c *CONSTANT_Methodref_info) Type() string { return "Methodref" }
func (c *CONSTANT_String_info) Type() string { return "String" }
func (c *CONSTANT_NameAndType_info) Type() string { return "NameAndType" }
func (c *CONSTANT_Utf8_info) Type() string { return "UTF8" }

func (c *CONSTANT_Class_info) String() string { return "Class" }
func (c *CONSTANT_Fieldref_info) String() string { return "Fieldref" }
func (c *CONSTANT_Methodref_info) String() string { return "Methodref" }
func (c *CONSTANT_String_info) String() string { return "String" }
func (c *CONSTANT_NameAndType_info) String() string { return "NameAndType" }
func (c *CONSTANT_Utf8_info) String() string { return "UTF8" }

func (c *CONSTANT_Class_info) getName() string {
	return cpool.getUTF8AsString(c.name_index)
}

func (c *CONSTANT_Fieldref_info) getClassInfo() *CONSTANT_Class_info {
	return cpool.getClassInfo(c.class_index)
}

func (c *CONSTANT_Fieldref_info) getNameAndType() *CONSTANT_NameAndType_info {
	return cpool.getNameAndType(c.name_and_type_index)
}

func (c *CONSTANT_Methodref_info) getClassInfo() *CONSTANT_Class_info {
	return cpool.getClassInfo(c.class_index)
}

func (c *CONSTANT_Methodref_info) getNameAndType() *CONSTANT_NameAndType_info {
	return cpool.getNameAndType(c.name_and_type_index)
}


func (c *CONSTANT_NameAndType_info) getName() string {
	return cpool.getUTF8AsString(c.name_index)
}

func (c *CONSTANT_NameAndType_info) getDescriptor() string {
	return cpool.getUTF8AsString(c.descriptor_index)
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

type ConstantPool []ConstantPoolEntry

// https://docs.oracle.com/javase/specs/jvms/se12/html/jvms-4.html#jvms-4.1z
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

	var constant_pool []ConstantPoolEntry
	constant_pool = append(constant_pool, nil)
	for i := u2(0); i < constant_pool_count-1; i++ {
		tag := readByte()
		//debugf("[i=%d] tag=%02X\n", i, tag)
		var e ConstantPoolEntry
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
		debugf("__EOF__\n")
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

func c2s(c interface{}) string {
	switch c.(type) {
	case *CONSTANT_Fieldref_info:
		cf := c.(*CONSTANT_Fieldref_info)
		return fmt.Sprintf("Fieldref\t#%d.#%d",
			cf.class_index, cf.name_and_type_index)
	case *CONSTANT_Methodref_info:
		cm := c.(*CONSTANT_Methodref_info)
		return fmt.Sprintf("Methodref\t#%d.#%d",
			cm.class_index, cm.name_and_type_index)
	case *CONSTANT_Class_info:
		return fmt.Sprintf("Class\t%d", c.(*CONSTANT_Class_info).name_index)
	case *CONSTANT_String_info:
		return fmt.Sprintf("String\t%d", c.(*CONSTANT_String_info).string_index)
	case *CONSTANT_NameAndType_info:
		cn := c.(*CONSTANT_NameAndType_info)
		return fmt.Sprintf("NameAndType\t#%d:#%d", cn.name_index, cn.descriptor_index)
	case *CONSTANT_Utf8_info:
		return fmt.Sprintf("Utf8\t%s", c.(*CONSTANT_Utf8_info).bytes)
	default:
		panic("Unknown constant pool")
	}

}

func debugConstantPool(cp ConstantPool)  {
	for i, c := range cp {
		if i == 0 {
			continue
		}
		debugf(" #%02d = ",i)
		s := c2s(c)
		debugf("%s\n", s)
	}
}

func (cp ConstantPool) get(i u2) interface{} {
	return cp[i]
}

func (cp ConstantPool) getFieldref(id u2) *CONSTANT_Fieldref_info {
	entry := cp.get(id)
	c, ok := entry.(*CONSTANT_Fieldref_info)
	if !ok {
		panic("type mismatch")
	}
	return c
}

func (cp ConstantPool) getMethodref(id u2) *CONSTANT_Methodref_info {
	entry := cp.get(id)
	c, ok := entry.(*CONSTANT_Methodref_info)
	if !ok {
		panic("type mismatch")
	}
	return c
}

func (cp ConstantPool) getClassInfo(id u2) *CONSTANT_Class_info {
	entry := cp.get(id)
	ci, ok := entry.(*CONSTANT_Class_info)
	if !ok {
		panic("type mismatch")
	}
	return ci
}

func (cp ConstantPool) getNameAndType(id u2) *CONSTANT_NameAndType_info {
	entry := cp.get(id)
	c, ok := entry.(*CONSTANT_NameAndType_info)
	if !ok {
		panic("type mismatch")
	}
	return c
}

func (cp ConstantPool) getString(id u2) string {
	entry := cp.get(id)
	c, ok := entry.(*CONSTANT_String_info)
	if !ok {
		panic("type mismatch")
	}

	return cp.getUTF8AsString(c.string_index)
}

func (cp ConstantPool) getUTF8AsString(id u2) string {
	return string(cp.getUTF8Bytes(id))
}

func (cp ConstantPool) getUTF8Bytes(id u2) []byte {
	entry := cp.get(id)
	utf8, ok := entry.(*CONSTANT_Utf8_info)
	if !ok {
		panic("type mismatch")
	}
	return utf8.bytes
}

func debugClassFile(cf *ClassFile) {
	cp := cf.constant_pool
	for _, char := range cf.magic {
		debugf("%x ", char)
	}

	debugf("\n")
	debugf("major_version = %d, minior_version = %d\n", cf.major_version, cf.minor_version)
	debugf("access_flags=%d\n", cf.access_flags)
	thisClassInfo := cp.getClassInfo(cf.this_class)
	debugf("class %s\n", thisClassInfo.getName())
	debugf("  super_class=%s\n", cp.getClassInfo(cf.super_class).getName())

	debugf("Constant pool:\n")
	debugConstantPool(cf.constant_pool)

	debugf("interface_count=%d\n", cf.interface_count)
	//debugf("interfaces=%d\n", interfaces)
	debugf("fields_count=%d\n", cf.fields_count)
	debugf("methods_count=%d\n", cf.methods_count)

	for _, methodInfo := range cf.methods{
		methodName := cp.getUTF8AsString(methodInfo.name_index)
		debugf(" %s:\n", methodName)
		for _, ca  := range methodInfo.ai {
			for _, c := range ca.code {
				debugf(" %02x", c)
			}
		}
		debugf("\n")
	}
	debugf("attributes_count=%d\n", cf.attributes_count)
	debugf("attribute=%v\n", cf.attributes[0])
}

func getByte() byte {
	b := bytes[byteIndex]
	byteIndex++
	return b
}

var stack []interface{}

func push(e interface{}) {
	stack = append(stack, e)
}

func pop() interface{} {
	e := stack[len(stack)-1]
	newStack := stack[0:len(stack)-1]
	stack = newStack
	return e
}

func executeCode(code []byte) {
	debugf("len code=%d\n", len(code))

	byteIndex = 0
	bytes = code
	for _, b := range code {
		debugf("0x%x ", b)
	}
	debugf("\n")
	for {
		if byteIndex >= len(bytes) {
			break
		}
		b := getByte()
		debugf("inst 0x%02x\n", b)
		switch b {
		case 0x12: // ldc
			operand := readByte()
			debugf("  ldc 0x%02x\n", operand)
			push(operand)
		case 0xb1: // return
			debugf("  return\n")
			return
		case 0xb2: // getstatic
			operand := readU2()
			debugf("  getstatic 0x%02x\n", operand)
			fieldRef := cpool.getFieldref(operand)
			classInfo := fieldRef.getClassInfo()
			name := fieldRef.getNameAndType().getName()
			desc := fieldRef.getNameAndType().getDescriptor()
			debugf("   => %s#%s#%s#%s\n", c2s(fieldRef), classInfo.getName(), name, desc)
			push(operand)
		case 0xb6: // invokevirtual
			operand := readU2()
			debugf("  invokevirtual 0x%02x\n", operand)
			methodRef := cpool.getMethodref(operand)
			methodClassInfo := methodRef.getClassInfo()
			methodNameAndType := methodRef.getNameAndType()
			methodName :=  cpool.getUTF8AsString(methodNameAndType.name_index)
			debugf("    invoking %s.%s()\n", methodClassInfo.getName(), methodName) // java/lang/System

			// argument info
			desc := methodNameAndType.getDescriptor()
			desc_args := strings.Split(desc, ";")
			num_args := len(desc_args) - 1
			debugf("    descriptor=%s, num_args=%d\n", desc, num_args)

			arg0ifc := pop()
			arg0id := arg0ifc.(u1)
			arg0 := cpool.getString(u2(arg0id))
			arg0StringValue := arg0
			debugf("    arg0=%s\n",  arg0StringValue)

			// receiverId info
			receiverId := pop()
			// System.out:PrintStream
			fieldRef := cpool.getFieldref(receiverId.(u2))
			fieldClassInfo := fieldRef.getClassInfo()         // class java/lang/System
			fieldNameAndType := fieldRef.getNameAndType()
			fieldName := fieldNameAndType.getName()  // out
			debugf("    receiver=%s.%s %s\n", fieldClassInfo.getName(), fieldName, fieldNameAndType.getDescriptor()) // java/lang/System.out Ljava/io/PrintStream;

			debugf("[Invoking]\n")
			receiver := classMap[fieldClassInfo.getName()].staicfields[fieldName]
			method := classMap[methodClassInfo.getName()].methods[methodName]
			method(receiver, arg0StringValue)
		default:
			panic("Unknown instruction")
		}
		debugf("#  stack=%#v\n", stack)
	}
}

var classMap map[string]*JavaClass

type JavaClass struct {
	staicfields map[string]interface{}
	methods map[string]func(...interface{})
}

type PrintStream struct {
	fp *os.File
}

func (methodInfo MethodInfo) invoke() {
	for _, ca := range methodInfo.ai {
		executeCode(ca.code)
		debugf("#---\n")
	}
}

var debug bool

func debugf(format string, args ...interface{}) {
	if debug {
		fmt.Fprintf(os.Stderr, "# " + format, args...)
	}
}

func initJava() {
	classMap = map[string]*JavaClass{
		"java/lang/System" : &JavaClass{
			staicfields: map[string]interface{}{
				"out": &PrintStream{
					fp: os.Stdout,
				},
			},
		},
		"java/io/PrintStream": &JavaClass{
			methods: map[string]func(...interface{}){
				"println": func(args ...interface{}) {
					ps, ok := args[0].(*PrintStream)
					if !ok {
						panic("Type mismatch")
					}
					s, ok := args[1].(string)
					if !ok {
						panic("Type mismatch")
					}

					fmt.Fprint(ps.fp, s + "\n")
				},
			},
		},
	}
}


func main() {
	debug = true
	initJava()
	cf := parseClassFile("/dev/stdin")
	cpool = cf.constant_pool
	debugClassFile(cf)
	for _, methodInfo := range cf.methods {
		methodName := cf.constant_pool.getUTF8AsString(methodInfo.name_index)
		if methodName == "main" {
			methodInfo.invoke()
		}
	}
}

