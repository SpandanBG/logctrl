---
status: accepted
---

# Passes are updated by messages broadcast from the UI

Each Pass needs the user to change its state (the Search its regex, later Passes their own settings), and giving the Executor and `cmd.Stream` a method per Pass (`Search(regex)`, and one more for each Pass to come) ties the UI to every Pass's signature. So every Pass has the same `Update(msg any) error`, in the style of a BubbleTea `Update`: its messages are typed structs of its own (`SearchFor{Regex}`), and it ignores any message it does not know. The UI calls the Executor's one `Update(msg any) error`, which hands the message to every Pass, under its lock, and returns their errors joined. Every Pass is built once in `main`, idle until a message gives it something to do, and `NewExecutor(stream, search, others...)` takes the Search apart so it always runs last. We rejected JSON-encoded arguments (untyped, and parsed on every message), a sealed message interface (a marker method on every message for little safety), and routing each message to its own Pass (the Executor would have to know every message type).

Amended (#17): the Passes every Executor needs (the Filter, the Search and the Gutter) are no longer built in `main`: the Executor builds them itself, idle, and `NewExecutor(stream, renderer, others...)` takes only the other Passes, run after the Filter and before the Search; the Gutter, not the Search, now runs last (ADR 0009).

## Consequences

- A message no Pass knows is silently ignored, as in BubbleTea; a typo in a message type is not caught by the compiler.
- An idle Pass still runs on each read and leaves the Lines as they are.
- The messages that read the History outside the Window (`FindCurrentLine{}`, `PreviousMatch{}`, `NextMatch{}`) are broadcast like any other; the Search acts on them through the port the Executor gives it (see ADR 0008).
