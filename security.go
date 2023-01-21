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
	"crypto/sha256"
	"log"
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

func VerifySignature(buf []byte, publicKey *ecdsa.PublicKey) bool {
	length := int((buf[5] << 8) + buf[6])
	signature := buf[BODY_FIRST_BYTE+length:BODY_FIRST_BYTE+length+SIGNATURE_LENGTH]
	var r, s big.Int
	r.SetBytes(signature[:32])
	s.SetBytes(signature[32:])
	hashed := sha256.Sum256(buf[:BODY_FIRST_BYTE+length])
	ok := ecdsa.Verify(publicKey, hashed[:], &r, &s)
	if DEBUG_MODE {
		fmt.Printf("Signature verified : %v\n",ok)
	}
	return ok
}

func CreateSignature(datagram []byte, datagramLength int, privateKey *ecdsa.PrivateKey) []byte {
	hashed := sha256.Sum256(datagram[:datagramLength-SIGNATURE_LENGTH])
	r, s, errorMessage := ecdsa.Sign(rand.Reader, privateKey, hashed[:])
	if errorMessage != nil {
		log.Fatalf("The method ecdsa.Sign() failed In the phase of building a datagram of type %d : %v \n", ROOT_REQUEST_TYPE, errorMessage)
	}
	signature := make([]byte, SIGNATURE_LENGTH)
	r.FillBytes(signature[:32])
	s.FillBytes(signature[32:])

	copy(datagram[datagramLength-SIGNATURE_LENGTH:], signature ) //signature

	myPublicKeyBytes, err := base64.RawStdEncoding.DecodeString(MyPublicKeyEncoded)
	if err != nil {
		panic(err)
	}
	ok := VerifySignature(datagram, ConvertBytesToEcdsaPublicKey(myPublicKeyBytes) )
	if !ok {
		panic(err)
	}

	return datagram
}

func CreatePrivateKeyForEncryption() *ecdsa.PrivateKey{
	privateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		panic(err)
	}
	return privateKey
}

func GeneratePublicEncodedKeyForEncryption(privateKey *ecdsa.PrivateKey) string {
	publicKey := privateKey.PublicKey
	publicKey64Bytes := make([]byte, 64)
	publicKey.X.FillBytes(publicKey64Bytes[:32])
	publicKey.Y.FillBytes(publicKey64Bytes[32:])
	publicKeyEncoded := base64.RawStdEncoding.EncodeToString(publicKey64Bytes)
	return publicKeyEncoded
}

func GenerateSharedKey(publicKeyForEncryptionFromPeer ecdsa.PublicKey, myPrivateKeyForEncryption *ecdsa.PrivateKey) []byte {
	sharedKey, err := publicKeyForEncryptionFromPeer.Curve.ScalarMult(publicKeyForEncryptionFromPeer.X, publicKeyForEncryptionFromPeer.Y, myPrivateKeyForEncryption.D.Bytes())
	if err != nil {
		panic(err)
	}
	return sharedKey.Bytes()
}