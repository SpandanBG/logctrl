# Graph Report - .  (2026-04-27)

## Corpus Check
- Corpus is ~3,749 words - fits in a single context window. You may not need a graph.

## Summary
- 118 nodes · 193 edges · 14 communities detected
- Extraction: 77% EXTRACTED · 23% INFERRED · 0% AMBIGUOUS · INFERRED: 45 edges (avg confidence: 0.8)
- Token cost: 0 input · 0 output

## Community Hubs (Navigation)
- [[_COMMUNITY_LogView Component|LogView Component]]
- [[_COMMUNITY_Architecture Documentation|Architecture Documentation]]
- [[_COMMUNITY_Ring Buffer|Ring Buffer]]
- [[_COMMUNITY_UI Layout System|UI Layout System]]
- [[_COMMUNITY_Prompt & View Rendering|Prompt & View Rendering]]
- [[_COMMUNITY_Toolbar Component|Toolbar Component]]
- [[_COMMUNITY_Main & Process Control|Main & Process Control]]
- [[_COMMUNITY_UI Model Orchestration|UI Model Orchestration]]
- [[_COMMUNITY_Stream Reader (V1)|Stream Reader (V1)]]
- [[_COMMUNITY_Responsive Sizing Types|Responsive Sizing Types]]
- [[_COMMUNITY_Mock Logger Tool|Mock Logger Tool]]
- [[_COMMUNITY_ANSI Constants|ANSI Constants]]
- [[_COMMUNITY_Command Executor|Command Executor]]
- [[_COMMUNITY_Input Prompt|Input Prompt]]

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
- `startChildProcess()` --calls--> `NewStreamV2()`  [INFERRED]
  main.go → reader/stream_v2.go
- `startChildProcess()` --calls--> `NewUI()`  [INFERRED]
  main.go → ui/layout.go
- `startPTY()` --calls--> `OnTerminalResize()`  [INFERRED]
  main.go → utils/signals.go
- `startDataPump()` --calls--> `copy()`  [INFERRED]
  main.go → reader/buffer.go
- `NewUI()` --calls--> `NewToolbar()`  [INFERRED]
  ui/layout.go → ui/components/toolbar.go

## Hyperedges (group relationships)
- **Log Data Flow: Pipe -> StreamV2 -> Buffer -> LogView** — claudemd_anonymous_pipe, claudemd_streamv2, claudemd_buffer, claudemd_logview [EXTRACTED 0.95]
- **UI Components implementing tea.Model** — claudemd_logview, claudemd_toolbar, claudemd_prompt [EXTRACTED 1.00]
- **Parent-Child IPC via Pipe and logctrl_child env var** — claudemd_parent_process, claudemd_child_process, claudemd_logctrl_child_env, claudemd_anonymous_pipe [EXTRACTED 0.95]

## Communities

### Community 0 - "LogView Component"
Cohesion: 0.18
Nodes (6): logView, teaLogCmd, TeaLogSizeUpdate, NewLogView(), StreamV2, NewStreamV2()

### Community 1 - "Architecture Documentation"
Cohesion: 0.2
Nodes (16): Anonymous Pipe, reader.Buffer, Child Process (startChildProcess), logctrl, logctrl_child Environment Variable, ui/components LogView, Parent-Child Process Model Design Rationale, Parent Process (main.go) (+8 more)

### Community 2 - "Ring Buffer"
Cohesion: 0.35
Nodes (7): copy(), NewBuffer(), Benchmark_Buffer(), Equal(), Test_BufferResizeTableDriven(), Test_BufferTableDriven(), Buffer

### Community 3 - "UI Layout System"
Cohesion: 0.2
Nodes (7): NewUI(), logTeaCmd, SizeFixed, SizeI, SizeModifier, SizeRatio, SizeType

### Community 4 - "Prompt & View Rendering"
Cohesion: 0.22
Nodes (3): prompt, TeaPromptToggle, NewPrompt()

### Community 5 - "Toolbar Component"
Cohesion: 0.24
Nodes (4): toolbar, NewToolbar(), ModifySize(), updateSize()

### Community 6 - "Main & Process Control"
Cohesion: 0.36
Nodes (7): main(), resizePty(), setupStreaming(), startChildProcess(), startDataPump(), startPTY(), OnTerminalResize()

### Community 7 - "UI Model Orchestration"
Cohesion: 0.47
Nodes (1): uiModel

### Community 8 - "Stream Reader (V1)"
Cohesion: 0.32
Nodes (3): Stream, NewStream(), startStream()

### Community 9 - "Responsive Sizing Types"
Cohesion: 0.83
Nodes (4): ui/utils Fixed Type, ui/utils Modifier Type, ui/utils Ratio Type, ui/utils Responsive Sizing

### Community 10 - "Mock Logger Tool"
Cohesion: 1.0
Nodes (2): makeLogs(), waitFor()

### Community 11 - "ANSI Constants"
Cohesion: 1.0
Nodes (0): 

### Community 12 - "Command Executor"
Cohesion: 1.0
Nodes (0): 

### Community 13 - "Input Prompt"
Cohesion: 1.0
Nodes (0): 

## Knowledge Gaps
- **7 isolated node(s):** `logTeaCmd`, `SizeType`, `SizeI`, `TeaLogSizeUpdate`, `TeaPromptToggle` (+2 more)
  These have ≤1 connection - possible missing edges or undocumented components.
- **Thin community `ANSI Constants`** (1 nodes): `constants.go`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Command Executor`** (1 nodes): `executor.go`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Input Prompt`** (1 nodes): `prompt.go`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.

## Suggested Questions
_Questions this graph is uniquely positioned to answer:_

- **Why does `NewUI()` connect `UI Layout System` to `LogView Component`, `Prompt & View Rendering`, `Toolbar Component`, `Main & Process Control`, `Stream Reader (V1)`?**
  _High betweenness centrality (0.279) - this node is a cross-community bridge._
- **Why does `NewLogView()` connect `LogView Component` to `UI Layout System`?**
  _High betweenness centrality (0.128) - this node is a cross-community bridge._
- **Why does `uiModel` connect `UI Model Orchestration` to `UI Layout System`, `Prompt & View Rendering`?**
  _High betweenness centrality (0.078) - this node is a cross-community bridge._
- **Are the 8 inferred relationships involving `NewUI()` (e.g. with `startChildProcess()` and `NewToolbar()`) actually correct?**
  _`NewUI()` has 8 INFERRED edges - model-reasoned connections that need verification._
- **Are the 4 inferred relationships involving `NewBuffer()` (e.g. with `Test_BufferTableDriven()` and `Test_BufferResizeTableDriven()`) actually correct?**
  _`NewBuffer()` has 4 INFERRED edges - model-reasoned connections that need verification._
- **What connects `logTeaCmd`, `SizeType`, `SizeI` to the rest of the system?**
  _7 weakly-connected nodes found - possible documentation gaps or missing edges._