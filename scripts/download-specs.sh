#!/usr/bin/env bash
# Download and prepare Productboard v2 OpenAPI specs for code generation.
# The API returns OpenAPI 3.1.1 JSON; we patch to 3.0.3 YAML for oapi-codegen compatibility.
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SPECS_DIR="$SCRIPT_DIR/../specs"

echo "Downloading Productboard v2 OpenAPI specs..."
curl -sL -o "$SPECS_DIR/entities.json.tmp" https://developer.productboard.com/openapi/entities.yaml
curl -sL -o "$SPECS_DIR/notes.json.tmp"    https://developer.productboard.com/openapi/notes.yaml

echo "Converting to OpenAPI 3.0.3 YAML (oapi-codegen requires 3.0.x)..."
python3 - <<'PYEOF'
import json, yaml, sys, os

specs_dir = os.path.join(os.path.dirname(os.path.abspath(__file__)), '..', 'specs')

for name in ['entities', 'notes']:
    src = os.path.join(specs_dir, f'{name}.json.tmp')
    dst = os.path.join(specs_dir, f'{name}.yaml')

    with open(src) as f:
        data = json.load(f)

    # Patch: downgrade OpenAPI version from 3.1.x to 3.0.3
    data['openapi'] = '3.0.3'

    # Patch: convert nullable type arrays [type, null] to nullable: true (3.0 syntax)
    def patch_nullable(obj):
        if isinstance(obj, dict):
            if 'type' in obj and isinstance(obj['type'], list):
                types = [t for t in obj['type'] if t != 'null']
                obj['type'] = types[0] if len(types) == 1 else types
                obj['nullable'] = True
            if 'anyOf' in obj:
                non_null = [s for s in obj['anyOf'] if not (isinstance(s, dict) and s.get('type') == 'null')]
                if len(non_null) < len(obj['anyOf']):
                    if len(non_null) == 1 and '$ref' in non_null[0]:
                        obj['allOf'] = non_null
                        obj['nullable'] = True
                        del obj['anyOf']
                    else:
                        obj['anyOf'] = non_null
                        obj['nullable'] = True
            for v in obj.values():
                patch_nullable(v)
        elif isinstance(obj, list):
            for item in obj:
                patch_nullable(item)

    patch_nullable(data)

    with open(dst, 'w') as f:
        yaml.dump(data, f, allow_unicode=True, sort_keys=False, default_flow_style=False)

    os.remove(src)
    print(f"  ✓ specs/{name}.yaml written ({os.path.getsize(dst):,} bytes)")

PYEOF

echo "Done. Run 'go generate ./...' to regenerate the Go clients."
