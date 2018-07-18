// Copyright © 2017 NAME HERE <EMAIL ADDRESS>
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package encryption

import (
	"github.com/tmc/keyring"
	"fmt"
	"os"
	"github.com/thalesignite/crypto11"
	"log"
	"crypto"
	"crypto/rsa"
	"crypto/rand"
	"crypto/aes"
	"io"
	"bytes"
	"crypto/cipher"
)

const openscLibPath = "/usr/local/lib/opensc-pkcs11.so"

func SetupCrypto() {
	var err error
	var password string
	if password, err = keyring.Get("ykpiv-ssh-agent-helper", "PIN"); err != nil {
		fmt.Printf("ERROR - could not extract yubikey PIN from keychain; error was: %x\n", err)
		os.Exit(1)
	}

	if _, err := os.Stat(openscLibPath); err != nil {
		log.Printf("OpenSC not found at %s!\n", openscLibPath)
		log.Printf("To fix, install OpenSC from https://github.com/OpenSC/OpenSC/releases (or install https://github.com/sandstorm/ykpiv-ssh-agent-helper/releases)\n")
		log.Fatalf("ABORTING!\n")
	}

	config := &crypto11.PKCS11Config{
		Path:        openscLibPath,
		TokenSerial: "00000000",
		Pin:         password,
	}
	crypto11.Configure(config)

}

func EncryptAesKeyViaYubikey(aesKey []byte) []byte {
	var yubikeyKeyPair crypto.PrivateKey
	var err error
	if yubikeyKeyPair, err = crypto11.FindKeyPairOnSlot(0, nil, nil); err != nil {
		log.Fatalf("ERROR: could not find yubikeyKeyPair pair on slot, error was: %s\n", err)
	}

	decrypter := yubikeyKeyPair.(crypto.Decrypter)
	rsaPubKey, ok := decrypter.Public().(*rsa.PublicKey)

	if !ok {
		log.Fatalf("FATAL: Yubikey KeyPair is NOT an RSA Key; this is the only one we support currently.\n")
	}
	encryptedAesKey, err := rsa.EncryptPKCS1v15(rand.Reader, rsaPubKey, aesKey)
	if err != nil {
		log.Fatalf("ERROR while encrypting the AES Key: %s", err)
	}

	log.Println("Trying to decrypt the AES Key; to ensure we can decrypt the data using the Yubikey again.")
	log.Println("Please tap your yubikey now TWICE if it blinks.")

	decrypterOpts := &rsa.PKCS1v15DecryptOptions{
	}

	plaintext, err := decrypter.Decrypt(rand.Reader, encryptedAesKey, decrypterOpts)
	if err != nil {
		log.Fatalf("ERROR while decrypting: %s\n", err)
	}

	if !bytes.Equal(aesKey, plaintext) {
		log.Fatalf("FATAL: AES Key was not decryptable again.")
	}
	return encryptedAesKey
}

func DecryptAesKeyViaYubikey(encryptedAesKey []byte) []byte {
	var yubikeyKeyPair crypto.PrivateKey
	var err error
	if yubikeyKeyPair, err = crypto11.FindKeyPairOnSlot(0, nil, nil); err != nil {
		log.Fatalf("ERROR: could not find yubikeyKeyPair pair on slot, error was: %s\n", err)
	}

	decrypter := yubikeyKeyPair.(crypto.Decrypter)
	decrypterOpts := &rsa.PKCS1v15DecryptOptions{
	}

	aesKey, err := decrypter.Decrypt(rand.Reader, encryptedAesKey, decrypterOpts)
	if err != nil {
		log.Fatalf("ERROR while decrypting: %s\n", err)
	}

	return aesKey
}

func GenerateRandomAesKey() []byte {
	key := make([]byte, 32)

	_, err := rand.Read(key)
	if err != nil {
		log.Fatalf("FATAL: there was an error generating a random AES Key: %v\n", err)
	}
	return key
}

func pad(src []byte) []byte {
	padding := aes.BlockSize - len(src)%aes.BlockSize
	padtext := bytes.Repeat([]byte{byte(padding)}, padding)
	return append(src, padtext...)
}

func unpad(src []byte) ([]byte, error) {
	length := len(src)
	unpadding := int(src[length-1])

	if unpadding > length {
		return nil, fmt.Errorf("unpad error. This could happen when incorrect encryption key is used")
	}

	return src[:(length - unpadding)], nil
}

func EncryptAes(key []byte, text []byte) []byte {
	block, err := aes.NewCipher(key)
	if err != nil {
		log.Fatalf("ERROR encrypting AES: %s\n", err)
	}

	msg := pad(text)
	ciphertext := make([]byte, aes.BlockSize+len(msg))
	iv := ciphertext[:aes.BlockSize]
	if _, err := io.ReadFull(rand.Reader, iv); err != nil {
		log.Fatalf("ERROR encrypting AES: %s\n", err)
	}

	cfb := cipher.NewCFBEncrypter(block, iv)
	cfb.XORKeyStream(ciphertext[aes.BlockSize:], []byte(msg))
	return ciphertext
}

func DecryptAes(key []byte, decodedMsg []byte) []byte {
	block, err := aes.NewCipher(key)
	if err != nil {
		log.Fatalf("ERROR decrypting AES: %s\n", err)
	}

	if (len(decodedMsg) % aes.BlockSize) != 0 {
		log.Fatalf("ERROR decrypting AES: blocksize must be multipe of decoded message length\n")
	}

	iv := decodedMsg[:aes.BlockSize]
	msg := decodedMsg[aes.BlockSize:]

	cfb := cipher.NewCFBDecrypter(block, iv)
	cfb.XORKeyStream(msg, msg)

	unpadMsg, err := unpad(msg)
	if err != nil {
		log.Fatalf("ERROR decrypting AES: %s\n", err)
	}

	return unpadMsg
}