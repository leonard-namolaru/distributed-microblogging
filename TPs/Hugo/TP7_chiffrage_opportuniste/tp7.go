package main

import (
	"crypto/rand"
	//"encoding/base64"
	"fmt"
	"io"
	"log"
	"math/big"
	"net"
	"crypto/sha256"
	"crypto/aes"
	"crypto/cipher"
	"encoding/hex"
)

var p, g, p1, zero, one big.Int

const NB_BITS = 768
const NB_BYTES = NB_BITS / 8

func main() {
	inita()

	connection, err := net.Dial("tcp", fmt.Sprintf("%v:%v", "127.0.0.1", "8081"))
	if err != nil {
		log.Fatalf("net.Dial() : %v\n", err)
	}

	buffer_of_bits := make([]byte, NB_BYTES)
	_, errorMessage := rand.Read(buffer_of_bits)
	if errorMessage != nil {
		log.Fatal("rand.Read() function : ", errorMessage)
	}

	var a, A, B, s big.Int
	a.SetBytes(buffer_of_bits)
	A.Exp(&g, &a, &p)

	//str := base64.StdEncoding.EncodeToString(A.Bytes())
	fmt.Printf("calcul du A : %v\n\n", A.String())

	_, err = connection.Write(A.Bytes())
	if err != nil {
		log.Fatal("Function connection.Write() : ", err)
	}

	buffer_of_bits = make([]byte, NB_BYTES)
	_, err = io.ReadFull(connection, buffer_of_bits)
	if err != nil {
		log.Fatal("Function io.ReadFull() : ", err)
	}

	B.SetBytes(buffer_of_bits)
	//str = base64.StdEncoding.EncodeToString(B.Bytes())
	fmt.Printf("calcul du B : %v\n\n", B.String())

	if zero.String() == B.String() || one.String() == B.String() || p1.String() == B.String() {
		log.Fatal("B trivial")
	}

	s.Exp(&B, &a, &p)

	//str = base64.StdEncoding.EncodeToString(s.Bytes())
	fmt.Printf("calcul du s : %v\n\n", s.String())


	iv := make([]byte, 16)
	_, err = io.ReadFull(connection, iv)
	if err != nil {
		log.Fatal("Function io.ReadFull() : ", err)
	}
	fmt.Printf("iv : %v\n\n", iv)

	ciphertext := make([]byte, 32)
	_, err = io.ReadFull(connection, ciphertext)
	if err != nil {
		log.Fatal("Function io.ReadFull() : ", err)
	}
	//fmt.Printf("msg_chiffre : %v\n\n", ciphertext)

	sbytes := make([]byte, 768/8)
	s.FillBytes(sbytes)
	h := sha256.New()
	h.Write(sbytes)
	k := h.Sum(nil)[0:16]

	fmt.Printf("clé en octets : %v\n\n", k)
	fmt.Printf("clé en hex : %v\n\n", hex.EncodeToString(k))


	block, err := aes.NewCipher(k)
	if err != nil {
		panic(err)
	}

	if len(ciphertext) < aes.BlockSize {
		panic("ciphertext too short")
	}

	//ciphertext = ciphertext[]

	// CBC mode always works in whole blocks.
	if len(ciphertext)%aes.BlockSize != 0 {
		panic("ciphertext is not a multiple of the block size")
	}

	mode := cipher.NewCBCDecrypter(block, iv)

	// CryptBlocks can work in-place if the two arguments are the same.
	mode.CryptBlocks(ciphertext, ciphertext)

	fmt.Printf("ciphertext: %s\n", ciphertext)



}

func inita(){
	p.SetString("FFFFFFFFFFFFFFFFC90FDAA22168C234C4C6628B80DC1CD129024E088A67CC74020BBEA63B139B22514A08798E3404DDEF9519B3CD3A431B302B0A6DF25F14374FE1356D6D51C245E485B576625E7EC6F44C42E9A63A36210000000000090563", 16)
	g.SetInt64(2)
	zero.SetInt64(0)
	one.SetInt64(1)
	p1.Sub(&p, &one)
}
