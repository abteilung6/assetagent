# Iteration 2 — Milestone Plan

**Stand:** 2026-08-05  
**Horizon:** ~12 weeks to Paid Beta + Retention gate  
**Status:** Phase C eng active — [phase-c-outcome-mvp.md](./phase-c-outcome-mvp.md); B eng ✅ (product gate may trail); A discovery open  
**Product goal:** Turn the working Sparkasse demo into a buyable Monthly Money Review product

---

## Document map

| Document | Role |
|----------|------|
| [Product vision](./assetagent_product_vision_2026-08-01.md) | Positioning, ICP, pricing, GTM, kill criteria |
| [Development roadmap](./assetagent_development_roadmap_2026-08-01.md) | Engineering principles, schema sketch, 30-issue backlog |
| **This file** | Authoritative milestone checklist + phase order for execution |
| [Phase A — Import Alpha](./phase-a-import-alpha.md) | Import commit plan (eng ✅; product discovery open) |
| [Phase B — Trusted Money Model](./phase-b-trusted-money-model.md) | Trusted money eng (B0–B13 ✅; product gate open) |
| [Phase C — Outcome MVP](./phase-c-outcome-mvp.md) | Active detail doc: baseline → review → forecast → scenarios → actions |
| [AI-native chat](./phase-ai-native-chat.md) | Parallel: chat starters, context, tools (A0–A5 ✅; A6 deferred) |
| [Baseline IA](./phase-baseline-ia.md) | Parallel: Typical month / Months nav + drivers layer |
| Later | `phase-d-paid-beta.md`, … (one implementation doc per active phase) |

**Rule:** Prefer one active implementation doc. Product discovery for prior phases may trail; trust/import blockers found in discovery interrupt C.

---

## Product north star

**Completed Money Reviews with a confirmed action** per month.

Activity metrics (chat messages, tokens, import row counts) are secondary.

### Positioning (locked)

> Private Finanz-CFO for German households: local vault, deterministic money facts, monthly review + one measurable action — not “ChatGPT for bank transactions.”

### Non-goals until Retention gate

PSD2/Open Banking · Assets/net worth · Tax · Native mobile · Multi-user SaaS · Advisor portal · Arbitrary SQL agent · Full ChatGPT-style session product · Extra LLM providers · Dashboard builder

---

## Principles (carry into every phase doc)

1. **Finance logic before chat polish.** Clean import → Trusted Money Model → Baseline → Review → Decision → Month-2 return.
2. **LLM orchestrates; domain calculates.** No numeric LLM math. Typed tool schemas with evidence.
3. **Uncertainty is visible.** Confidence, source, correction path — not fake precision.
4. **Phase ends on product gate**, not ticket count.
5. **Vertical slices, green builds, no dead code** — same commit rules as `tmp/implementation.md`.
6. **Database-first** for schema: migration → sqlc → repository → service → API/UI in one commit when possible.

---

## Milestone overview

```text
Phase 0  Baseline harden          (short)
    ↓
Phase A  Import Alpha             ← eng ✅; discovery may trail
    ↓
Phase B  Trusted Money Model      ← eng ✅; product gate may trail
    ↓
Phase C  Outcome MVP              ← ACTIVE eng
    ↓
Phase D  Paid Beta packaging
    ↓
Phase E  Retention proof          (month-2 loop)
    ↓
Decision Continue B2C / Iterate / Advisor pivot / Stop commercial
```

| ID | Milestone | Approx | Outcome | Exit gate | Detail doc | Status |
|----|-----------|--------|---------|-----------|------------|--------|
| **0** | Baseline | ~3–5 days | Happy path measurable; discovery started | E2E import→cashflow→evidence in CI; ≥3 Sparkasse fixtures; ≥5 testers booked | *(inline below)* | 🟨 eng CI ✅ / discovery open |
| **A** | Import Alpha | ~2 weeks | Non-technical user imports without CLI | ≥90% unaided import; ≥98% valid rows; dedupe + rollback | [phase-a-import-alpha.md](./phase-a-import-alpha.md) | 🟨 eng A0–A10 ✅ / discovery open |
| **B** | Trusted Money Model | ~3 weeks | Household economics, not raw bookings | Golden datasets cent-exact; transfers net 0; ≥90% baseline numbers trusted | [phase-b-trusted-money-model.md](./phase-b-trusted-money-model.md) | 🟨 eng B0–B13 ✅ / product gate open |
| **C** | Outcome MVP | ~3 weeks | Persisted Monthly Money Review + decision | 10/20 confirm insight; ≥5 choose action; no ungrounded numbers | [phase-c-outcome-mvp.md](./phase-c-outcome-mvp.md) | ⬜ eng C0–C12 |
| **D** | Paid Beta | ~2 weeks | Self-host / web access + Founding Pro purchasable | 10 strangers use web app; 10 pay; &lt;10% material number errors | *TBD after C* | ⬜ |
| **E** | Retention | ~2 weeks | Second monthly review proves loop | ≥40% return; ≥30% confirmed actions; support &lt;30 min/user/mo | *TBD after D* | ⬜ |

---

## Phase 0 — Baseline (inline, no separate doc)

**Goal:** Make today’s demo reproducible and quality-measurable before Import Alpha.

### Engineering

- [x] End-to-end test: CLI/API import → cashflow tool question → evidence link (`TestIntegration_ImportCashflowEvidence`)
- [x] ≥3 anonymized/synthetic Sparkasse fixtures (happy, duplicates, bad rows) — `minimal`, `sample`, `overlap`, `headers_only` already in `testdata/sparkasse/`
- [ ] Confirm money math stays on `decimal` (no `float64` in finance domain)
- [ ] Structured error codes sketch for import/tool failures
- [ ] Telemetry audit: no full booking text / IBAN in Langfuse at default detail

### Product discovery (parallel, not blocked by eng)

- [ ] Landing hypothesis + Founding Pro offer (59 € / first year)
- [ ] Interview script focused on decision problems
- [ ] Book ≥5 concierge reviews with ICP

### Exit

Phase 0 is done when the three engineering bullets above are green **and** five suitable testers are scheduled. Then open Phase A execution.

---

## Phase summaries (scope only — detail lives in phase docs)

### Phase A — Import Alpha

In-app upload, preview before commit, `ImportRun`, accounts naming, idempotent commit, undo, fixtures + E2E.

**Out of scope:** categories, transfers, forecast, payment, new chat features beyond CTA to “prepare review.”

### Phase B — Trusted Money Model

Accounts/transfers, merchants, categories + correction queue, recurring series, golden money suite, tools v2 (`get_cashflow_v2`, recurring, changes, anomalies) with evidence contract.

**Delivery:** responsive **web UI** (desktop or mobile browser). CLI is **dev-only**. No native installer in B.

**Out of scope:** persisted Review UI (that’s C), desktop/native packaging, checkout.

### Phase C — Outcome MVP

`FinancialBaseline` (5 confirmable numbers), persisted `MoneyReview` + findings, 90-day forecast, 3 typed scenarios, Decision/Action ledger.

**Out of scope:** Open Banking, native installers, payment.

### Phase D — Paid Beta packaging

Self-host / web access hardening, backup/restore/delete, Privacy Local vs Smart Cloud UX, Founding checkout/license, review-level feedback, support runbook.

**Note:** Product remains a **web app** unless we explicitly revive a native shell. “Install” in older vision copy = get API + open console in a browser, not a mandatory desktop installer.

### Phase E — Retention proof

Follow-up import + review diff, action verification, light reminder, support load check — then **one** strategic decision (Continue / Iterate / Advisor / Stop).

---

## Cross-cutting backlog anchors

From the development roadmap (issues 1–30). Phase docs own the commit breakdown; this table owns priority order.

| Issues | Milestone | Theme |
|--------|-----------|--------|
| 1 | 0 | E2E happy-path lock |
| 2–8 | A | ImportRun → upload UI → accounts onboarding |
| 9–16 | B | Transfers → classify → recurrence → golden → tools v2 |
| 17–23 | C | Baseline → Review → forecast → scenarios → Decision/Action |
| 24–28 | D | Feedback, web/self-host packaging, backup, privacy modes, checkout |
| 29–30 | E | Follow-up import diff + action verification |

---

## Architecture direction (stable)

Keep monolith layers. Evolve packages toward:

```text
internal/importing      Preview, validate, commit, rollback
internal/transactions   Queries + evidence
internal/classify       Merchants, categories, transfers, recurrence   (Phase B+)
internal/finance        Baseline + deterministic calcs                 (Phase C+)
internal/forecast       Projection + scenarios                         (Phase C+)
internal/review         Review generation / versioning                 (Phase C+)
internal/decisions      Actions + outcome verification                 (Phase C+)
internal/agenttools     Thin typed adapters over domain
internal/evals          Golden datasets + regression runner
```

Exact names may match existing `internal/service`, `internal/parser`, `internal/chat/tools` during Phase A; introduce new packages when boundaries get painful — not as a rewrite.

OpenAPI remains the contract. Console uses generated clients only.

---

## Go / No-Go after Phase E

| Signal | Move |
|--------|------|
| ≥10 payers, ≥40% return, support manageable | Continue B2C → harden Sparkasse → second bank → licensed aggregator later |
| Numbers trusted, retention weak | Iterate ICP / monthly occasion / action UX — no platform features |
| Households love it, won’t pay; coaches would | Advisor concierge pilots — not multi-tenant SaaS yet |
| &lt;5/30 qualified pay with hand-holding | Stop commercial investment; keep as portfolio/OSS |

---

## Working agreement

- **Active phase (eng):** C — [phase-c-outcome-mvp.md](./phase-c-outcome-mvp.md) (C0–C11 ✅; C12 deferred)  
- **Parallel tracks (optional):**  
  - AI-native chat — [phase-ai-native-chat.md](./phase-ai-native-chat.md) (A0–A5 ✅; A6 deferred)  
  - Baseline IA — [phase-baseline-ia.md](./phase-baseline-ia.md) (Typical month / Months / drivers; **not** Phase D)  
- **Phase B:** eng ✅ (B0–B13); product gate may continue in parallel (trusted-number friction interrupts C).  
- **Phase A:** eng ✅; product discovery may continue in parallel (import friction interrupts C).  
- **Delivery default:** web console in browser (desktop or mobile); CLI = developers only; no installer required for B/C.  
- When C’s exit gate passes (or eng DoD + explicit deferral of cohort metrics): mark C ✅ here, create `phase-d-paid-beta.md`, set D as active.  
- Do not implement Phase D+ work while C is open unless it unblocks C’s gate.

---

## Changelog

| Date | Change |
|------|--------|
| 2026-08-02 | Initial milestone plan + Phase A detail doc created from vision/roadmap analysis |
| 2026-08-02 | Phase A eng A0–A10 done; Phase B detail doc created; web-only delivery + CLI=dev locked |
| 2026-08-05 | Phase B eng B0–B13 done; Phase C detail doc created; C set as active eng |
| 2026-08-08 | C11 golden baseline/forecast landed in eng; C12 deferred. AI-native chat track planned in [phase-ai-native-chat.md](./phase-ai-native-chat.md) (not Phase D). |
| 2026-08-08 | Baseline IA track planned in [phase-baseline-ia.md](./phase-baseline-ia.md): Typical month / Months nav split + drivers as explanation layer (not Phase D). |
