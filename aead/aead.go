/*
package aead uses aead crypto to encrypt and authenticate context composed of a plaintext metadata string and a plaintext data string.
An encryption results in a string literal of the form <b64data>.<b64ciphertext>.<b64nonce>. The user of this package must supply a
crypto.AEAD created with the same key in order to encrypt, transmit and decrypt a literal produced by Encrypt.
*/
package aead

import (
	"bytes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"strings"
)

//Encrypt generates an AEAD literal of the form <b64data>.<b64ciphertext>.<b64nonce> given an AEAD
func Encrypt(aeadCipher cipher.AEAD, data, plaintext string) (string, error) {

	var (
		nonce         = make([]byte, aeadCipher.NonceSize())
		ciphertext    []byte
		b64data       []byte
		b64ciphertext []byte
		b64nonce      []byte
		buf           bytes.Buffer
		err           error
	)

	//A nonce of the length required by the AEAD is generated
	_, err = rand.Read(nonce)
	if err != nil {
		return "", err
	}

	//Seal encrypts the plaintext using the key and nonce and appends an authentication code for the additional data
	ciphertext = aeadCipher.Seal(ciphertext, nonce, []byte(plaintext), []byte(data))

	//Base64 Encode data, ciphertext and nonce
	b64data = make([]byte, base64.StdEncoding.EncodedLen(len([]byte(data))))
	base64.StdEncoding.Encode(b64data, []byte(data))
	b64ciphertext = make([]byte, base64.StdEncoding.EncodedLen(len(ciphertext)))
	base64.StdEncoding.Encode(b64ciphertext, ciphertext)
	b64nonce = make([]byte, base64.StdEncoding.EncodedLen(len(nonce)))
	base64.StdEncoding.Encode(b64nonce, nonce)

	//Compose a <b64data>.<b64ciphertext>.<b64nonce> literal
	buf.Write(b64data)
	buf.Write([]byte("."))
	buf.Write(b64ciphertext)
	buf.Write([]byte("."))
	buf.Write(b64nonce)

	//Return the AEAD literal
	return string(buf.Bytes()), nil
}

//Decrypt decrypts an AEAD literal of the form <b64data>.<b64ciphertext>.<b64nonce> given an AEAD
//producing data, plaintext
func Decrypt(aeadCipher cipher.AEAD, literal string) (string, string, error) {
	var (
		literalSubStrings []string
		data              []byte
		ciphertext        []byte
		nonce             []byte
		plaintext         []byte
		err               error
	)

	//Split the literal into its base64 encoded data, ciphertext and nonce components
	literalSubStrings = strings.Split(literal, ".")
	if len(literalSubStrings) != 3 {
		return "", "", fmt.Errorf("Bad AEAD Literal: %v\n", literal)
	}

	//Decode the data, ciphertext and nonce
	data, err = base64.StdEncoding.DecodeString(literalSubStrings[0])
	if err != nil {
		return "", "", fmt.Errorf("Decode data failed: %v\n", literal)
	}
	ciphertext, err = base64.StdEncoding.DecodeString(literalSubStrings[1])
	if err != nil {
		return "", "", fmt.Errorf("Decode ciphertext failed: %v\n", literal)
	}
	nonce, err = base64.StdEncoding.DecodeString(literalSubStrings[2])
	if err != nil {
		return "", "", fmt.Errorf("Decode nonce failed: %v\n", literal)
	}

	//Open validates the integrity of the additional data using the authentication code appended to the ciphertext
	//and, if valid, decrypts the ciphertext
	plaintext, err = aeadCipher.Open(plaintext, nonce, ciphertext, data)
	if err != nil {
		return "", "", err
	}
	return string(data), string(plaintext), nil
}
