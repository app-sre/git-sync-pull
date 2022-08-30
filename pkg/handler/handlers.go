package handler

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/app-sre/git-sync-pull/pkg/utils"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
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

	gpg, err := utils.NewGpgHelper()
	if err != nil {
		log.Println(err)
		return
	}

	repoArchives, err := gpg.DecryptBundles(updated)
	fmt.Println(repoArchives)
}
