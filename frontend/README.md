# CYBEROS Frontend

Thin, **non-authoritative** operator console.

## Stack

Vite · React · TypeScript · Tailwind CSS · React Router · Lucide

## Commands

```bash
npm install
npm run dev      # http://localhost:5173
npm run build
```

## Architecture

```
UI → src/api/client.ts → MOCK adapter (default) → fixtures
                      ↘ future: /api/v1 → Go services → Security spine
```

Default is **Development · Mock API**. Do not treat MOCK decisions, grants, or executions as live infrastructure control.

## Security rules

- No Grant minting in the browser  
- No Trust mutation authority  
- No Broker / Host / Executor implementation  
- No private keys or raw credentials  
- Integrity ≠ Signature ≠ Trust ≠ Authorization ≠ Execution  

## Status

Phase **24B** UI foundation — **PARTIAL** relative to production. Real API integration is **24C (DEFERRED)**.
