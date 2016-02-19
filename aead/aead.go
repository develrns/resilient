/*
package aead uses AEAD crypto with AES keys to encrypt and authenticate content composed of a plaintext metadata string and a plaintext data string.
An encryption results in a string literal of the form <b64URLmetadata>.<b64URLciphertext>.<b64URLnonce>.
*/
package aead

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"strings"
)

/*
NewAEADCipher creates a new AEAD cipher using the provided AES key.
The key argument should be either 16, 24, or 32 bytes to select AES-128, AES-192, or AES-256.

If the key is nil, a new 32 byte AES key is generated.
This option is used when the scope of key use is limited to within a single program execution.
*/
func NewAEADCipher(key []byte) (cipher.AEAD, error) {
	var (
		keyval      []byte
		cipherBlock cipher.Block
		aeadCipher  cipher.AEAD
		err         error
	)

	//If no key is provided, generate one.
	if key == nil {
		keyval = make([]byte, 32)
		_, err = rand.Read(keyval)
		if err != nil {
			return nil, err
		}
	} else {
		switch len(key) {
		case 16, 24, 36:
		default:
			return nil, fmt.Errorf("An aead key must be of length 16. 24, or 32. This key is of length: ", len(key))
		}
		keyval = key
	}

	//The key is used to create an AES Cipher Block
	cipherBlock, err = aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	//The AES Cipher Block is used to create an AEAD GCM which is a 128-bit, block cipher wrapped
	//in a Galois Counter Mode with the standard nonce length.
	//This is used to encrypt/decrypt all subscriber identifiers in the hidden fields of TBD 2nd Factor Selection Forms.
	aeadCipher, err = cipher.NewGCM(cipherBlock)
	if err != nil {
		return nil, err
	}
	return aeadCipher, nil
}

/*
Encrypt generates a literal of the form <b64URLmetadata>.<b64URLciphertext>.<b64URLnonce> given an AEAD cipher, a metadata string and a data
string. Only the data is encrypted - the metadata must be appropriate to expose in the clear. Each call generates a random
nonce of the length required by the cipher.
*/
func Encrypt(aeadCipher cipher.AEAD, metadata, data string) (string, error) {

	var (
		nonce         = make([]byte, aeadCipher.NonceSize())
		ciphertext    []byte
		b64metadata   []byte
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

	//Seal encrypts the data using the aeadCipher's key and the nonce and appends an authentication code for the metadata
	ciphertext = aeadCipher.Seal(ciphertext, nonce, []byte(data), []byte(metadata))

	//Base64 Encode metadata, ciphertext and nonce
	b64metadata = make([]byte, base64.URLEncoding.EncodedLen(len([]byte(metadata))))
	base64.StdEncoding.Encode(b64metadata, []byte(metadata))
	b64ciphertext = make([]byte, base64.URLEncoding.EncodedLen(len(ciphertext)))
	base64.URLEncoding.Encode(b64ciphertext, ciphertext)
	b64nonce = make([]byte, base64.URLEncoding.EncodedLen(len(nonce)))
	base64.URLEncoding.Encode(b64nonce, nonce)

	//Compose a <b64URLmetadata>.<b64URLciphertext>.<b64URLnonce> literal
	buf.Write(b64metadata)
	buf.Write([]byte("."))
	buf.Write(b64ciphertext)
	buf.Write([]byte("."))
	buf.Write(b64nonce)

	//Return the AEAD literal
	return string(buf.Bytes()), nil
}

/*
Decrypt decrypts a literal of the form <b64URLmetadata>.<b64URLciphertext>.<b64URLnonce> given an AEAD cipher and
produces a metadata and data string.
*/
func Decrypt(aeadCipher cipher.AEAD, literal string) (string, string, error) {
	var (
		literalSubStrings []string
		metadata          []byte
		ciphertext        []byte
		nonce             []byte
		data              []byte
		err               error
	)

	//Split the literal into its base64 encoded metadata, ciphertext and nonce components
	literalSubStrings = strings.Split(literal, ".")
	if len(literalSubStrings) != 3 {
		return "", "", fmt.Errorf("Bad AEAD Literal: %v\n", literal)
	}

	//Decode the metadata, ciphertext and nonce
	metadata, err = base64.URLEncoding.DecodeString(literalSubStrings[0])
	if err != nil {
		return "", "", fmt.Errorf("Decode metadata failed: %v\n", literal)
	}
	ciphertext, err = base64.URLEncoding.DecodeString(literalSubStrings[1])
	if err != nil {
		return "", "", fmt.Errorf("Decode ciphertext failed: %v\n", literal)
	}
	nonce, err = base64.URLEncoding.DecodeString(literalSubStrings[2])
	if err != nil {
		return "", "", fmt.Errorf("Decode nonce failed: %v\n", literal)
	}

	//Open validates the integrity of the metadata using the authentication code in the ciphertext
	//and, if valid, decrypts the ciphertext
	data, err = aeadCipher.Open(data, nonce, ciphertext, metadata)
	if err != nil {
		return "", "", err
	}
	return string(metadata), string(data), nil
}
