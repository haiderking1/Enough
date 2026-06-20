// Import this once at runtime boot — registers all Agent methods.
// Side-effect imports attach Agent.prototype methods in safe dependency order.
// ES module hoisting guarantees the Agent class is available before any
// prototype assignment runs.

import { Agent, New } from "./agent";

// Core infrastructure first (guard/compaction/evidence wiring)
import "./session_fingerprints";
import "./overflow";
import "./guard";
import "./compaction";
import "./loop";
import "./prompt";
import "./system_prompt";
import "./models";
import "./goal_lock";
import "./memory_correction";
import "./coding_context";

// Tool implementations
import "./tools";
import "./tools_registry";
import "./tool_path";
import "./read_file";
import "./write_file";
import "./edit_file";
import "./list_dir";
import "./glob";
import "./grep";
import "./bash";
import "./browser";
import "./web_search";
import "./web_fetch";
import "./verifier";
import "./completion";
import "./stuck";
import "./step_score";
import "./parallel_fork";
import "./agent_swarm";
import "./workflow_worker";
import "./background_review";
import "./curator";
import "./notify_helpers";

export { Agent, New };
