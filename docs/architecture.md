# Architecture

Folio splits text into lowercase words. Letters and digits stay in a word. `the`, `is`, `in`, `of`, `and`, `to`, `for`, `a`, and `an` are dropped. A repeated word in one query counts once.

Each term points at the document ids that contain it. A search adds one to a document for every query term it contains. The sort is that count descending, then the id ascending. Replacing a document removes its old terms before the new ones are stored. Deleting a document removes it from every term.

`Open` reads `index.log`. Each line is one add or one delete, and the file is synced after the line is written. A final line that does not end in a newline is dropped on the next open, and the file is truncated back to the last complete line. The demo index is only in memory.
