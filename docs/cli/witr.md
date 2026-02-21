## witr

Why is this running?

### Synopsis

witr explains why a process or port is running by tracing its ancestry.

```
witr [process name] [flags]
```

### Examples

```

  # Inspect a running process by name
  witr nginx

  # Look up a process by PID
  witr --pid 1234

  # Find the process listening on a specific port
  witr --port 5432

  # Find the process holding a lock on a file
  witr --file /var/lib/dpkg/lock

  # Inspect a process by name with exact matching (no fuzzy search)
  witr bun --exact

  # Show the full process ancestry (who started whom)
  witr postgres --tree

  # Show only warnings (suspicious env, arguments, parents)
  witr docker --warnings

  # Display only environment variables of the process
  witr node --env

  # Short, single-line output (useful for scripts)
  witr sshd --short

  # Disable colorized output (CI or piping)
  witr redis --no-color

  # Output machine-readable JSON
  witr chrome --json

  # Launch Interactive Mode
  witr
  or
  witr --interactive

  # Show extended process information (memory, I/O, file descriptors)
  witr mysql --verbose

  # Combine flags: inspect port, show environment variables, output JSON
  witr --port 8080 --env --json

```

### Options

```
      --env           show environment variables for the process
  -x, --exact         use exact name matching (no substring search)
  -f, --file string   file path to find process for
  -h, --help          help for witr
  -i, --interactive   interactive mode (TUI)
      --json          show result as JSON
      --no-color      disable colorized output
  -p, --pid string    pid to look up
  -o, --port string   port to look up
  -s, --short         show only ancestry
  -t, --tree          show only ancestry as a tree
      --verbose       show extended process information
      --warnings      show only warnings
```

