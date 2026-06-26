import { default_registry } from "./models";

// ResolveContextWindow returns the context limit for a provider/model pair.
export const resolve_context_window = (provider: string, model_id: string): number => {
  return default_registry.resolve_context_window(provider, model_id);
};

