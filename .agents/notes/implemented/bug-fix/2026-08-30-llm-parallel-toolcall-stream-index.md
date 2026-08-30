# Agent Note: Rewrite parallel tool-call SSE by index

Status: implemented

## Problem

After a MiMo (or other OpenAI-compatible) turn that emitted many parallel `tool_calls`, the next `chat/completions` request failed immediately with `400 Param Incorrect`. A single tool call in the same run succeeded. langchaingo's stream assembler ignores the SSE `index` field and appends every type-less argument fragment onto the last tool call, so N-1 calls are echoed with `arguments:""` and the last call carries concatenated JSON. MiMo then rejects the roundtrip payload.

## Decision

The OpenAI-compatible HTTP stack rewrites `text/event-stream` `chat/completions` bodies before langchaingo parses them: argument deltas are merged by the JSON `index` field, then emitted as complete `type=function` tool-call events that langchaingo can append as distinct calls. Non-SSE JSON responses are not wrapped. HTTP 4xx/5xx bodies are logged with `error.message` and `error.param` because langchaingo's error type drops `param`. Argument-only deltas that omit `index` attach to the last merged tool on that choice so they do not all collapse onto slot 0.

## Alternatives considered

- **Fork langchaingo `updateToolCalls` to honor `index`.** Correct at the source, but a fork is out of scope; the transport already exists for thinking-field injection.
- **Add an SSE client library to split the stream.** Rejected: langchaingo already frames `data:` lines. A library would not restore `index` onto `tool_calls`, so the merge logic would still be owned and the swap fails the net-deletion bar ([deps policy](../process/2026-08-13-dependencies-over-hand-rolling.md)).
- **Disable parallel tool execution / prompt the model to call one tool.** Shrinks the blast radius without fixing the wire format; the model can still emit parallel `tool_calls` in one completion.
- **Validate arguments before echoing and rewrite empties to `{}`.** Masks the assembly bug; tools would still run with the wrong argument assignment.

## Consequences

Parallel tool-call streams from OpenAI-compatible providers (MiMo, DeepSeek, GPT) assemble with per-call arguments intact when SSE `index` is present. JSON completions keep their original bytes. 400 responses log the provider `param` field so the next mismatch is diagnosable from logs.

## Verification

`TestIndexedToolCallReaderRewritesStream` (two- and twelve-call indexed SSE, unindexed argument attach) and `TestLangchaingoAssemblesIndexedParallelToolCalls` (two, three, and twelve calls) in `pkg/agent/llm`. `TestToolCallIndexTransportLeavesNonSSEUnchanged` asserts JSON completions are byte-identical. `TestMimoToolRoundTripPayload` asserts `reasoning_content` and JSON object arguments on the echoed request. `TestErrorLogTransportRestoresAPIErrorBody` and `TestParseLLMAPIErrorExtractsMessageAndParam` cover 400 body logging.
