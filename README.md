# GoEscape

GoEscape is a static escape analysis tool for Go. It analyzes Go source code to determine which variables can be safely allocated on the STACK and which must escape to the HEAP. 

Unlike the standard Go compiler (`go build -gcflags='-m'`), which uses conservative algorithms for resolving interface methods and function pointers, GoEscape implements precise interprocedural analysis using Call Graph construction algorithms:
- **CHA (Class Hierarchy Analysis)**: Fast but conservative.
- **RTA (Rapid Type Analysis)**: More precise, pruning uninstantiated types.
- **VTA (Variable Type Analysis)**: The most precise, using data-flow analysis to resolve function calls.

GoEscape extracts the Go program into SSA (Static Single Assignment) form, builds a call graph using the chosen algorithm, and runs a data-flow fixpoint algorithm to compute the escape decisions.

## How to Run GoEscape

You can run GoEscape on any local Go package. To do this, place your `.go` files in a directory and run GoEscape pointing to that directory.

**Basic Usage:**
```bash
# Using CHA (Class Hierarchy Analysis)
go run . --cg=cha ./path/to/your/package/

# Using RTA (Rapid Type Analysis)
go run . --cg=rta ./path/to/your/package/

# Using VTA (Variable Type Analysis)
go run . --cg=vta ./path/to/your/package/
```

**Advanced Options:**
If you want to save the Callgraph JSON, and analysis JSON to an output directory, use the `--save` and `--out` flags:
```bash
go run . --cg=vta --save --out=./output ./path/to/your/package/
```

## Test Data

The project includes two sets of test suites for evaluating the escape analysis algorithms, located in the `testdata/` directory:

1. **`testdata/Synthetic/`**: A suite of custom, targeted test cases designed to test specific escape scenarios (e.g., closures, channels, goroutines, deep pointers, interfaces, struct fields, etc.).
2. **`testdata/GoByExample/`**: A collection of real-world patterns adapted from Go By Example.

Each test case is placed inside its own subfolder (e.g., `testdata/Synthetic/loops/loops.go`).

## Running the Evaluation

To automate the testing process and compare GoEscape against the standard Go compiler, the project includes an evaluation script.

1. **Run the script from the project root:**
   ```bash
   ./evaluate.sh
   ```

2. **What the script does:**
   The script iterates over every subfolder in `testdata/` (both Synthetic and GoByExample). For each test file, it runs:
   - The Go compiler's escape analysis (`go build -gcflags='-m'`)
   - GoEscape with CHA
   - GoEscape with RTA
   - GoEscape with VTA

3. **Where to see the results:**
   The results are saved as readable Markdown (`.md`) files in the `outputs/` directory. For example, the results for `testdata/Synthetic/loops/` will be saved at:
   ```
   outputs/Synthetic/loops.md
   ```
   Each generated `.md` file contains the exact terminal output from all four analyses side-by-side, making it easy to compare the precision of the different call graph algorithms against the official Go compiler.

## Result Replication via Docker

To ensure complete reproducibility of the results without needing to manually configure a Go environment, a `Dockerfile` is provided.

1. **Build the Docker image:**
   ```bash
   docker build -t goescape-evaluation .
   ```

2. **Run the evaluation container:**
   ```bash
   docker run --rm goescape-evaluation
   ```
   This will execute the `evaluate.sh` script inside the container and print the progress to your terminal. 

3. **Extract and Save the Outputs:**
   Running the command below will execute the evaluation and save all the generated `.md` evaluation files directly inside your local `outputs/` directory:
   ```bash
   docker run --rm -v "$(pwd)/outputs:/app/outputs" goescape-evaluation
   ```
