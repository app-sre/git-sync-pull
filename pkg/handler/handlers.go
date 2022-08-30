package handler

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
)

type Handler struct {
	s3Client   *s3.Client
	bucket     string
	processing sync.Mutex
	repos      map[string]time.Time
}

// returns a new handler with aws s3 client initialized using environment variables
func NewHandler(ctx context.Context, bucket string) (*Handler, error) {
	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		return nil, err
	}
	return &Handler{
		s3Client: s3.NewFromConfig(cfg),
		bucket:   bucket,
		repos:    make(map[string]time.Time),
	}, nil
}

func (h *Handler) Sync(w http.ResponseWriter, req *http.Request) {
	// only allow one request to process at a time
	// simple approach to synchronization bc only one s3 bucket/gitlab org is being targetted
	h.processing.Lock()
	defer h.processing.Unlock()

	updated, err := h.getUpdatedObjects()
	if err != nil {
		log.Println(err)
		return
	}
	fmt.Println(updated)
}

// call to aws api to list all objects within bucket set by AWS_S3_BUCKET var
// returns list of objects that do not match in memory repo name to modified date map
func (h *Handler) getUpdatedObjects() ([]types.Object, error) {
	objects, err := h.s3Client.ListObjectsV2(context.TODO(), &s3.ListObjectsV2Input{
		Bucket: &h.bucket,
	})
	if err != nil {
		return nil, err
	}

	updated := []types.Object{}
	for _, object := range objects.Contents {
		// include objects that do not exist in in-memory cache
		// or objects that have different modified times
		_, exists := h.repos[*object.Key]
		if !exists || !object.LastModified.Equal(h.repos[*object.Key]) {
			updated = append(updated, object)
		}
	}

	var wg sync.WaitGroup
	ch := make(chan S3object)

	for _, updatedObj := range updated {
		wg.Add(1)
		go h.getS3Object(*updatedObj.Key, &wg, ch)
	}

	go func() {
		wg.Wait()
		close(ch)
	}()

	for obj := range ch {
		if obj.err != nil {
			return nil, err
		}

	}

	return updated, err
}

type S3object struct {
	Key  string
	Body io.ReadCloser
	err  error
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
	object.Key = key
	ch <- object
}
