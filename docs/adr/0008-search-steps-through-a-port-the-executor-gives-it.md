---
status: accepted
---

# The Search steps between Lines with a Match through a port the Executor gives it

ADR 0007 had the Executor find and step between Lines with a Match itself, so that no Pass reached outside the Window. That left Search logic, and Search state, in the Executor, whose job is only to run the Passes together: hand each message to every Pass, run every Pass over the Lines, and join their shown text for the UI. So all of it moves into the Search, and the Executor gives the Search a narrow port onto the History instead (`history`, declared in `search.go` and attached by `NewExecutor`). The port can read: the Line Numbers in the Window, Total, First, and the Lines in a range of Line Numbers as every Pass but the Search leaves them. It can do one thing to the Window: Pin it centred on a Line Number. The Executor implements `Read` by releasing its lock while it reads the Stream and then running the other Passes over what it read, so running the Passes and the lock stay in the Executor, and the Search matches in what the user would see (ADR 0005). The step messages (`FindCurrentLine{}`, `PreviousMatch{}`, `NextMatch{}`) are now broadcast to every Pass like any other message; the other Passes ignore them. The Search counts every other message it gets, and since every message reaches it, it cancels its own scan when a message arrives during one. We rejected giving the Search the whole `reader.Stream` (it could close or resize the Stream, would read raw text, and would have to release the Executor's lock itself), giving every Pass the port in its `Update` (every Pass would change for the sake of one), and having the Search return read and Pin requests for the Executor to carry out (the Executor would then run the scan, batch by batch).

## Consequences

- A Pass can no longer be counted on to see only the Window: the Search reads the History through its port. No Pass can write to the History, and the only way a Pass can move the Window is to Pin it.
- The Executor keeps no Search state. Its `Update` hands every message to every Pass, the Search last, and joins their errors.
- Amended (#18): the port's `Window` gives the Line Numbers of the Lines in the Window the Filter does not hide, Lines taken in included, so the first step starts from the bottom shown Line; `PinCentredOn` centres the Current Line among those Lines. The Search itself is unchanged.
