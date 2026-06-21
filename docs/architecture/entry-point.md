# Entry Point Analysis (main.go)

## Flow
1. Load environment file (`loadServiceEnvFile`)
2. Parse command-line arguments (run, setting, migrate, cert, migrate-db)
3. Initialize components:
   - Configuration loading
   - Logger initialization
   - Database connection (SQLite/PostgreSQL)
   - Web server setup (Gin)
   - Xray core management
   - Subscriptions server
4. Start web server on configured port
5. Wait for signals (SIGHUP, SIGTERM, SIGUSR1, Interrupt)

## Key Functions
- `runWebServer()` - Main server startup
- `main()` - Entry point with CLI subcommands
- `loadServiceEnvFile()` - Load systemd environment file

## Signal Handling
- `SIGHUP` - Restart web server + sub server
- `SIGUSR1` - Restart Xray core
- `SIGTERM/Interrupt` - Graceful shutdown

## Initialization Order
1. Logger config (log level)
2. Database init (migrations)
3. Web server (Gin router + handlers)
4. Subscription server
5. Signal handler loop
