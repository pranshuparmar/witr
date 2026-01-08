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

  # Show extended process information (memory, I/O, file descriptors)
  witr mysql --verbose

  # Combine flags: inspect port, show environment variables, output JSON
  witr --port 8080 --env --json

```

### Options

```
      --env           show environment variables for the process
  -h, --help          help for witr
      --json          show result as JSON
      --no-color      disable colorized output
      --pid string    pid to look up
      --port string   port to look up
      --short         show only ancestry
      --tree          show only ancestry as a tree
      --verbose       show extended process information
      --warnings      show only warnings
```

