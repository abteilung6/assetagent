# Console frontend — agent guide

Conventions for `console/` (React + Vite + TypeScript). Applies to hand-written code; see exceptions below.

## React components

Use **`React.FC`** for components. Type props explicitly when the component accepts props.

```tsx
import type React from "react";

// No props
export const ThemeToggle: React.FC = () => {
  return <button type="button">Toggle</button>;
};

// With props
type PageHeaderProps = {
  title: string;
};

export const PageHeader: React.FC<PageHeaderProps> = ({ title }) => {
  return <h1>{title}</h1>;
};

// Default export (e.g. route pages)
const TransactionsPage: React.FC = () => {
  return <div>Transactions</div>;
};

export default TransactionsPage;
```

### Do not use

```tsx
// ❌ function declaration for components
export function ThemeToggle() { ... }

// ❌ untyped arrow without React.FC
export const ThemeToggle = () => { ... };
```

### Exceptions

| Path | Rule |
|------|------|
| `src/components/ui/**` | shadcn-generated — keep CLI output; do not refactor to `React.FC` (regenerated on `shadcn add`) |
| `src/main.tsx` | Entry mount — not a component |
| `src/hooks/**` | Custom hooks — `function useX()` or `export function useX()` |
| `src/lib/**` | Non-React utilities |

After `shadcn add`, move files from `console/@/` into `src/` if the CLI writes to the wrong folder.

## UI and theming

- shadcn/ui **base-nova** defaults only — no custom colours or one-off components
- Light and dark modes via `next-themes` + `ThemeProvider`
- Phase 4 focuses on **transactions**; `/chat` starts in Phase 5

## Testing (from commit 18 onward)

- Vitest + Testing Library
- Use **`testRender`** from `@/test/render` for all component and page tests — it wraps `ThemeProvider` + `QueryClientProvider` (test client: no retries, `gcTime: 0`)
  - Routed pages: `testRender({ route: "/transactions" })`
  - Isolated components: `testRender(<MyComponent />)`
- Mock generated SDK functions via `mockApiResponse()` in `@/test/mocks`, not `fetch` directly
- `afterEach(cleanup)` runs globally in `src/test/setup.ts`

## Local dev

`console-dev` proxies `/api` to `http://localhost:8080`. Start the stack before the UI:

```bash
make dev-up serve    # postgres + API (separate terminal)
make console-dev     # Vite on :5173
```

Or run both API and console together: `make dev`

Without the API running, `/api/health` returns 500 from the Vite proxy and the header shows **API offline**.
