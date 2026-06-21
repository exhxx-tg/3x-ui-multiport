# Frontend Structure (frontend/)

## Build System
- **Framework:** React 18+ with TypeScript
- **Bundler:** Vite 5+
- **Testing:** Vitest
- **Linting:** ESLint (flat config)

## Directory Layout
```
frontend/
├── src/
│   ├── components/     # Reusable React components
│   ├── pages/          # Route pages
│   ├── api/            # API client (Axios)
│   ├── hooks/          # Custom React hooks
│   ├── store/          # State management
│   ├── utils/          # Utility functions
│   ├── styles/         # CSS/Tailwind styles
│   ├── types/          # TypeScript type definitions
│   ├── locales/        # i18n translation files
│   ├── App.tsx         # Main app component
│   └── main.tsx        # Entry point
├── public/             # Static assets
├── index.html          # HTML template
├── login.html          # Login page (separate)
├── subpage.html        # Subscription page (separate)
├── vite.config.js      # Vite configuration
├── tsconfig.json       # TypeScript config
├── package.json        # Dependencies
└── vitest.config.ts    # Test configuration
```

## Build Output
- Built to `internal/web/dist/` (embedded via Go embed)
- Served as static assets from Go binary
- Index.html served for all panel routes (SPA routing)

## Key Pages/Screens
- **Dashboard** - System overview, traffic graphs, online users
- **Inbounds** - List, create, edit, delete proxy inbounds
- **Inbound Detail** - Client management, config editing
- **Settings** - Panel configuration, TLS, Telegram bot
- **Admin** - User management

## API Integration
- Axios HTTP client
- Response interceptor for auth errors
- Real-time updates via WebSocket

## Current Limitations
- No protocol-specific management UI
- No standalone service management
- Limited monitoring/visualization
- No RBAC UI
- Single language (no full i18n)
- No dark/light theme toggle
