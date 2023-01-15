package main

import (
	"crypto/aes"
	"io"
	"errors"
	"crypto/cipher"
	"crypto/rand"
	"encoding/pem"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/x509"
	"strings"
	"bytes"
	"math/big"
	"io/fs"
	"io/ioutil"
	"encoding/base64"
	"fmt"
)

func Encrypt(key []byte, plainText []byte) []byte {

	//Create a new AES cipher using the key
	block, err := aes.NewCipher(key)

	//IF NewCipher failed, exit:
	if err != nil {
		panic(err)
	}

	//Make the cipher text a byte array of size BlockSize + the length of the message
	cipherText := make([]byte, aes.BlockSize+len(plainText))

	//iv is the ciphertext up to the blocksize (16)
	iv := cipherText[:aes.BlockSize]
	if _, err = io.ReadFull(rand.Reader, iv); err != nil {
		panic(err)
	}

	//Encrypt the data:
	stream := cipher.NewCFBEncrypter(block, iv)
	stream.XORKeyStream(cipherText[aes.BlockSize:], plainText)

	//Return string encoded in base64
	return cipherText
}

func Decrypt(key []byte, cipherText []byte) []byte {
	//Create a new AES cipher with the key and encrypted message
	block, err := aes.NewCipher(key)

	//IF NewCipher failed, exit:
	if err != nil {
		panic(err)
	}

	//IF the length of the cipherText is less than 16 Bytes:
	if len(cipherText) < aes.BlockSize {
		err := errors.New("Ciphertext block size is too short!")
		panic(err)
	}

	iv := cipherText[:aes.BlockSize]
	cipherText = cipherText[aes.BlockSize:]

	//Decrypt the message
	stream := cipher.NewCFBDecrypter(block, iv)
	stream.XORKeyStream(cipherText, cipherText)

	return cipherText // cipherText == plainText
}

func CreateOrFindPrivateKey(fileInfo fs.FileInfo, err error) *ecdsa.PrivateKey {
	keyForFile := []byte("asuperstrong32bitpasswordgohere!") //32 bit key for AES-256

	if err != nil || fileInfo.Size() == 0 {

		privateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
		privateKeyDr, err := x509.MarshalECPrivateKey(privateKey)
		if err != nil {
			panic(err)
		}
		privPEM := pem.EncodeToMemory(
			&pem.Block{
				Type:  "EC PRIVATE KEY",
				Bytes: privateKeyDr,
			},
		)

		cipherPrivPEM := Encrypt(keyForFile,privPEM)

		err = ioutil.WriteFile(NAME_FILE_PRIVATE_KEY, cipherPrivPEM, 0644)
		if err != nil {
			panic(err)
		}

	}

	cipherData, err := ioutil.ReadFile(NAME_FILE_PRIVATE_KEY)
	if err != nil {
		panic(err)
	}

	plainData := Decrypt(keyForFile, cipherData)

	lines := strings.Split(string(plainData), "\n")
	lines = lines[1 : len(lines)-2]

	privateKeyString := strings.Join(lines, "")

	if DEBUG_MODE {
		fmt.Printf("privateKeyString : %v\n", privateKeyString)
	}

	// Create the private key
	privateKey, err := ecdsa.GenerateKey(elliptic.P256(), bytes.NewReader([]byte(privateKeyString)))
	if err != nil {
		panic(err)
	}

	return privateKey
}

func CreatePublicKeyEncoded(privateKey *ecdsa.PrivateKey) string {

	publicKey, _ := privateKey.Public().(*ecdsa.PublicKey)
	publicKey64Bytes := make([]byte, 64)
	publicKey.X.FillBytes(publicKey64Bytes[:32])
	publicKey.Y.FillBytes(publicKey64Bytes[32:])
	publicKeyEncoded := base64.RawStdEncoding.EncodeToString(publicKey64Bytes)

	if DEBUG_MODE {
		fmt.Printf("Our public key : %s\n", publicKeyEncoded)
	}

	return publicKeyEncoded
}

func ConvertBytesToEcdsaPublicKey(keyBytes []byte) *ecdsa.PublicKey {
	var x, y big.Int
	x.SetBytes(keyBytes[:32])
	y.SetBytes(keyBytes[32:])
	publicKey := &ecdsa.PublicKey{
		Curve: elliptic.P256(),
		X: &x,
		Y: &y,
	}
	return publicKey
}