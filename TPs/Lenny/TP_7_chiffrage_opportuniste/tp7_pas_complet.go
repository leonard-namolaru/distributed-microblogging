package main

import (
	// Using math/rand is not considered cryptographically secure (https://www.practical-go-lessons.com/post/how-to-generate-random-bytes-with-golang-ccc9755gflds70ubqc2g)

	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"fmt"
	"io"
	"log"
	"math/big"
	"net"
)

const a_LENGTH = 768
const a_LENGTH_IN_BYTES = a_LENGTH / 8
const IV_LENGTH = 2

const HOST_URL = "127.0.0.1:8081"

func main() {
	var a, g, p, A big.Int

	aBuffer := make([]byte, a_LENGTH_IN_BYTES)
	_, errorMessage := rand.Read(aBuffer)
	if errorMessage != nil {
		log.Fatal("rand.Read() function : ", errorMessage)
	}

	p.SetString("FFFFFFFFFFFFFFFFC90FDAA22168C234C4C6628B80DC1CD129024E088A67CC74020BBEA63B139B22514A08798E3404DDEF9519B3CD3A431B302B0A6DF25F14374FE1356D6D51C245E485B576625E7EC6F44C42E9A63A36210000000000090563", 16)
	g.SetInt64(2)

	a.SetBytes(aBuffer)

	A.Exp(&g, &a, &p)

	fmt.Printf("A : %s \n", A.String())

	connection, errorMessage := net.Dial("tcp", HOST_URL)
	if errorMessage != nil {
		log.Fatal("Function net.Dial() : ", errorMessage)
	}

	_, errorMessage = connection.Write(A.Bytes())
	if errorMessage != nil {
		log.Fatal("Function connection.Write() : ", errorMessage)
	}

	var B, s big.Int
	buffer := make([]byte, a_LENGTH_IN_BYTES)
	_, errorMessage = io.ReadFull(connection, buffer)
	if errorMessage != nil {
		log.Fatal("Function bufio.NewReader().Read() : ", errorMessage)
	}

	fmt.Printf("\nWe receive B : \n")
	B.SetBytes(buffer)
	fmt.Printf("%s \n", B.String())

	var p1, zero, one big.Int
	zero.SetInt64(0)
	one.SetInt64(1)
	p1.Sub(&p, &one)

	if zero.String() == B.String() || one.String() == B.String() || p1.String() == B.String() {
		log.Fatal("Pbm with B ")
	}

	s.Exp(&B, &a, &p)
	fmt.Printf("s : %s \n", s.String())

	IV := make([]byte, IV_LENGTH)
	_, errorMessage = io.ReadFull(connection, IV)
	if errorMessage != nil {
		log.Fatal("Function bufio.NewReader().Read() : ", errorMessage)
	}

	fmt.Printf("IV : %v \n", IV)

	ciphertext := make([]byte, 4)
	_, errorMessage = io.ReadFull(connection, ciphertext)
	if errorMessage != nil {
		log.Fatal("Function bufio.NewReader().Read() : ", errorMessage)
	}

	fmt.Printf("ciphertext : %v \n", ciphertext)

	h := sha256.New()
	h.Write(s.Bytes())
	k := h.Sum(nil)[0:16]

	fmt.Printf("k : %x \n", k)

	block, errorMessage := aes.NewCipher(k)
	if errorMessage != nil {
		log.Fatal("aes.NewCipher() function : ", errorMessage)
	}

	iv := ciphertext[:IV_LENGTH*8]
	ciphertext = ciphertext[IV_LENGTH*8:]
	mode := cipher.NewCBCDecrypter(block, iv)
	mode.CryptBlocks(ciphertext, ciphertext)
	fmt.Printf("%s\n", ciphertext)
	/*
		ciphertext = ciphertext[IV_LENGTH:]
		mode := cipher.NewCBCDecrypter(block, IV)
		mode.CryptBlocks(ciphertext, ciphertext)
		ciphertext = bytes.TrimRight(ciphertext, "0")

		fmt.Printf("plaintext : %s \n", ciphertext)
	*/
	/*

		IV := make([]byte, 16)
		_, errorMessage = rand.Read(IV)
		if errorMessage != nil {
			log.Fatal("rand.Read() function : ", errorMessage)
		}

		plaintext := make([]byte, 32)
		_, errorMessage = rand.Read(plaintext)
		if errorMessage != nil {
			log.Fatal("rand.Read() function : ", errorMessage)
		}

		fmt.Printf("k : %x \n", k)
		fmt.Printf("IV : %x \n", IV)
		fmt.Printf("plaintext : %x \n", plaintext)

		block, errorMessage := aes.NewCipher(k)
		if errorMessage != nil {
			log.Fatal("aes.NewCipher() function : ", errorMessage)
		}

		ciphertext := make([]byte, IV_LENGTH+len(plaintext))
		mode := cipher.NewCBCEncrypter(block, IV)
		mode.CryptBlocks(ciphertext[IV_LENGTH:], plaintext)
		fmt.Printf("ciphertext : %x \n", ciphertext)

		ciphertext = ciphertext[IV_LENGTH:]
		mode = cipher.NewCBCDecrypter(block, IV)
		mode.CryptBlocks(ciphertext, ciphertext)
		//ciphertext = bytes.TrimRight(ciphertext, "0")

		fmt.Printf("plaintext : %x\n", ciphertext)
	*/
	connection.Close()

}
