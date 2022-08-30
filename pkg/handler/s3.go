package handler

import (
	"context"
	"io"
	"sync"

	"github.com/app-sre/git-sync-pull/pkg/utils"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

type S3object struct {
	ObjKey string
	Body   io.ReadCloser
	err    error
}

func (s S3object) Key() string {
	return s.ObjKey
}

func (s S3object) Reader() io.ReadCloser {
	return s.Body
}

// call to aws api to list all objects within bucket set by AWS_S3_BUCKET var
// returns list of objects that do not match in memory repo name to modified date map
func (h *Handler) getUpdatedObjects() ([]utils.EncryptedObject, error) {
	objects, err := h.s3Client.ListObjectsV2(context.TODO(), &s3.ListObjectsV2Input{
		Bucket: &h.bucket,
	})
	if err != nil {
		return nil, err
	}

	updatedObjKeys := []*string{}
	for _, object := range objects.Contents {
		// include objects that do not exist in in-memory cache
		// or objects that have different modified times
		_, exists := h.repos[*object.Key]
		if !exists || !object.LastModified.Equal(h.repos[*object.Key]) {
			updatedObjKeys = append(updatedObjKeys, object.Key)
		}
	}

	var wg sync.WaitGroup
	ch := make(chan S3object)

	for _, key := range updatedObjKeys {
		wg.Add(1)
		go h.getS3Object(*key, &wg, ch)
	}

	go func() {
		wg.Wait()
		close(ch)
	}()

	result := []S3object{}
	for obj := range ch {
		if obj.err != nil {
			return nil, err
		}
		result = append(result, obj)
	}

	return convert(result), nil
}

// goroutine func
// aws call for details of specific object. returned via channel
func (h *Handler) getS3Object(key string, wg *sync.WaitGroup, ch chan<- S3object) {
	defer wg.Done()
	object := S3object{}
	result, err := h.s3Client.GetObject(context.TODO(), &s3.GetObjectInput{
		Bucket: &h.bucket,
		Key:    &key,
	})
	if err != nil {
		object.err = err
		ch <- object
	}
	object.Body = result.Body
	object.ObjKey = key
	ch <- object
}

func convert(originals []S3object) []utils.EncryptedObject {
	converted := []utils.EncryptedObject{}
	for _, o := range originals {
		converted = append(converted, utils.EncryptedObject(o))
	}
	return converted
}
