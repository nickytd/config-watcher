package watcher

import (
	"context"
	"crypto/sha256"
	"fmt"
	"github.com/nickytd/config-watcher/metrics"
	"go.uber.org/zap"
	"io"
	"os"
	"sort"
	"strings"
	"sync"
	"time"
)

var log *zap.Logger

func RunTotalHashCalc(ctx context.Context, watchedDir string) <-chan string {

	if l := ctx.Value("logger"); l != nil {
		log = l.(*zap.Logger).Named("watcher")
	} else {
		log = zap.NewNop()
	}

	result := make(chan string, 2)
	ticker := time.NewTicker(5 * time.Second)
	result <- getTotalHash(watchedDir)

	go func() {
		for {
			select {
			case <-ticker.C:
				result <- getTotalHash(watchedDir)
			case <-ctx.Done():
				log.Debug("stopping ticker")
				ticker.Stop()
				return
			}
		}
	}()
	return result
}

func getTotalHash(watchedDir string) string {

	//contains folder file names as keys and corresponding hashes as values
	var filesMap = map[string]string{}
	// synchronizing map access
	var mapMutex = sync.RWMutex{}
	// waiting for calculation of files hashes to finish
	var wg = sync.WaitGroup{}

	dir, err := os.ReadDir(watchedDir)
	if err != nil {
		log.Error(
			"error reading watched dir",
			zap.Error(err),
		)
		return ""
	}

	for _, f := range dir {
		wg.Add(1)
		go func(watched string) {
			if s, _ := getSha256(watched); s != "" {
				mapMutex.Lock()
				filesMap[watched] = s
				mapMutex.Unlock()
			}
			wg.Done()
		}(watchedDir + "/" + f.Name())
	}

	wg.Wait()

	mapMutex.RLock()
	keys := make([]string, len(filesMap))
	for k, _ := range filesMap {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	builder := strings.Builder{}
	for _, k := range keys {
		builder.Grow(len(filesMap[k]))
		builder.WriteString(filesMap[k])
	}

	hash := fmt.Sprintf("%x", sha256.Sum256([]byte(builder.String())))
	log.Debug(
		"total hash calculated",
		zap.String("hash", hash),
	)
	for _, k := range keys {
		metrics.AddFileHash(k, filesMap[k], hash)
	}

	metrics.IncreaseCalculatedHashes()
	mapMutex.RUnlock()
	return hash
}

func getSha256(file string) (string, error) {

	log.Debug("checking", zap.String("name", file))
	stat, err := os.Stat(file)
	if err != nil {
		log.Error(
			"error reading file stats",
			zap.String("file", file),
			zap.Error(err),
		)
		return "", err
	}
	if stat.IsDir() {
		log.Debug(
			"skipping",
			zap.String("folder", file),
		)
		return "", nil
	}

	hash := sha256.New()
	f, err := os.Open(file)
	defer f.Close()
	if err != nil {
		log.Error(
			"error opening file",
			zap.String("file", file),
			zap.Error(err),
		)
		return "", err
	}
	if _, err := io.Copy(hash, f); err != nil {
		log.Error(
			"error reading file",
			zap.String("file", file),
			zap.Error(err),
		)
		return "", err
	}

	s := fmt.Sprintf("%x", hash.Sum(nil))
	log.Debug(
		"calculated hash",
		zap.String("file", file),
		zap.String("hash", s),
	)
	return s, nil
}
