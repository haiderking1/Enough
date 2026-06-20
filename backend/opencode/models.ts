// PORT: mirrors backend/opencode/models.go

import { Effect } from "effect";
import {
  provider_codex,
  provider_neuralwatt,
  provider_opencode,
  provider_opencode_zen,
  type model_info,
  codex_known_models,
  codex_models
} from "./providers";
import {
  opencode_mandatory_thinking_id,
  supported_thinking_levels,
  format_thinking_label,
  format_thinking_level_for_model,
  supports_thinking
} from "./thinking";
import {
  catalog_model,
  catalog_model_for_provider,
  refresh_models_dev_catalog,
  catalog_loaded_once,
  get_catalog_models
} from "./catalog";
import { fetch_codex_models, codex_context_fallback_for } from "./codex_catalog";
import { neuralwatt_known_models, neuralwatt_models, merge_neuralwatt_model } from "./neuralwatt";

export const known_models: Record<string, model_info> = {
  "deepseek-v4-flash": { id: "deepseek-v4-flash", name: "DeepSeek V4 Flash", context_window: 1_000_000, reasoning: true },
  "deepseek-v4-pro": { id: "deepseek-v4-pro", name: "DeepSeek V4 Pro", context_window: 1_000_000, reasoning: true },
  "glm-5": { id: "glm-5", name: "GLM-5", context_window: 202_752, reasoning: true },
  "glm-5.1": { id: "glm-5.1", name: "GLM-5.1", context_window: 202_752, reasoning: true },
  "kimi-k2.5": { id: "kimi-k2.5", name: "Kimi K2.5", context_window: 262_144, reasoning: true, supports_images: true },
  "kimi-k2.6": { id: "kimi-k2.6", name: "Kimi K2.6", context_window: 262_144, reasoning: true, supports_images: true },
  "mimo-v2.5": { id: "mimo-v2.5", name: "MiMo V2.5", context_window: 1_000_000, reasoning: true },
  "mimo-v2.5-pro": { id: "mimo-v2.5-pro", name: "MiMo V2.5 Pro", context_window: 1_048_576, reasoning: true },
  "mimo-v2-pro": { id: "mimo-v2-pro", name: "MiMo V2 Pro", context_window: 1_000_000, reasoning: true },
  "mimo-v2-omni": { id: "mimo-v2-omni", name: "MiMo V2 Omni", context_window: 1_000_000, reasoning: true, supports_images: true },
  "minimax-m2.5": { id: "minimax-m2.5", name: "MiniMax M2.5", context_window: 204_800, reasoning: true },
  "minimax-m2.7": { id: "minimax-m2.7", name: "MiniMax M2.7", context_window: 204_800, reasoning: true },
  "minimax-m3": { id: "minimax-m3", name: "MiniMax M3", context_window: 512_000, reasoning: true, supports_images: true },
  "qwen3.6-plus": { id: "qwen3.6-plus", name: "Qwen3.6 Plus", context_window: 1_000_000, reasoning: true },
  "qwen3.5-plus": { id: "qwen3.5-plus", name: "Qwen3.5 Plus", context_window: 1_000_000, reasoning: true },
  "qwen3.7-max": { id: "qwen3.7-max", name: "Qwen3.7 Max", context_window: 1_000_000, reasoning: true },
  "qwen3.7-plus": { id: "qwen3.7-plus", name: "Qwen3.7 Plus", context_window: 1_000_000, reasoning: true },
  "hy3-preview": { id: "hy3-preview", name: "HY3 Preview", context_window: 256_000, reasoning: true },
};

export class registry {
  private _models: model_info[] = [];
  private _zen_models: model_info[] = [];
  private _neuralwatt_models: model_info[] = [];
  private _codex_models: model_info[] = [];
  private _err: Error | null = null;
  private _zen_err: Error | null = null;
  private _neuralwatt_err: Error | null = null;
  private _codex_err: Error | null = null;

  models(): model_info[] {
    return [...this._models];
  }

  err(): Error | null {
    return this._err;
  }

  lookup(id: string): [model_info, boolean] {
    for (const m of this._models) {
      if (m.id === id) {
        return [m, true];
      }
    }
    const [cat_m, ok] = catalog_model(id);
    if (ok) {
      return [cat_m, true];
    }
    const [zen_m, ok_zen] = catalog_model_for_provider(provider_opencode_zen, id);
    if (ok_zen) {
      return [zen_m, true];
    }
    const known = known_models[id];
    if (known !== undefined) {
      return [normalize_model({ ...known }), true];
    }
    return [{ id: "", name: "", context_window: 0, reasoning: false }, false];
  }

  async refresh(
    ctx: AbortSignal | undefined,
    provider: string,
    endpoint: string,
    api_key: string
  ): Promise<Error | null> {
    try {
      await Effect.runPromise(refresh_models_dev_catalog(ctx));
    } catch {}

    try {
      const fetched = await Effect.runPromise(fetch_models(ctx, provider, endpoint, api_key));
      if (provider === provider_opencode_zen) {
        this._zen_models = fetched;
        this._zen_err = null;
      } else if (provider === provider_neuralwatt) {
        this._neuralwatt_models = fetched;
        this._neuralwatt_err = null;
      } else {
        this._models = fetched;
        this._err = null;
      }
      return null;
    } catch (err) {
      const error = err instanceof Error ? err : new Error(String(err));
      const models = fallback_models(provider);
      if (provider === provider_opencode_zen) {
        this._zen_models = models;
        this._zen_err = error;
      } else if (provider === provider_neuralwatt) {
        this._neuralwatt_models = models;
        this._neuralwatt_err = error;
      } else {
        this._models = models;
        this._err = error;
      }
      return error;
    }
  }

  err_for(provider: string): Error | null {
    if (provider === provider_opencode_zen) {
      return this._zen_err;
    }
    if (provider === provider_neuralwatt) {
      return this._neuralwatt_err;
    }
    return this._err;
  }

  zen_models_list(): model_info[] {
    if (this._zen_models.length > 0) {
      return [...this._zen_models];
    }
    return fallback_models(provider_opencode_zen);
  }

  neuralwatt_models_list(): model_info[] {
    if (this._neuralwatt_models.length > 0) {
      return [...this._neuralwatt_models];
    }
    return neuralwatt_models();
  }

  lookup_neuralwatt(id: string): [model_info, boolean] {
    for (const m of this._neuralwatt_models) {
      if (m.id === id) {
        return [m, true];
      }
    }
    const known = neuralwatt_known_models[id];
    if (known !== undefined) {
      return [normalize_model({ ...known }), true];
    }
    return [{ id: "", name: "", context_window: 0, reasoning: false }, false];
  }

  async refresh_codex(ctx: AbortSignal | undefined, access_token: string): Promise<Error | null> {
    try {
      const fetched = await Effect.runPromise(fetch_codex_models(ctx, access_token));
      this._codex_models = fetched;
      this._codex_err = null;
      return null;
    } catch (err) {
      const error = err instanceof Error ? err : new Error(String(err));
      this._codex_err = error;
      if (this._codex_models.length === 0) {
        this._codex_models = codex_models();
      }
      return error;
    }
  }

  codex_err(): Error | null {
    return this._codex_err;
  }

  codex_models_list(): model_info[] {
    if (this._codex_models.length > 0) {
      return [...this._codex_models];
    }
    return codex_models();
  }

  lookup_codex(id: string): [model_info, boolean] {
    for (const m of this._codex_models) {
      if (m.id === id) {
        return [m, true];
      }
    }
    const known = codex_known_models[id];
    if (known !== undefined) {
      return [normalize_model({ ...known }), true];
    }
    return [{ id: "", name: "", context_window: 0, reasoning: false }, false];
  }

  resolve_context_window(provider: string, model_id: string): number {
    if (provider === "") {
      provider = provider_opencode;
    }
    model_id = model_id.trim();
    if (model_id === "") {
      return 0;
    }

    switch (provider) {
      case provider_codex:
        for (const m of this._codex_models) {
          if (m.id === model_id && m.context_window > 0) {
            return m.context_window;
          }
        }
        const known_cdx = codex_known_models[model_id];
        if (known_cdx !== undefined && known_cdx.context_window > 0) {
          return known_cdx.context_window;
        }
        return codex_context_fallback_for(model_id);

      case provider_opencode_zen:
        for (const m of this._zen_models) {
          if (m.id === model_id && m.context_window > 0) {
            return m.context_window;
          }
        }
        const [m_zen, ok_zen] = catalog_model_for_provider(provider_opencode_zen, model_id);
        if (ok_zen && m_zen.context_window > 0) {
          return m_zen.context_window;
        }
        return 0;

      case provider_neuralwatt:
        for (const m of this._neuralwatt_models) {
          if (m.id === model_id && m.context_window > 0) {
            return m.context_window;
          }
        }
        const known_nw = neuralwatt_known_models[model_id];
        if (known_nw !== undefined && known_nw.context_window > 0) {
          return known_nw.context_window;
        }
        return 128_000;

      default:
        for (const m of this._models) {
          if (m.id === model_id && m.context_window > 0) {
            return m.context_window;
          }
        }
        const [m_cat, ok_cat] = catalog_model(model_id);
        if (ok_cat && m_cat.context_window > 0) {
          return m_cat.context_window;
        }
        const known_m = known_models[model_id];
        if (known_m !== undefined && known_m.context_window > 0) {
          return known_m.context_window;
        }
    }
    return 0;
  }
}

export const default_registry = new registry();

export const lookup_model = (id: string): [model_info, boolean] => {
  return default_registry.lookup(id);
};

export const model_context_window = (id: string): number => {
  return default_registry.resolve_context_window(provider_opencode, id);
};

export const fetch_models = (
  ctx: AbortSignal | undefined,
  provider: string,
  endpoint: string,
  api_key: string
): Effect.Effect<model_info[], Error> => {
  return Effect.tryPromise({
    try: async () => {
      const trimmed_endpoint = endpoint.replace(/\/+$/, "");
      const url = `${trimmed_endpoint}/models`;

      const headers: Record<string, string> = {};
      if (api_key !== "") {
        headers["Authorization"] = `Bearer ${api_key}`;
      }

      const controller = new AbortController();
      if (ctx) {
        ctx.addEventListener("abort", () => controller.abort());
      }
      const timeout = setTimeout(() => controller.abort(), 15000);

      try {
        const resp = await fetch(url, { signal: controller.signal, headers });
        const raw = await resp.text();
        clearTimeout(timeout);

        if (resp.status >= 400) {
          throw new Error(`models ${resp.status}: ${raw.trim()}`);
        }

        const list = JSON.parse(raw) as { data: { id: string }[] };
        const seen = new Set<string>();
        const out: model_info[] = [];
        for (const entry of list.data ?? []) {
          const id = (entry.id ?? "").trim();
          if (id === "") {
            continue;
          }
          if (seen.has(id)) {
            continue;
          }
          seen.add(id);

          const [m, ok] = catalog_model_for_provider(provider, id);
          if (ok) {
            out.push(m);
          } else if (provider === provider_neuralwatt) {
            out.push(merge_neuralwatt_model(id));
          }
        }

        if (out.length === 0) {
          return fallback_models(provider);
        }

        sort_models(out);
        return out;
      } catch (err) {
        clearTimeout(timeout);
        throw err instanceof Error ? err : new Error(String(err));
      }
    },
    catch: (cause) => cause instanceof Error ? cause : new Error(String(cause)),
  });
};

export const fallback_models = (provider: string): model_info[] => {
  const out = get_catalog_models(provider);
  if (out.length > 0) {
    sort_models(out);
    return out;
  }
  if (provider === provider_opencode_zen) {
    return [];
  }
  if (provider === provider_neuralwatt) {
    return neuralwatt_models();
  }
  const out_known: model_info[] = [];
  for (const id of Object.keys(known_models)) {
    out_known.push(merge_model(id));
  }
  sort_models(out_known);
  return out_known;
};

export const fallback_models_default = (): model_info[] => {
  return fallback_models(provider_opencode);
};

export const merge_model = (id: string): model_info => {
  const [m1, ok1] = catalog_model(id);
  if (ok1) return m1;
  const [m2, ok2] = catalog_model_for_provider(provider_opencode_zen, id);
  if (ok2) return m2;
  const known = known_models[id];
  if (known !== undefined) {
    return normalize_model({ ...known });
  }
  return { id: "", name: "", context_window: 0, reasoning: false };
};

export const normalize_model = (m: model_info): model_info => {
  if (m.mandatory_thinking === undefined) {
    m.mandatory_thinking = opencode_mandatory_thinking_id(m.id);
  }
  const levels = supported_thinking_levels(m.id);
  m.thinking_levels = levels.map((lvl) => ({
    id: lvl,
    name: format_thinking_label(lvl)
  }));
  return m;
};

export const sort_models = (models: model_info[]): void => {
  models.sort((a, b) => a.name.toLowerCase().localeCompare(b.name.toLowerCase()));
};

export const format_context_window = (n: number): string => {
  if (n >= 1_000_000) {
    if (n % 1_000_000 === 0) {
      return `${n / 1_000_000}M`;
    }
    return `${(n / 1_000_000).toFixed(1)}M`;
  }
  if (n >= 1000) {
    if (n % 1000 === 0) {
      return `${n / 1000}k`;
    }
    return `${(n / 1000).toFixed(1)}k`;
  }
  return `${n}`;
};

export const format_thinking_badge = (m: model_info, level: string): string => {
  if (!supports_thinking(m.id)) {
    if (m.reasoning) {
      return "reasoning";
    }
    return "";
  }
  return format_thinking_level_for_model(m.id, level as any);
};

export const supports_images = (model: string): boolean => {
  model = model.trim().toLowerCase();
  const [m, ok] = catalog_model(model);
  if (ok) {
    return !!m.supports_images;
  }
  if (model.startsWith("gpt-5")) {
    return true;
  }
  return false;
};

/*
PORT STATUS
source path: backend/opencode/models.go
source lines: 450
draft lines: 440
confidence: high
status: phase_b_compile
*/
