const { test } = require('node:test');
const assert = require('node:assert/strict');
const { validate, readInventory } = require('./validate_sbom.cjs');
const fs = require('node:fs');
const os = require('node:os');
const path = require('node:path');

const inventory = () => ({ bomFormat: 'CycloneDX', specVersion: '1.6', version: 1,
  metadata: { component: { type: 'application', name: 'Flow', 'bom-ref': 'flow' } },
  components: [{ type: 'library', name: 'fixture', 'bom-ref': 'fixture',
    licenses: [{ license: { id: 'MIT' } }], hashes: [{ alg: 'SHA-256', content: 'a'.repeat(64) }] }],
  dependencies: [{ ref: 'flow', dependsOn: ['fixture'] }, { ref: 'fixture', dependsOn: [] }] });

test('official schema and complete dependency graph accept a valid inventory', () => {
  assert.equal(validate(inventory()).components, 2);
});

test('official schema rejects invalid versions, SPDX identifiers, hashes and unknown fields', () => {
  const changes = [x => { x.specVersion = '1.7'; },
    x => { x.components[0].licenses[0].license.id = 'not-a-license'; },
    x => { x.components[0].hashes[0].content = 'not-a-hash'; },
    x => { x.components[0].private_path = 'private-fixture'; }];
  for (const change of changes) {
    const document = inventory(); change(document);
    assert.throws(() => validate(document), /schema rejected/);
  }
});

test('dangling edges, duplicate identities, duplicate vertices and missing leaves fail closed', () => {
  const changes = [x => { x.dependencies[0].dependsOn = ['missing']; },
    x => { x.components.push({ ...x.components[0], name: 'different' }); },
    x => { x.dependencies.push({ ref: 'fixture', dependsOn: ['flow'] }); },
    x => { x.dependencies.pop(); }];
  for (const change of changes) {
    const document = inventory(); change(document);
    assert.throws(() => validate(document), /Inventory|Duplicate/);
  }
});

test('file loading accepts valid JSON and refuses oversized or non-file input', t => {
  const directory = fs.mkdtempSync(path.join(os.tmpdir(), 'flow-inventory-'));
  t.after(() => fs.rmSync(directory, { recursive: true, force: true }));
  const file = path.join(directory, 'inventory.json');
  fs.writeFileSync(file, JSON.stringify(inventory()));
  assert.equal(validate(readInventory(file)).components, 2);
  fs.writeFileSync(file, '{invalid');
  assert.throws(() => readInventory(file), SyntaxError);
  const descriptor = fs.openSync(file, 'w');
  try { fs.ftruncateSync(descriptor, 16 * 1024 * 1024 + 1); }
  finally { fs.closeSync(descriptor); }
  assert.throws(() => readInventory(file), /16 MiB/);
  assert.throws(() => readInventory(directory));
});
