import type { CleanFileResult, FileEntry } from './types';

export function applyCleanResult(files: FileEntry[], result: CleanFileResult): FileEntry[] {
    return files.map(file => {
        if (file.id !== result.fileId) return file;
        if (result.inspection) return result.inspection;
        if (!result.success && result.error) return { ...file, cleanError: result.error };
        return file;
    });
}

export function cleanSummary(results: CleanFileResult[], files: FileEntry[]): string {
    const ok = results.filter(result => result.success).length;
    const failures = results.filter(result => !result.success).map(result => {
        const file = files.find(entry => entry.id === result.fileId);
        return `${file?.name ?? result.fileId}: ${result.error ?? 'Cleaning failed'}`;
    });
    return `Cleaning finished: ${ok}/${results.length} file(s) succeeded.` +
        (failures.length ? ` ${failures.join(' | ')}` : ' Inspector and reports show the cleaned files.');
}
