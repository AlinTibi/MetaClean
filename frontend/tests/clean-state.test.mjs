import { test } from 'node:test';
import assert from 'node:assert/strict';
import { applyCleanResult, cleanSummary } from '../src/clean-state.ts';

const original = { id: 'f1', name: 'sample.jpg', path: 'sample.jpg', metadata: [{ tag: 'Artist', value: 'stale' }], status: 'sensitive_metadata_found' };

test('successful cleaning replaces inspector metadata, path, count and status', () => {
    const actual = { ...original, path: 'out/sample_clean.jpg', name: 'sample_clean.jpg', metadata: [], sensitiveCategories: [], status: 'clean' };
    const result = { fileId: 'f1', success: true, inspection: actual };
    assert.deepEqual(applyCleanResult([original], result), [actual]);
    assert.equal(original.metadata.length, 1);
});

test('failed verification clears stale metadata rather than claiming a clean file', () => {
    const actual = { ...original, metadata: [], status: 'error', error: 'cannot inspect output' };
    const result = { fileId: 'f1', success: false, inspection: actual, error: actual.error };
    assert.equal(applyCleanResult([original], result)[0], actual);
    assert.match(cleanSummary([result], [original]), /0\/1.*cannot inspect output/);
});

test('partial batch failure remains visible after the final done event', () => {
    const results = [{ fileId: 'f1', success: true }, { fileId: 'f2', success: false, error: 'read-only format' }];
    assert.match(cleanSummary(results, [original, { id: 'f2', name: 'sample.docx' }]), /1\/2.*sample\.docx: read-only format/);
});

test('unchanged failures preserve their inspection and late results do not restore removed files', () => {
    const files = [original];
    assert.equal(applyCleanResult(files, { fileId: 'f1', success: false })[0], original);
    const failed = applyCleanResult(files, { fileId: 'f1', success: false, error: 'read-only format' })[0];
    assert.equal(failed.metadata, original.metadata);
    assert.equal(failed.path, original.path);
    assert.equal(failed.cleanError, 'read-only format');
    assert.deepEqual(applyCleanResult([], { fileId: 'f1', inspection: original }), []);
});
