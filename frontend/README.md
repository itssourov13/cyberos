# CYBEROS Frontend

Non-authoritative operator console and public site. TypeScript · React · Vite · React Router · lucide-react.

```bash
npm install
npm run dev        # http://localhost:5173 (proxies /api to :8080)
npm run typecheck
npm run lint
npm run build
```

Runs on a **preview adapter** (`src/api/preview.ts`) until Phase 24D wires the live `/api/v1` adapter behind the same `CyberosApi` interface. The browser never issues grants, decides authorisation, changes trust, verifies results or runs tools.

See `../docs/FRONTEND_REBUILD.md` for architecture, routes and limitations.
