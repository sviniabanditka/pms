# AI roadmap — own, self-hosted, learns from your data

No external LLM API. Everything runs locally (embed sidecar → later a small
generative model). The through-line: one embedding foundation, then a cascade of
features, and a **feedback loop** so the models improve on the workspace's own
history — that's what makes it "our neural net", not a static model.

Decisions locked (2026-07): **fastembed sidecar** (pure ONNX, no torch),
**multilingual-e5-small** (~384-dim, RU+EN). ML boundary = the sidecar's HTTP
interface (`/embed`, later `/classify`, `/generate`) — swap runtime without
touching the rest.

---

## Phase 1 — Embedding foundation · semantic search + dup detection  ← START

- pgvector in Postgres 16; `issues.embedding vector(384)` + HNSW (cosine) index.
- fastembed sidecar (`POST /embed {texts[]}`) in docker-compose.
- River worker: on issue create/update → embed `title+description` → upsert vector;
  one-time backfill of existing issues.
- RPC `SemanticSearch(query)` (embed → cosine top-k) + `SimilarIssues(id)`.
- UI: semantic mode in ⌘K; "similar / possible duplicate" panel on create.
- **DoD:** meaning-based search works in RU+EN; dup warning on create.

## Phase 2 — kNN recommendations (same vectors, no new model)

Query the existing embeddings differently — near-zero extra infra.

- Auto-label suggestion (labels of nearest neighbours).
- Team routing + **assignee suggestion** (who resolved the nearest issues).
- "Related issues" panel on the detail view.
- Extend embeddings to **docs + comments** → search across everything.
- Workspace-wide background dedup report.

## Phase 3 — Trained models on your history (the "learns from you" part)

Train small models offline on the tenant's own data; serve via the sidecar
(ONNX). Retrain on a River cron. Features = embedding + structured (team,
reporter, labels, keywords).

- **Priority prediction** (classifier).
- **Effort / estimate prediction** (regression).
- **Cycle-time / slip-risk** ("this will likely miss its due date") → risk flags.
- **Trained triage** (label/team/assignee) — beats kNN once there's data.
- Per-workspace models (RLS already isolates tenants).

## Phase 4 — Unsupervised insights (clustering)

- **Topic clustering** (HDBSCAN over embeddings) → themes / suggested epics.
- Anomaly detection (spike in a topic = incident signal).
- **Multilingual sentiment** on comments/chat (small model) → team-health signal.
- **Extractive** cycle/thread summaries (TextRank over embeddings — no LLM yet).

## Phase 5 — Small local generative (still self-hosted)

Add a tiny instruct model (Qwen2.5-0.5/1.5B / Phi) via llama.cpp/ollama in the
sidecar. Abstractive where extractive isn't enough.

- Issue / thread / cycle summaries; auto **standup digest**.
- **NL quick-create**: "bug in auth, P1, assign me, due fri" → structured issue
  (NER for dates/priority/@mentions + small model).
- Smart replies / autocomplete in chat.

## Phase 6 — Retrieval-augmented + agentic (frontier)

- **RAG "ask your workspace"**: retrieval (Phase 1) + local generative →
  "what's blocking the payments epic?", "summarize open bugs in team X".
- **Auto-triage agent**: new issue → embed → dedup → suggest label/team/assignee/
  priority → auto-apply if confident, else suggest.
- **Learning-to-rank** search (train on click data).

---

## Cross-cutting

- **Feedback loop** (built from Phase 2): log every suggestion accept/reject →
  training signal → periodic retrain. Active learning. This is the core "it gets
  smarter" mechanism; wire the telemetry early even if retraining comes later.
- **Sidecar evolution**: embed-only → +classifier serving → +generative. Stays
  the single ML boundary; the Go backend only speaks HTTP to it.
- **Privacy**: fully local, per-tenant (RLS), nothing leaves the deployment — a
  selling point vs cloud-AI PMS tools.
- **Infra cost**: Phases 1–4 are CPU-cheap (small ONNX). Phase 5–6 want a bit
  more (small LLM on CPU is OK for async digests; GPU optional).

## Sequencing

Phase 1 now. 2 follows cheaply (same vectors). 3–4 once there's usage data to
train on. 5–6 when the generative story is worth the extra infra. Each phase
ships standalone value.
