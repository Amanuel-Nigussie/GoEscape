#!/usr/bin/env bash
# ─────────────────────────────────────────────────────────────────────────────
# evaluate.sh — Run Go compiler escape analysis + GoEscape (CHA/RTA/VTA)
#               on every subfolder inside each testdata/ folder and collect
#               the terminal output into per-subfolder JSON files in outputs/.
#
# Layout:
#   testdata/
#     folder1/
#       sub1/          <- each subfolder is a standalone Go package
#         main.go
#       sub2/
#         main.go
#     folder2/
#       sub1/
#         main.go
#
# Produces:
#   outputs/
#     folder1/
#       sub1.md      <- readable markdown with 4 analysis outputs
#       sub2.md
#     folder2/
#       sub1.md
# ─────────────────────────────────────────────────────────────────────────────
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
TESTDATA_DIR="${SCRIPT_DIR}/testdata"
OUTPUTS_DIR="${SCRIPT_DIR}/outputs"


for folder in "${TESTDATA_DIR}"/*/; do
    [ -d "${folder}" ] || continue

    folder_name="$(basename "${folder}")"

    echo "═══════════════════════════════════════════════════════════════"
    echo "  Processing folder: ${folder_name}"
    echo "═══════════════════════════════════════════════════════════════"

    # Create a matching subdirectory in outputs/
    mkdir -p "${OUTPUTS_DIR}/${folder_name}"

    for subfolder in "${folder}"*/; do
        [ -d "${subfolder}" ] || continue

        subfolder_name="$(basename "${subfolder}")"
        md_file="${OUTPUTS_DIR}/${folder_name}/${subfolder_name}.md"

        go_files=("${subfolder}"*.go)
        if [ ! -e "${go_files[0]}" ]; then
            echo "  ⚠  Skipping ${folder_name}/${subfolder_name} — no .go files found"
            continue
        fi

        test_file_name="$(basename "${go_files[0]}")"

        echo ""
        echo "  ── ${folder_name}/${subfolder_name}/${test_file_name} ──"

        # Go compiler escape analysis
        echo "    → go build -gcflags='-m' ..."
        gcflags_output=$(cd "${subfolder}" && go build -gcflags='-m' ./... 2>&1) || true

        # GoEscape — CHA
        echo "    → GoEscape --cg=cha ..."
        cha_output=$(cd "${SCRIPT_DIR}" && go run . --cg=cha "${subfolder}" 2>&1) || true

        # GoEscape — RTA
        echo "    → GoEscape --cg=rta ..."
        rta_output=$(cd "${SCRIPT_DIR}" && go run . --cg=rta "${subfolder}" 2>&1) || true

        # GoEscape — VTA
        echo "    → GoEscape --cg=vta ..."
        vta_output=$(cd "${SCRIPT_DIR}" && go run . --cg=vta "${subfolder}" 2>&1) || true

        # Write Markdown
        cat > "${md_file}" <<EOF
# ${test_file_name}

## Go Compiler Escape Analysis (\`go build -gcflags='-m'\`)

\`\`\`
${gcflags_output}
\`\`\`

## GoEscape — CHA (Class Hierarchy Analysis)

\`\`\`
${cha_output}
\`\`\`

## GoEscape — RTA (Rapid Type Analysis)

\`\`\`
${rta_output}
\`\`\`

## GoEscape — VTA (Variable Type Analysis)

\`\`\`
${vta_output}
\`\`\`
EOF

        echo "    ✓ Saved → ${md_file}"
    done

    echo ""
done

echo "════════════════════════════════════════════════════════════════"
echo "  All done! Results are in: ${OUTPUTS_DIR}/"
echo "════════════════════════════════════════════════════════════════"
