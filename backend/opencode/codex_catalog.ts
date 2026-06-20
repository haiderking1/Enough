// PORT: mirrors backend/opencode/codex_catalog.go

import { Effect } from "effect";
import { codex_cloudflare_headers } from "../auth/codex_headers";
import { codex_models, codex_known_models, default_reasoning_levels, type model_info } from "./providers";
import { normalize_model } from "./models";

export let codex_models_url = "https://chatgpt.com/backend-api/codex/models?client_version=1.0.0";

// codexContextFallback mirrors Hermes' verified Codex OAuth limits (Apr 2026).
export const codex_context_fallback: Record<string, number> = {
  "gpt-5.5": 272_000,
  "gpt-5.4": 272_000,
  "gpt-5.4-mini": 272_000,
  "gpt-5.3-codex": 272_000,
  "gpt-5.3-codex-spark": 128_000,
  "gpt-5-codex": 272_000,
};

export type codex_models_response = { models: codex_model_entry[] };
export type codex_model_entry = { slug: string; title: string; context_window: number; visibility: string; priority: number };

export const codex_context_fallback_for = (model_id: string): number => codex_context_fallback[model_id] ?? 272_000;

// FetchCodexModels loads the live Codex model catalog with context windows.
export const fetch_codex_models = (ctx: AbortSignal | undefined, access_token: string): Effect.Effect<model_info[], Error> =>
  Effect.tryPromise({
    try: async () => {
      access_token = access_token.trim();
      if (access_token === "") throw new Error("codex: missing access token");
      const resp = await fetch(codex_models_url, { signal: ctx, headers: { Authorization: `Bearer ${access_token}`, ...codex_cloudflare_headers(access_token) } });
      const raw = await resp.text();
      if (resp.status >= 400) throw new Error(`codex models ${resp.status}: ${raw.trim()}`);
      const payload = JSON.parse(raw) as codex_models_response;
      const sortable: { rank: number; m: model_info }[] = [];
      for (const entry of payload.models) {
        const slug = entry.slug.trim();
        if (slug === "") continue;
        const vis = entry.visibility.trim().toLowerCase();
        if (vis === "hide" || vis === "hidden") continue;
        let name = entry.title.trim();
        if (name === "") name = codex_known_models[slug]?.name ?? slug;
        let ctx_window = entry.context_window;
        if (ctx_window <= 0) ctx_window = codex_context_fallback_for(slug);
        const m = normalize_model({ id: slug, name, context_window: ctx_window, reasoning: true, thinking_levels: [...default_reasoning_levels] });
        const rank = entry.priority > 0 ? entry.priority : 10_000;
        sortable.push({ rank, m });
      }
      if (sortable.length === 0) throw new Error("codex models: empty list");
      sortable.sort((a, b) => a.rank !== b.rank ? a.rank - b.rank : a.m.name.toLowerCase().localeCompare(b.m.name.toLowerCase()));
      return sortable.map((item) => item.m);
    },
    catch: (cause) => cause instanceof Error ? cause : new Error(String(cause)),
  }).pipe(Effect.catchAll((err) => Effect.fail(err.message.startsWith("codex") ? err : err)));

export const fetch_codex_models_fallback = (ctx: AbortSignal | undefined, access_token: string): Effect.Effect<model_info[], Error> =>
  fetch_codex_models(ctx, access_token).pipe(Effect.catchAll((err) => Effect.succeed(codex_models()).pipe(Effect.zipRight(Effect.fail(err)))));

/*
PORT STATUS
source path: backend/opencode/codex_catalog.go
source lines: 134
draft lines: 70
confidence: medium
status: phase_a_draft
todos:
  - reconcile fallback-return-on-error behavior with Go's `(CodexModels(), err)` convention
notes:
  - FetchCodexModels returns ([]ModelInfo, error), modeled as Effect.Effect<model_info[], Error>.
  - Reuses auth codex_cloudflare_headers and provider model metadata.
*/
