# di
Simple Go dependency injection framework based on modules, module imports and constructors.

## Installation
```bash
go get github.com/ivankorobkov/di
```

## Usage
```go
import (
    "log"
    "github.com/ivankorobkov/di"
)

// Define modules with your dependencies and constructors.
func DbModule(m *di.Module) {
    m.Add(func() *sql.DB { 
        db, err := sql.Open(...) 
        if err != nil {
            panic(err)
        }
        return db
    })
}
func HelloModule(m *di.Module) {
    m.Import(DbModule)
    m.Add(newHelloService)
}

// Example service with dependencies.
type Hello interface {
    HelloWorld() string
}
type helloService struct {
    db *sql.DB
}
func (h *helloService) HelloWorld() string {
    return "Hello, world"
}
func newHelloService(db *sql.Db) Hello {
    return &helloService{db: db}
}

// Build an object graph and initialize a struct.
// Dependencies are automatically injected into constructors.
func main() {
    var app struct {
        Hello Hello
    }
    di.Init(&app, HelloModule)

    msg := app.Hello.HelloWorld()
    log.Println(msg)
}
```
