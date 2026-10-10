// Build-time only: validate generated inventories against the unmodified,
// immutable official CycloneDX 1.6 schemas. No remote schema resolution.
const fs = require('node:fs');
const path = require('node:path');
const crypto = require('node:crypto');
const { createRequire } = require('node:module');
const requireWeb = createRequire(path.join(__dirname, '../web/package.json'));
const Ajv = requireWeb('ajv');
const applyFormats = requireWeb('ajv-formats-draft2019');

const SOURCE = '55343ba19dee1785acf1ce9191540d5fd7b590db';
const HASHES = Object.freeze({
  'bom-1.6.schema.json': '3e92dddbc30cf7f6a02b80f0942b1a4cfd4fb1c26f1dfc4310afa9d613cafb93',
  'spdx.schema.json': 'baa9d3bd1ed57b6751b0887edead6b5063ff53ff7429cf85d476c6c94af0166e',
  'jsf-0.82.schema.json': '8bae002c25e723db7ee1f26afde680ae1a2b1a8f6b4b4b0fd65dc3becb090aae',
  LICENSE: '6c29f22a4a7385285c6f579ec9f33c5e989f00739d6b257243a0b082ec9447ae',
});
let compiled;
function validator() {
  if (compiled) return compiled;
  const schemas = {};
  for (const [name, expected] of Object.entries(HASHES)) {
    const bytes = fs.readFileSync(path.join(__dirname, 'schema/cyclonedx-1.6', name));
    if (crypto.createHash('sha256').update(bytes).digest('hex') !== expected) {
      throw new Error('Official CycloneDX schema integrity mismatch');
    }
    if (name.endsWith('.json')) schemas[name] = JSON.parse(bytes);
  }
  const ajv = new Ajv({ allErrors: false, jsonPointers: true, logger: false });
  applyFormats(ajv, { formats: ['iri-reference', 'idn-email'] });
  ajv.addSchema(schemas['spdx.schema.json']);
  ajv.addSchema(schemas['jsf-0.82.schema.json']);
  compiled = ajv.compile(schemas['bom-1.6.schema.json']);
  return compiled;
}

function validate(document) {
  // The official schema intentionally declares specVersion as a string without
  // an enum. Explicitly select the supported version before using that schema.
  if (document?.specVersion !== '1.6') throw new Error('CycloneDX 1.6 schema rejected unsupported inventory version');
  const check = validator();
  if (!check(document)) {
    // Do not echo instance values, package paths, or schema descriptions.
    const error = check.errors[0];
    throw new Error(`CycloneDX 1.6 schema rejected inventory at ${error.dataPath || '/'} (${error.keyword})`);
  }
  // JSON Schema cannot express cross-object reference integrity. Flow's generated
  // inventories use a complete component graph, including empty leaf vertices.
  const refs = new Set();
  const add = component => {
    if (!component || typeof component['bom-ref'] !== 'string' || !component['bom-ref']) {
      throw new Error('Inventory component has no identity');
    }
    if (refs.has(component['bom-ref'])) throw new Error('Duplicate inventory component identity');
    refs.add(component['bom-ref']);
    for (const nested of component.components || []) add(nested);
  };
  add(document.metadata?.component);
  for (const component of document.components || []) add(component);
  const vertices = new Set();
  for (const edge of document.dependencies || []) {
    if (!refs.has(edge.ref) || vertices.has(edge.ref)) {
      throw new Error('Inventory graph has an unknown or duplicate vertex');
    }
    vertices.add(edge.ref);
    for (const ref of edge.dependsOn || []) {
      if (!refs.has(ref)) throw new Error('Inventory graph has a dangling dependency');
    }
  }
  if (vertices.size !== refs.size) throw new Error('Inventory graph omits a component');
  return { schema: 'CycloneDX 1.6', schema_source_commit: SOURCE,
    components: refs.size, dependency_vertices: vertices.size };
}

module.exports = { validate };
if (require.main === module) {
  try {
    if (process.argv.length !== 3) throw new Error('Usage: node build/validate_sbom.cjs <inventory.json>');
    const file = process.argv[2];
    if (fs.statSync(file).size > 16 * 1024 * 1024) throw new Error('Inventory exceeds the 16 MiB validation limit');
    console.log(JSON.stringify(validate(JSON.parse(fs.readFileSync(file, 'utf8')))));
  } catch (error) {
    // Parser/filesystem errors can include input or host paths; report their class.
    console.error(error instanceof SyntaxError ? 'Invalid inventory JSON' :
      error.code ? 'Inventory could not be read' : error.message);
    process.exitCode = 1;
  }
}
