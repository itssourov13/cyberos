// Package ai defines the provider-neutral Brain contract and Plan intermediate
// representation (Phase 12A).
//
// Trust model:
//   - AI output is untrusted.
//   - Plan validation is structural only — not authorization.
//   - Plans target capabilities, never executables or shell commands.
//   - There is no path from Brain to Broker/Executor in this phase.
//
// Future providers (OpenAI-compatible, local LLM, external JARVIS/Zoya) implement
// Brain behind adapters without changing the Plan IR or security spine.
package ai
