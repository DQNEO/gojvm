package main

import (
	"fmt"
	"io/ioutil"
)

var bytes []byte
var byteIndex int = 0

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
	for i:=0; i< constant_pool_count -1; i++ {
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
			readByte()
		}
		//e.tag = tag
		entries = append(entries, e)
	}

	fmt.Printf("Entries=%d\n", len(entries))
	for _, e := range entries {
		fmt.Printf("Entry=%#v\n", e)
	}
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
