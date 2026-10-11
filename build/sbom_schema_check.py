"""Test-only bridge for CI validation of actual Python-generated inventories."""
import json
import os
from pathlib import Path
import subprocess
import tempfile


def check_generated_inventory(document):
    # Stdlib portable tests remain usable without frontend dependencies. The
    # dedicated CI schema job enables this and fails on a missing Node/Ajv tool.
    if os.environ.get('FLOW_SBOM_SCHEMA_CHECK') != '1':
        return
    with tempfile.TemporaryDirectory() as directory:
        inventory = Path(directory) / 'generated.cdx.json'
        inventory.write_text(json.dumps(document), encoding='utf-8')
        subprocess.run(['node', str(Path(__file__).with_name('validate_sbom.cjs')), str(inventory)],
                       check=True, timeout=30)
