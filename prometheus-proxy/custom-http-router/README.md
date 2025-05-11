# Custom HTTP Router

This project implements a custom HTTP router in Go, providing a lightweight alternative to traditional web servers like Nginx. The router supports middleware, request handling, and configuration management.

## Project Structure

```
custom-http-router
├── cmd
│   └── server
│       └── main.go          # Entry point of the application
├── pkg
│   ├── router
│   │   ├── router.go        # Custom HTTP router implementation
│   │   ├── middleware.go     # Middleware functions for routes
│   │   ├── handler.go        # Request handlers for various routes
│   │   └── router_test.go    # Unit tests for the router
│   ├── config
│   │   └── config.go         # Configuration loading and parsing
│   └── metrics
│       └── metrics.go        # Application metrics collection
├── internal
│   └── utils
│       └── utils.go          # Utility functions for the application
├── config
│   └── config.yaml           # YAML configuration file
├── go.mod                    # Module dependencies
├── go.sum                    # Module checksums
├── Makefile                  # Build instructions and commands
└── README.md                 # Project documentation
```

## Setup Instructions

1. **Clone the repository:**
   ```
   git clone <repository-url>
   cd custom-http-router
   ```

2. **Install dependencies:**
   ```
   go mod tidy
   ```

3. **Run the application:**
   ```
   go run cmd/server/main.go
   ```

## Usage

Once the server is running, you can access the API endpoints defined in the router. Refer to the documentation in `pkg/router/handler.go` for available routes and their usage.

## Testing

To run the tests for the router, use the following command:
```
go test ./pkg/router
```

## Contributing

Contributions are welcome! Please open an issue or submit a pull request for any improvements or bug fixes.

## License

This project is licensed under the MIT License. See the LICENSE file for details.