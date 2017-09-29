package main

import (
	"io/ioutil"
	"log"
	"os"
)

func main() {
	os.Chdir(`C:\Users\sjon\go\src\github.com\Soreil\arw\samples`)
	buf,err :=ioutil.ReadFile("sr2buf.txt")
	if err != nil {
		panic(err)
	}
	log.Println(buf[:128])

	DecryptSR2(buf,11442017450)
}

func DecryptSR2(buf []byte,key uint32) ([]byte, error){
	var pad [128]uint32

	for i := 0; i < 4; i++ {
		key = key * 0x0edd + 1
		pad[i]=key
	}
	pad[3] = pad[3] << 1 | (pad[0]^pad[2]) >> 31

	for i:=4; i < 127; i++ {
		pad[i] = (pad[i-4]^pad[i-2]) << 1 | (pad[i-3]^pad[i-1]) >> 31
	}

	for i := 127;i < len(buf)+127; i++ {
		or :=  pad[(i+1) & 127] ^ pad[(i+65) & 127]
		pad[i & 127] = or
		buf[i-127] ^= byte(or)
	}

	return buf,nil
}