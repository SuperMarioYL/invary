Output assets/demo.gif

Set FontSize 15
Set Width 1100
Set Height 600
Set Padding 20

# Build the single binary (m1: no API key, no network).
Type "go build -o invary ." Enter
Sleep 2s

# 1. The silent break: Kimi emitted arguments as single-quoted pseudo-JSON.
Type "./invary check --trace examples/sample_kimi_output.json --schema examples/tool-schema.json" Enter
Sleep 3s

# 2. A conforming trace: all four invariants pass.
Type "./invary check --trace examples/sample_deepseek_output.json --schema examples/tool-schema.json" Enter
Sleep 3s

# 3. Schema drift: an extra field the schema does not declare.
Type "./invary check --trace examples/sample_glm_output.json --schema examples/tool-schema.json" Enter
Sleep 4s
