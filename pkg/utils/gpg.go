package utils

import (
	"bytes"
	"io"
	"log"
	"os"
	"sync"

	"golang.org/x/crypto/openpgp"
)

const (
	PRIVATE_GPG_PATH       = "PRIVATE_GPG_PATH"
	PRIVATE_GPG_PASSPHRASE = "PRIVATE_GPG_PASSPHRASE"
)

type GpgHelper struct {
	Entity openpgp.EntityList
}

// NewGpgHelper initializes a GpgHelper object and configures the private key
func NewGpgHelper() (GpgHelper, error) {
	path, exists := os.LookupEnv(PRIVATE_GPG_PATH)
	if !exists {
		log.Fatalf("Missing environment variable: %s", PRIVATE_GPG_PATH)
	}
	passphrase, exists := os.LookupEnv(PRIVATE_GPG_PASSPHRASE)
	if !exists {
		log.Fatalf("Missing environment variable: %s", PRIVATE_GPG_PASSPHRASE)
	}

	helper := GpgHelper{}

	// open private key file
	buffer, err := os.Open(path)
	if err != nil {
		return helper, err
	}
	defer buffer.Close()

	// retrieve entity from private key
	entityList, err := openpgp.ReadKeyRing(buffer)
	if err != nil {
		return helper, err
	}
	entity := entityList[0]

	// read private key using passphrase
	passphraseBytes := []byte(passphrase)
	entity.PrivateKey.Decrypt(passphraseBytes)
	for _, subkey := range entity.Subkeys {
		subkey.PrivateKey.Decrypt(passphraseBytes)
	}

	helper.Entity = entityList
	return helper, nil
}

type EncryptedObject interface {
	Key() string
	Reader() io.ReadCloser
}

type DecryptedObject struct {
	Key     string
	Archive string
	err     error
}

// accepts list of encrypted interface objects and concurrently descrypts the objects
func (g *GpgHelper) DecryptBundles(objects []EncryptedObject) ([]DecryptedObject, error) {
	result := []DecryptedObject{}

	var wg sync.WaitGroup
	ch := make(chan DecryptedObject)

	for _, obj := range objects {
		wg.Add(1)
		go g.decrypt(&wg, ch, obj)
	}

	go func() {
		wg.Wait()
		close(ch)
	}()

	for dec := range ch {
		if dec.err != nil {
			return nil, dec.err
		}
	}

	return result, nil
}

// goroutine func that sends a DecryptedObject on the channel
// accepts an s3object and decrypts the content using gpg private key
func (g *GpgHelper) decrypt(wg *sync.WaitGroup, ch chan<- DecryptedObject, object EncryptedObject) {
	defer wg.Done()
	dec := DecryptedObject{Key: object.Key()}

	// read s3 object contents
	objBytes, err := io.ReadAll(object.Reader())
	if err != nil {
		dec.err = err
		ch <- dec
		return
	}

	// decrypt body(repo archive) using private key
	details, err := openpgp.ReadMessage(bytes.NewBuffer(objBytes), g.Entity, nil, nil)
	if err != nil {
		dec.err = err
		ch <- dec
		return
	}

	// read content of decrypted details body
	decBytes, err := io.ReadAll(details.UnverifiedBody)
	if err != nil {
		dec.err = err
		ch <- dec
		return
	}
	dec.Archive = string(decBytes)

	ch <- dec
}
