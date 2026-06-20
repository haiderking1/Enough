// PORT STATUS: active
// source path: runtime/schemas.ts
// confidence: high
// status: phase_b_compile

import * as Schema from "@effect/schema/Schema";

// ==========================================
// Command Schemas (mirroring serve.go types)
// ==========================================

export const ListSessions = Schema.Struct({
  type: Schema.Literal("listSessions"),
});
export type ListSessions = Schema.Schema.Type<typeof ListSessions>;

export const OpenSession = Schema.Struct({
  type: Schema.Literal("openSession"),
  id: Schema.String,
});
export type OpenSession = Schema.Schema.Type<typeof OpenSession>;

export const NewSession = Schema.Struct({
  type: Schema.Literal("newSession"),
  cwd: Schema.optional(Schema.String),
});
export type NewSession = Schema.Schema.Type<typeof NewSession>;

export const DeleteSession = Schema.Struct({
  type: Schema.Literal("deleteSession"),
  id: Schema.String,
});
export type DeleteSession = Schema.Schema.Type<typeof DeleteSession>;

export const AttachmentSchema = Schema.Struct({
  mime: Schema.String,
  data: Schema.String, // base64
});
export type AttachmentSchema = Schema.Schema.Type<typeof AttachmentSchema>;

export const Prompt = Schema.Struct({
  type: Schema.Literal("prompt"),
  text: Schema.String,
  cwd: Schema.optional(Schema.String),
  attachments: Schema.optional(Schema.Array(AttachmentSchema)),
});
export type Prompt = Schema.Schema.Type<typeof Prompt>;

export const Interrupt = Schema.Struct({
  type: Schema.Literal("interrupt"),
});
export type Interrupt = Schema.Schema.Type<typeof Interrupt>;

export const SetModel = Schema.Struct({
  type: Schema.Literal("setModel"),
  provider: Schema.String,
  model: Schema.String,
  thinkingLevel: Schema.optional(Schema.String),
});
export type SetModel = Schema.Schema.Type<typeof SetModel>;

export const ListModels = Schema.Struct({
  type: Schema.Literal("listModels"),
});
export type ListModels = Schema.Schema.Type<typeof ListModels>;

export const DesktopCommand = Schema.Union(
  ListSessions,
  OpenSession,
  NewSession,
  DeleteSession,
  Prompt,
  Interrupt,
  SetModel,
  ListModels
);
export type DesktopCommand = Schema.Schema.Type<typeof DesktopCommand>;

// ==========================================
// Event Schemas (subset of backend/core/events.ts)
// ==========================================

export const AssistantStartEvent = Schema.Struct({
  kind: Schema.Literal("assistant_start"),
  data: Schema.Unknown,
});

export const AssistantDeltaEvent = Schema.Struct({
  kind: Schema.Literal("assistant_delta"),
  data: Schema.String,
});

export const ToolStartEvent = Schema.Struct({
  kind: Schema.Literal("tool_start"),
  data: Schema.Unknown,
});

export const ToolDeltaEvent = Schema.Struct({
  kind: Schema.Literal("tool_delta"),
  data: Schema.Unknown,
});

export const ToolResultEvent = Schema.Struct({
  kind: Schema.Literal("tool_result"),
  data: Schema.Unknown,
});

export const ErrorEvent = Schema.Struct({
  kind: Schema.Literal("error"),
  data: Schema.String,
});

export const SystemEvent = Schema.Struct({
  kind: Schema.Literal("system"),
  data: Schema.String,
});

export const CompactionStartEvent = Schema.Struct({
  kind: Schema.Literal("compaction_start"),
  data: Schema.Unknown,
});

export const CompactionEndEvent = Schema.Struct({
  kind: Schema.Literal("compaction_end"),
  data: Schema.Unknown,
});

export const DesktopEvent = Schema.Union(
  AssistantStartEvent,
  AssistantDeltaEvent,
  ToolStartEvent,
  ToolDeltaEvent,
  ToolResultEvent,
  ErrorEvent,
  SystemEvent,
  CompactionStartEvent,
  CompactionEndEvent
);
export type DesktopEvent = Schema.Schema.Type<typeof DesktopEvent>;

// ==========================================
// Decode/Encode Helpers
// ==========================================

export const decodeCommand = Schema.decodeUnknownOption(DesktopCommand);
export const encodeCommand = Schema.encode(DesktopCommand);

export const decodeEvent = Schema.decodeUnknownOption(DesktopEvent);
export const encodeEvent = Schema.encode(DesktopEvent);
