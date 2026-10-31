---
status: accepted
---

# The Window lives with the Passes, in `cmd`

The Window counts only the Lines the Filter does not hide (GLOSSARY), but two modules held it. `reader.Window` placed it by Line Number and knew nothing of the Filter. The Executor in `cmd` grew it on every read until it had as many shown Lines as its height, and never kept the result. So Scroll, centring the Current Line and the Search's first step all acted on a Window nobody sees (#18). The Executor also embedded the whole `reader.Stream`, so most of its interface passed straight through, and `headless` turned Scroll, Resize and Follow into reader calls.

The Window now lives in `cmd`. It is a type the Executor holds, guarded by the Executor's lock alongside the Passes, and it reads the History through the `reader.Stream` it is given (`Read`, `Total`, `First`). `reader.Window` is gone, and `reader.Stream` is the History of the Log Feed plus `Start` and `Close`. Scroll, Resize and Follow are messages to the Executor's `Update`, like any other:
- `Scroll` is handed to the Passes, then moves the Window.
- `Resize` resizes the Window and never reaches the Passes, so it cancels no step.
- `Follow` is handed to the Passes and makes the Window Following, both under the one lock, so no step still scanning can Pin the Window after it.

`headless.Logs.Update` hands every message on. The Executor holds the `reader.Stream` instead of embedding it, so its interface is `cmd.Stream`'s: `Start`, `Lines`, `Update` and `Close`.

We rejected two alternatives:
- **A Window package of its own between `reader` and `cmd`:** the Filter's verdict on each Line would have to be passed across to it.
- **Giving `reader.Window` a function that says which Lines are hidden:** `reader` would learn of the Filter, and of the order the Passes run in.

## Consequences

- ADR 0001 (the Window addresses Line Numbers) and ADR 0002 (Following is derived on read) still hold, inside `cmd`. The Window's behaviour is unchanged by the move: Scroll, centring and the first step still go by Line Number, and the Executor still grows the Window on read. #18 makes them Filter-aware.
- The Window reads through the `reader.Stream` the Executor is given, so a test double wrapping it sees the Window's own reads as well as a step's scan. Doubles that act on a scan pick out reads longer than a Window (`logfeed.Gated`).
- The Window is tested through the Executor's `Update` and `Lines`, and the History through `History.Lines` and `reader.Stream.Read`.
- Amended (#26): the Window also grows itself on read while the Filter hides Lines, given the Passes before the Search by the Executor, and the Search makes it Following on a new regex through its port. The Search's port still sees the Window without the Lines taken in (#18).
- Amended (#30): the Window answers Scroll, Resize and Follow in its own `Update`, and satisfies the Executor's `Pass` interface, with `Run` giving its shown Lines whatever it is handed. The Executor only broadcasts: each message reaches the Window first, then the other Passes, then the Search, then the Gutter, under the one lock. `Resize` now reaches every Pass, and the Search ignores it, so it still cancels no step. The cost: the Search cancels on any message it does not know, so any other message only the Window answers must be ignored by the Search too. The Window is still not a Pass in GLOSSARY.
