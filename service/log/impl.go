package log

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/smallhouse123/go-library/service/config"
	"go.uber.org/fx"
)

var (
	Service = fx.Provide(New)
)

type Impl struct {
	logDir         string
	currentHour    string
	currentFile    *os.File
	logBuffer      []string
	flushThreshold int
	flushPeriod    int
	mu             sync.Mutex
	flushChan      chan struct{}
	done           chan struct{}
	wg             sync.WaitGroup

	configService config.Config
}

var ROOT_DIR = os.Getenv("APP_ROOT")

const (
	DEFAULT_FLUSH_THRESHOLD = 1000
	DEFAULT_FLUSH_PERIOD    = 5 // minutes
)

func New(configService config.Config) Log {
	podName := os.Getenv("K8S_POD_NAME")

	var subDir string
	if podName != "" {
		subDir = filepath.Join("logs", podName)
	}

	fullDir := filepath.Join(ROOT_DIR, subDir)
	if fullDir == "" {
		fmt.Println("ROOT_DIR is not set. Logs will not be stored.")
		return nil
	}

	if err := os.MkdirAll(fullDir, os.ModePerm); err != nil {
		fmt.Printf("Failed to create log directory: %v\n", err)
		return nil
	}

	flushThreshold := getConfigInt(configService, "LOG_FLUSH_THRESHOLD", DEFAULT_FLUSH_THRESHOLD)
	flushPeriod := getConfigInt(configService, "LOG_FLUSH_PERIOD", DEFAULT_FLUSH_PERIOD)

	im := &Impl{
		logDir:         fullDir,
		logBuffer:      make([]string, 0, flushThreshold),
		flushChan:      make(chan struct{}, 1),
		done:           make(chan struct{}),
		configService:  configService,
		flushThreshold: flushThreshold,
		flushPeriod:    flushPeriod,
	}

	go im.flushLoop()

	return im
}

func getConfigInt(configService config.Config, key string, defaultValue int) int {
	val, err := configService.Get(key)
	if err != nil {
		return defaultValue
	}
	if valInt, ok := val.(int); ok {
		return valInt
	}
	return defaultValue
}

func (im *Impl) WriteLog(logName string, requestEvent *RequestEvent) {
	currentHour := time.Now().Format("06_01_02__15")

	im.mu.Lock()
	defer im.mu.Unlock()

	if im.currentHour != currentHour {
		im.flush()
		if im.currentFile != nil {
			im.currentFile.Close()
		}

		logFilePath := filepath.Join(im.logDir, currentHour+".log")
		file, err := os.OpenFile(logFilePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			fmt.Printf("Failed to open log file: %v\n", err)
			im.currentFile = nil
			return
		}

		im.currentFile = file
		im.currentHour = currentHour
	}

	eventJSON, err := json.Marshal(&requestEvent)
	if err != nil {
		fmt.Printf("Failed to encode event to JSON: %v\n", err)
		return
	}

	im.logBuffer = append(im.logBuffer, string(eventJSON))

	if len(im.logBuffer) >= im.flushThreshold {
		select {
		case im.flushChan <- struct{}{}:
		default: // Avoid blocking if the channel is full
		}
	}
}

func (im *Impl) flush() {
	if len(im.logBuffer) == 0 || im.currentFile == nil {
		return
	}

	for _, logEntry := range im.logBuffer {
		if _, err := im.currentFile.WriteString(logEntry + "\n"); err != nil {
			fmt.Printf("Failed to write log entry: %v\n", err)
		}
	}

	im.logBuffer = im.logBuffer[:0]
}

func (im *Impl) flushLoop() {
	im.wg.Add(1)       // Increment the WaitGroup counter
	defer im.wg.Done() // Decrement when the flush loop exits

	ticker := time.NewTicker(time.Duration(im.flushPeriod) * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-im.flushChan:
			im.mu.Lock()
			im.flush()
			im.mu.Unlock()
		case <-ticker.C:
			im.mu.Lock()
			im.flush()
			im.mu.Unlock()
		case <-im.done:
			im.mu.Lock()
			im.flush()
			if im.currentFile != nil {
				im.currentFile.Close()
			}
			im.mu.Unlock()
			return
		}
	}
}

func (im *Impl) Close() {
	im.done <- struct{}{}
	im.wg.Wait()
}
