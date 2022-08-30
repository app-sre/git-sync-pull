package git

import (
	"bytes"
	"io"
	"log"
	"os"
	"sync"

	"github.com/app-sre/git-sync-pull/pkg/handler"
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
func NewGpgHelper() GpgHelper {
	path, exists := os.LookupEnv(PRIVATE_GPG_PATH)
	if !exists {
		log.Fatalf("Missing environment variable: %s", PRIVATE_GPG_PATH)
	}
	passphrase, exists := os.LookupEnv(PRIVATE_GPG_PASSPHRASE)
	if !exists {
		log.Fatalf("Missing environment variable: %s", PRIVATE_GPG_PASSPHRASE)
	}

	// open private key file
	buffer, err := os.Open(path)
	if err != nil {
		log.Fatal(err.Error())
	}
	defer buffer.Close()

	// retrieve entity from private key
	entityList, err := openpgp.ReadKeyRing(buffer)
	if err != nil {
		log.Fatal(err.Error())
	}
	entity := entityList[0]

	// read private key using passphrase
	passphraseBytes := []byte(passphrase)
	entity.PrivateKey.Decrypt(passphraseBytes)
	for _, subkey := range entity.Subkeys {
		subkey.PrivateKey.Decrypt(passphraseBytes)
	}

	return GpgHelper{
		Entity: entityList,
	}
}

type DecryptedObject struct {
	Key     string
	Archive string
	err     error
}

// goroutine func that sends a DecryptedObject on the channel
// accepts an s3object and decrypts the content using gpg private key
func (g *GpgHelper) DecryptBundle(wg *sync.WaitGroup, ch chan<- DecryptedObject, object handler.S3object) {
	defer wg.Done()
	dec := DecryptedObject{Key: object.Key}

	// read s3 object contents
	objBytes, err := io.ReadAll(object.Body)
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

func PushLatest() {}
