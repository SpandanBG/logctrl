# Graph Report - /Users/sudokid/Projects/GoLang/logctrl  (2026-04-27)

## Corpus Check
- 15 files · ~13,174 words
- Verdict: corpus is large enough that graph structure adds value.

## Summary
- 111 nodes · 183 edges · 13 communities detected
- Extraction: 77% EXTRACTED · 23% INFERRED · 0% AMBIGUOUS · INFERRED: 43 edges (avg confidence: 0.8)
- Token cost: 0 input · 0 output

## Community Hubs (Navigation)
- [[_COMMUNITY_Community 0|Community 0]]
- [[_COMMUNITY_Community 1|Community 1]]
- [[_COMMUNITY_Community 2|Community 2]]
- [[_COMMUNITY_Community 3|Community 3]]
- [[_COMMUNITY_Community 4|Community 4]]
- [[_COMMUNITY_Community 5|Community 5]]
- [[_COMMUNITY_Community 6|Community 6]]
- [[_COMMUNITY_Community 7|Community 7]]
- [[_COMMUNITY_Community 8|Community 8]]
- [[_COMMUNITY_Community 9|Community 9]]
- [[_COMMUNITY_Community 10|Community 10]]
- [[_COMMUNITY_Community 11|Community 11]]
- [[_COMMUNITY_Community 12|Community 12]]

## God Nodes (most connected - your core abstractions)
1. `uiModel` - 10 edges
2. `NewUI()` - 9 edges
3. `logView` - 9 edges
4. `Buffer` - 7 edges
5. `NewBuffer()` - 7 edges
6. `Test_BufferResizeTableDriven()` - 6 edges
7. `prompt` - 6 edges
8. `reader.StreamV2` - 6 edges
9. `main()` - 5 edges
10. `startPTY()` - 5 edges

## Surprising Connections (you probably didn't know these)
- `startChildProcess()` --calls--> `NewUI()`  [INFERRED]
  main.go → ui/layout.go
- `startPTY()` --calls--> `OnTerminalResize()`  [INFERRED]
  main.go → utils/signals.go
- `startDataPump()` --calls--> `copy()`  [INFERRED]
  main.go → reader/buffer.go
- `startChildProcess()` --calls--> `NewStream()`  [INFERRED]
  main.go → reader/stream.go
- `NewUI()` --calls--> `NewToolbar()`  [INFERRED]
  ui/layout.go → ui/components/toolbar.go

## Hyperedges (group relationships)
- **Log Data Flow: Pipe -> StreamV2 -> Buffer -> LogView** — claudemd_anonymous_pipe, claudemd_streamv2, claudemd_buffer, claudemd_logview [EXTRACTED 0.95]
- **UI Components implementing tea.Model** — claudemd_logview, claudemd_toolbar, claudemd_prompt [EXTRACTED 1.00]
- **Parent-Child IPC via Pipe and logctrl_child env var** — claudemd_parent_process, claudemd_child_process, claudemd_logctrl_child_env, claudemd_anonymous_pipe [EXTRACTED 0.95]

## Communities

### Community 0 - "Community 0"
Cohesion: 0.18
Nodes (10): NewLogView(), main(), resizePty(), setupStreaming(), startChildProcess(), startDataPump(), startPTY(), Stream (+2 more)

### Community 1 - "Community 1"
Cohesion: 0.2
Nodes (16): Anonymous Pipe, reader.Buffer, Child Process (startChildProcess), logctrl, logctrl_child Environment Variable, ui/components LogView, Parent-Child Process Model Design Rationale, Parent Process (main.go) (+8 more)

### Community 2 - "Community 2"
Cohesion: 0.35
Nodes (7): copy(), NewBuffer(), Benchmark_Buffer(), Equal(), Test_BufferResizeTableDriven(), Test_BufferTableDriven(), Buffer

### Community 3 - "Community 3"
Cohesion: 0.2
Nodes (7): NewUI(), logTeaCmd, SizeFixed, SizeI, SizeModifier, SizeRatio, SizeType

### Community 4 - "Community 4"
Cohesion: 0.31
Nodes (3): logView, teaLogCmd, TeaLogSizeUpdate

### Community 5 - "Community 5"
Cohesion: 0.22
Nodes (3): prompt, TeaPromptToggle, NewPrompt()

### Community 6 - "Community 6"
Cohesion: 0.24
Nodes (4): toolbar, NewToolbar(), ModifySize(), updateSize()

### Community 7 - "Community 7"
Cohesion: 0.47
Nodes (1): uiModel

### Community 8 - "Community 8"
Cohesion: 0.83
Nodes (4): ui/utils Fixed Type, ui/utils Modifier Type, ui/utils Ratio Type, ui/utils Responsive Sizing

### Community 9 - "Community 9"
Cohesion: 1.0
Nodes (2): makeLogs(), waitFor()

### Community 10 - "Community 10"
Cohesion: 1.0
Nodes (0): 

### Community 11 - "Community 11"
Cohesion: 1.0
Nodes (0): 

### Community 12 - "Community 12"
Cohesion: 1.0
Nodes (0): 

## Knowledge Gaps
- **7 isolated node(s):** `logTeaCmd`, `SizeType`, `SizeI`, `TeaLogSizeUpdate`, `TeaPromptToggle` (+2 more)
  These have ≤1 connection - possible missing edges or undocumented components.
- **Thin community `Community 10`** (1 nodes): `constants.go`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 11`** (1 nodes): `executor.go`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 12`** (1 nodes): `prompt.go`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.

## Suggested Questions
_Questions this graph is uniquely positioned to answer:_

- **Why does `NewUI()` connect `Community 3` to `Community 0`, `Community 5`, `Community 6`?**
  _High betweenness centrality (0.266) - this node is a cross-community bridge._
- **Why does `NewLogView()` connect `Community 0` to `Community 3`, `Community 4`?**
  _High betweenness centrality (0.141) - this node is a cross-community bridge._
- **Why does `uiModel` connect `Community 7` to `Community 3`, `Community 5`?**
  _High betweenness centrality (0.081) - this node is a cross-community bridge._
- **Are the 8 inferred relationships involving `NewUI()` (e.g. with `startChildProcess()` and `NewToolbar()`) actually correct?**
  _`NewUI()` has 8 INFERRED edges - model-reasoned connections that need verification._
- **Are the 4 inferred relationships involving `NewBuffer()` (e.g. with `.SetBufferSize()` and `Test_BufferTableDriven()`) actually correct?**
  _`NewBuffer()` has 4 INFERRED edges - model-reasoned connections that need verification._
- **What connects `logTeaCmd`, `SizeType`, `SizeI` to the rest of the system?**
  _7 weakly-connected nodes found - possible documentation gaps or missing edges._