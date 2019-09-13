package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"strings"
)

var cf *ClassFile
var cpool ConstantPool

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

func readMagic(br *ByteReader) [4]byte {
	br.byteIndex += 4
	return [4]byte{br.bytes[0], br.bytes[1], br.bytes[2], br.bytes[3]}
}

type u1 uint8
type u2 uint16
type u4 uint32

func  (br *ByteReader) readU1() u1 {
	b := br.bytes[br.byteIndex]
	br.byteIndex++
	return u1(b)
}

func  (br *ByteReader) readU2() u2 {
	left := br.bytes[br.byteIndex]
	right := br.bytes[br.byteIndex+1]
	br.byteIndex += 2

	return u2(u2(left)*256 + u2(right))
}
func  (br *ByteReader) readU4() u4 {
	b1 := br.bytes[br.byteIndex]
	b2 := br.bytes[br.byteIndex+1]
	b3 := br.bytes[br.byteIndex+2]
	b4 := br.bytes[br.byteIndex+3]
	br.byteIndex += 4

	return u4(u4(b1)*256*256*256 + u4(b2)*256*256 + u4(b3)*256 + u4(b4))
}

func  (br *ByteReader) readBytes(n int) []byte {
	r := br.bytes[br.byteIndex : br.byteIndex+n]
	br.byteIndex += n
	return r
}

func readAttributeInfo(br *ByteReader) AttributeInfo {
	a := AttributeInfo{
		attribute_name_index: br.readU2(),
		attribute_length:     br.readU4(),
	}
	a.body = br.readBytes(int(a.attribute_length))
	return a
}

func readExceptionTable(br *ByteReader) {
	br.readU2()
	br.readU2()
	br.readU2()
	br.readU2()
}

func readCodeAttribute(br *ByteReader) CodeAttribute {
	a := CodeAttribute{
		attribute_name_index: br.readU2(),
		attribute_length:     br.readU4(),
		max_stqck:            br.readU2(),
		max_locals:           br.readU2(),
		code_length:          br.readU4(),
	}
	a.code = br.readBytes(int(a.code_length))
	a.exception_table_length = br.readU2()
	for i := u2(0); i < a.exception_table_length; i++ {
		readExceptionTable(br)
	}
	a.attributes_count = br.readU2()
	for i := u2(0); i < a.attributes_count; i++ {
		readAttributeInfo(br)
	}
	return a
}

func (br *ByteReader) readMethodInfo() MethodInfo {
	methodInfo := MethodInfo{
		access_flags:     br.readU2(),
		name_index:       br.readU2(),
		descriptor_index: br.readU2(),
		attributes_count: br.readU2(),
	}
	var cas []CodeAttribute
	for i := u2(0); i < methodInfo.attributes_count; i++ {
		ca := readCodeAttribute(br)
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
	methodMap           map[u2]*MethodInfo
}

type ByteReader struct {
	bytes []byte
	byteIndex int
}

func parseClassFile(filename string) *ClassFile {
	bytes, err := ioutil.ReadFile(filename)
	if err != nil {
		panic(err)
	}

	br := &ByteReader{
		byteIndex:0,
		bytes: bytes,
	}
	magic := readMagic(br) // cafebabe
	minor_version := br.readU2()
	major_version := br.readU2()
	constant_pool_count := br.readU2()

	var constant_pool []ConstantPoolEntry
	constant_pool = append(constant_pool, nil)
	for i := u2(0); i < constant_pool_count-1; i++ {
		tag := br.readU1()
		//debugf("[i=%d] tag=%02X\n", i, tag)
		var e ConstantPoolEntry
		switch tag {
		case 0x09:
			e = &CONSTANT_Fieldref_info{
				class_index:         br.readU2(),
				name_and_type_index: br.readU2(),
				tag:                 tag,
			}
		case 0x0a:
			e = &CONSTANT_Methodref_info{
				class_index:         br.readU2(),
				name_and_type_index: br.readU2(),
				tag:                 tag,
			}
		case 0x08:
			e = &CONSTANT_String_info{
				string_index: br.readU2(),
				tag:          tag,
			}
		case 0x07:
			e = &CONSTANT_Class_info{
				name_index: br.readU2(),
				tag:        tag,
			}
		case 0x01:
			ln := br.readU2()
			e = &CONSTANT_Utf8_info{
				tag:    tag,
				length: ln,
				bytes:  br.readBytes(int(ln)),
			}
		case 0x0c:
			e = &CONSTANT_NameAndType_info{
				name_index:       br.readU2(),
				descriptor_index: br.readU2(),
				tag:              tag,
			}
		default:
			panic("unknown tag")
		}
		//e.tag = tag
		constant_pool = append(constant_pool, e)
	}

	access_flags := br.readU2()
	this_class := br.readU2()
	super_class := br.readU2()
	interface_count := br.readU2()
	fields_count := br.readU2()
	methods_count := br.readU2()

	var methods []MethodInfo = make([]MethodInfo, methods_count)
	for i := u2(0); i < methods_count; i++ {
		methodInfo := br.readMethodInfo()
		methods[i] = methodInfo
	}
	attributes_count := br.readU2()
	var attributes []AttributeInfo
	for i := u2(0); i < attributes_count; i++ {
		attr := readAttributeInfo(br)
		attributes = append(attributes, attr)
	}
	if len(bytes) == br.byteIndex {
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
		return fmt.Sprintf("Fieldref\t#0x%02x.#0x%02x",
			cf.class_index, cf.name_and_type_index)
	case *CONSTANT_Methodref_info:
		cm := c.(*CONSTANT_Methodref_info)
		return fmt.Sprintf("Methodref\t#0x%02x.#0x%02x",
			cm.class_index, cm.name_and_type_index)
	case *CONSTANT_Class_info:
		return fmt.Sprintf("Class\t0x%02x", c.(*CONSTANT_Class_info).name_index)
	case *CONSTANT_String_info:
		return fmt.Sprintf("String\t0x%02x", c.(*CONSTANT_String_info).string_index)
	case *CONSTANT_NameAndType_info:
		cn := c.(*CONSTANT_NameAndType_info)
		return fmt.Sprintf("NameAndType\t#0x%02x:#0x%02x", cn.name_index, cn.descriptor_index)
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
		s := c2s(c)
		debugf(" #0x%02x = %s\n",i, s)
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
		panic(fmt.Sprintf("CONSTANT_String_info expected, but got %T", entry))
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

	cf.methodMap = make(map[u2]*MethodInfo)
	for _, methodInfo := range cf.methods{
		cf.methodMap[methodInfo.name_index] = &methodInfo
		methodName := cp.getUTF8AsString(methodInfo.name_index)
		debugf(" #0x%02x %s:\n", methodInfo.name_index, methodName)
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

func executeCode(code []byte, localvars []interface{}) {
	debugf("len code=%d\n", len(code))

	br := &ByteReader{
		bytes: code,
	}
	for _, b := range code {
		debugf("0x%02x ", b)
	}
	debugf("\n")

	for {
		if br.byteIndex >= len(br.bytes) {
			break
		}
		b := br.readU1()
		debugf("inst 0x%02x\n", b)
		// https://docs.oracle.com/javase/specs/jvms/se12/html/jvms-6.html#jvms-6.5
		switch b {
		case 0x10: // bipush
			operand := br.readU1()
			debugf("  bipush 0x%02x\n", operand)
			push(operand)
		case 0x12: // ldc
			operand := br.readU1()
			debugf("  ldc 0x%02x\n", operand)
			push(operand)
		case 0x1a: // iload_0
			debugf("  iload_0\n")
			arg := localvars[0]
			push(arg)
		case 0x1b: // iload_1
			debugf("  iload_1\n")
			arg := localvars[1]
			push(arg)
		case 0x3c: // istore_1
			arg := pop()
			debugf("  istore_1\n")
			localvars[1] = arg
		case 0x60: // iadd
			debugf("  iadd\n")
			arg0 := pop().(u1)
			arg1 := pop().(u1)
			ret := arg0 + arg1
			debugf("    result=%d\n", ret)
			push(ret)
		case 0xac: // ireturn
			ret := pop().(u1)
			rax = ret
			return
		case 0xb1: // return
			debugf("  return\n")
			return
		case 0xb2: // getstatic
			operand := br.readU2()
			debugf("  getstatic 0x%02x\n", operand)
			fieldRef := cpool.getFieldref(operand)
			classInfo := fieldRef.getClassInfo()
			name := fieldRef.getNameAndType().getName()
			desc := fieldRef.getNameAndType().getDescriptor()
			debugf("   => %s#%s#%s#%s\n", c2s(fieldRef), classInfo.getName(), name, desc)
			push(operand)
		case 0xb6: // invokevirtual
			operand := br.readU2()
			debugf("  invokevirtual 0x%02x\n", operand)
			methodRef := cpool.getMethodref(operand)
			methodClassInfo := methodRef.getClassInfo()
			methodNameAndType := methodRef.getNameAndType()
			methodName :=  cpool.getUTF8AsString(methodNameAndType.name_index)
			debugf("    invoking %s.%s()\n", methodClassInfo.getName(), methodName) // java/lang/System

			// argument info
			desc := methodNameAndType.getDescriptor()
			var arg0 interface{}
			var num_args int
			if strings.Contains(desc, ";") {
				desc_args := strings.Split(desc, ";")
				num_args = len(desc_args) - 1
				debugf("    descriptor=%s, num_args=%d\n", desc, num_args)
				arg0ifc := pop()
				debugf("    arg0ifc=%T, %v\n", arg0ifc, arg0ifc)
				arg0id := u2(arg0ifc.(u1))
				arg0c := cpool.getString(u2(arg0id))
				arg0 = arg0c
			} else {
				num_args = strings.Count(desc, "I")
				debugf("    descriptor=%s, num_args=%d\n", desc, num_args)
				arg0ifc := pop()
				debugf("    arg0ifc=%T, %v\n", arg0ifc, arg0ifc)
				arg0 = arg0ifc.(u1)
			}
			debugf("    arg0=%v\n", arg0)

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
			method(receiver, arg0)
		case 0xb8: // invokestatic
			// https://docs.oracle.com/javase/specs/jvms/se12/html/jvms-6.html#jvms-6.5.invokestatic
			indexbyte1 := br.readU1()
			indexbyte2 := br.readU1()
			index := (indexbyte1 << 8) | indexbyte2
			debugf("  invokestatic 0x%02x, 0x%02x => 0x%02x\n", indexbyte1, indexbyte2, index)
			methodRef := cpool.getMethodref(u2(index))
			methodClassInfo := methodRef.getClassInfo()
			methodNameAndType := methodRef.getNameAndType()
			methodName :=  cpool.getUTF8AsString(methodNameAndType.name_index)
			debugf("    invoking #0x%02x %s.%s()\n",
				methodNameAndType.name_index,  methodClassInfo.getName(), methodName) // java/lang/System

			// argument info
			desc := methodNameAndType.getDescriptor() // (II)I
			num_args := 2 // (II) => 2
			debugf("    descriptor=%s, num_args=%d\n", desc, num_args)
			arg1 := pop().(u1)
			arg2 := pop().(u1)
			methodInfo,ok := cf.methodMap[methodNameAndType.name_index]
			if !ok {
				panic("Method not found")
			}
			params := []interface{}{arg1, arg2}
			methodInfo.invoke(params)
			debugf("returned back\n")
			ret := rax
			debugf("sum %d = %d + %d\n", ret, arg1, arg2)
			push(ret)
		default:
			panic(fmt.Sprintf("Unknown instruction: 0x%02X", b))
		}
		debugf("#  stack=%#v\n", stack)
	}
}

var rax interface{}
var rbx interface{}

var classMap map[string]*JavaClass

type JavaClass struct {
	staicfields map[string]interface{}
	methods map[string]func(...interface{})
}

type PrintStream struct {
	fp *os.File
}

func (methodInfo MethodInfo) invoke(localvars []interface{}) {
	debugf("# in method %s\n", cf.constant_pool.getUTF8AsString(methodInfo.name_index))
	for _, ca := range methodInfo.ai {
		executeCode(ca.code, localvars)
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
					fmt.Fprintln(ps.fp, args[1])
				},
			},
		},
	}
}


func main() {
	debug = false
	initJava()
	cf = parseClassFile("/dev/stdin")
	cpool = cf.constant_pool
	debugClassFile(cf)
	for _, methodInfo := range cf.methods {
		methodName := cf.constant_pool.getUTF8AsString(methodInfo.name_index)
		if methodName == "main" {
			var localvars []interface{} = make([]interface{}, 16)
			methodInfo.invoke(localvars)
		}
	}
}

