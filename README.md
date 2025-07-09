# Go Library

A comprehensive Go library providing essential services for microservices architecture, including logging, Redis, metrics, and configuration management. Built with [Uber FX](https://github.com/uber-go/fx) for dependency injection.

## 🚀 Features

- **Configuration Management**: Unified configuration loading from config maps and vault
- **Structured Logging**: Request-based logging with structured events
- **Metrics Collection**: Prometheus-based metrics with timers and counters
- **Redis Client**: Full-featured Redis client with cluster support
- **Dependency Injection**: Built-in Uber FX integration
- **Mock Support**: Complete mock implementations for testing

## 📦 Installation

```bash
go get github.com/smallhouse123/go-library
```

## 🏗️ Architecture

This library follows a service-oriented architecture with dependency injection:

```
go-library/
├── service/
│   ├── config/          # Configuration management
│   ├── log/             # Structured logging
│   ├── metrics/         # Prometheus metrics
│   └── redis/           # Redis client
└── go.mod
```

## 🛠️ Services

### Configuration Service

Manages application configuration from various sources.

```go
type Config interface {
    // Get key value from either configMap or vault
    Get(key string) (interface{}, error)
}
```

**Usage:**
```go
import (
    "github.com/smallhouse123/go-library/service/config"
    "go.uber.org/fx"
)

func main() {
    fx.New(
        config.Service,
        fx.Invoke(func(cfg config.Config) {
            value, err := cfg.Get("DATABASE_URL")
            if err != nil {
                log.Fatal(err)
            }
            fmt.Println("Database URL:", value)
        }),
    ).Run()
}
```

### Log Service

Structured logging with request events and user tracking.

```go
type Log interface {
    // Write log to destination file
    WriteLog(logName string, requestEvent *RequestEvent)
    // Close logger instance
    Close()
}
```

**Usage:**
```go
import (
    "github.com/smallhouse123/go-library/service/log"
    "go.uber.org/fx"
)

func main() {
    fx.New(
        log.Service,
        fx.Invoke(func(logger log.Log) {
            event := &log.RequestEvent{
                RequestCommon: &log.RequestCommon{
                    MicroTimestamp: float64(time.Now().UnixNano()) / 1e6,
                    VisitorId:      "visitor123",
                    UserName:       "john_doe",
                },
                UserEvents: []*log.UserEvent{
                    {
                        EventType: "page_view",
                        Metadata:  map[string]interface{}{"page": "/home"},
                        Count:     1,
                    },
                },
            }
            
            logger.WriteLog("user_activity", event)
        }),
    ).Run()
}
```

### Metrics Service

Prometheus-based metrics collection with timers and counters.

```go
type Metrics interface {
    // BumpTime wrap prometheus histogram for measuring func time
    BumpTime(key string, tags ...string) (Endable, error)
    // BumpCount wrap prometheus counter for key counting
    BumpCount(key string, val float64, tags ...string) error
}
```

**Usage:**
```go
import (
    "github.com/smallhouse123/go-library/service/metrics"
    "go.uber.org/fx"
)

func main() {
    fx.New(
        metrics.Service,
        fx.Invoke(func(m metrics.Metrics) {
            // Measure execution time
            timer, err := m.BumpTime("api_request_duration", "method", "GET", "endpoint", "/users")
            if err != nil {
                log.Fatal(err)
            }
            
            // Simulate work
            time.Sleep(100 * time.Millisecond)
            timer.End()
            
            // Increment counter
            m.BumpCount("api_requests_total", 1, "method", "GET", "status", "200")
        }),
    ).Run()
}
```

### Redis Service

Full-featured Redis client with cluster support and compression. The Redis service is provided through the `redismaincluster` package for production use.

```go
type Redis interface {
    Set(ctx context.Context, key string, val []byte, ttl time.Duration, zip bool) error
    Get(ctx context.Context, key string, zip bool) (val []byte, err error)
    Del(ctx context.Context, keys ...string) (int, error)
    Incr(ctx context.Context, key string) (int64, error)
    Exists(ctx context.Context, key string) (int64, error)
    TTL(ctx context.Context, key string) (int, error)
    // ... and more
}
```

**Usage:**
```go
import (
    "context"
    "github.com/smallhouse123/go-library/service/redis"
    "github.com/smallhouse123/go-library/service/redis/redismaincluster"
    "go.uber.org/fx"
)

func main() {
    fx.New(
        redismaincluster.Service,
        fx.Invoke(func(rdb redis.Redis) {
            ctx := context.Background()
            
            // Set a value with TTL
            err := rdb.Set(ctx, "user:123", []byte("john_doe"), time.Hour, false)
            if err != nil {
                log.Fatal(err)
            }
            
            // Get the value
            val, err := rdb.Get(ctx, "user:123", false)
            if err != nil {
                log.Fatal(err)
            }
            
            fmt.Println("User:", string(val))
        }),
    ).Run()
}
```

## 🔧 Complete Application Example

```go
package main

import (
    "context"
    "log"
    "time"

    "github.com/smallhouse123/go-library/service/config"
    "github.com/smallhouse123/go-library/service/log"
    "github.com/smallhouse123/go-library/service/metrics"
    "github.com/smallhouse123/go-library/service/redis"
    "github.com/smallhouse123/go-library/service/redis/redismaincluster"
    "go.uber.org/fx"
)

type App struct {
    config  config.Config
    logger  log.Log
    metrics metrics.Metrics
    redis   redis.Redis
}

func NewApp(cfg config.Config, logger log.Log, m metrics.Metrics, r redis.Redis) *App {
    return &App{
        config:  cfg,
        logger:  logger,
        metrics: m,
        redis:   r,
    }
}

func (a *App) Run() {
    // Example of using all services together
    ctx := context.Background()
    
    // Get configuration
    dbURL, err := a.config.Get("DATABASE_URL")
    if err != nil {
        log.Printf("Config error: %v", err)
    }
    
    // Measure operation time
    timer, _ := a.metrics.BumpTime("db_operation", "type", "query")
    defer timer.End()
    
    // Cache operation
    err = a.redis.Set(ctx, "config:db_url", []byte(dbURL.(string)), time.Hour, false)
    if err != nil {
        log.Printf("Redis error: %v", err)
    }
    
    // Log the event
    a.logger.WriteLog("app_start", &log.RequestEvent{
        RequestCommon: &log.RequestCommon{
            MicroTimestamp: float64(time.Now().UnixNano()) / 1e6,
            VisitorId:      "system",
        },
    })
    
    // Increment counter
    a.metrics.BumpCount("app_starts", 1, "version", "1.0.0")
}

func main() {
    fx.New(
        // Register all services
        config.Service,
        log.Service,
        metrics.Service,
        redismaincluster.Service,
        
        // Application parameters
        fx.Provide(
            func() string { return "production" },
            fx.Annotate(func() string { return "production" }, fx.ResultTags(`name:"environment"`)),
            fx.Annotate(func() string { return "/config" }, fx.ResultTags(`name:"configMapPath"`)),
            fx.Annotate(func() string { return "/vault" }, fx.ResultTags(`name:"vaultPath"`)),
            fx.Annotate(func() string { return "myapp" }, fx.ResultTags(`name:"serviceName"`)),
        ),
        
        // Register app
        fx.Provide(NewApp),
        
        // Run the application
        fx.Invoke(func(app *App) {
            app.Run()
        }),
    ).Run()
}
```

## 🧪 Testing with Mocks

Each service provides mock implementations for testing:

```go
import (
    "testing"
    "github.com/smallhouse123/go-library/service/config/mocks"
    "github.com/stretchr/testify/mock"
)

func TestConfigService(t *testing.T) {
    // Create mock
    mockConfig := mocks.NewConfig(t)
    
    // Set expectations
    mockConfig.On("Get", "API_KEY").Return("test-key", nil)
    
    // Use in your code
    apiKey, err := mockConfig.Get("API_KEY")
    
    // Assertions
    assert.NoError(t, err)
    assert.Equal(t, "test-key", apiKey)
    
    // Verify all expectations were met
    mockConfig.AssertExpectations(t)
}
```

## 📋 Requirements

- Go 1.21 or higher
- Required dependencies are managed via `go.mod`

## 🔑 Key Dependencies

- **Uber FX**: Dependency injection framework
- **Prometheus**: Metrics collection
- **Redis**: Go Redis client with cluster support
- **Zap**: Structured logging
- **Testify**: Testing framework with mocks

## 📝 License

This project is licensed under the MIT License.

## 🤝 Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Add tests
5. Submit a pull request

## 📖 Documentation

For more detailed documentation on each service, refer to the individual service directories:

- [Config Service](./service/config/)
- [Log Service](./service/log/)
- [Metrics Service](./service/metrics/)
- [Redis Service](./service/redis/)

## 🚀 Getting Started

1. **Clone the repository**
2. **Install dependencies**: `go mod tidy`
3. **Run tests**: `go test ./...`
4. **Generate mocks**: `mockery --all` (if needed)

---

Built with ❤️ using Go and Uber FX 