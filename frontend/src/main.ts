import './style.css';
import { applyCleanResult, cleanSummary } from './clean-state';
import {
    AddFilesDialog, AddFolderDialog, RemoveFiles, ClearFiles, GetFiles,
    StartClean, CancelClean, SelectOutputFolder, OpenOutputFolder,
    ExportMetadataCSV, ExportPrivacyReportJSON, GetSettings, SaveSettings, GetEngineStatus,
} from '../wailsjs/go/main/App';
import { EventsOn } from '../wailsjs/runtime/runtime';
import {
    CATEGORY_LABELS, STATUS_LABELS,
    type Category, type CleanFileResult, type CleanProgress, type EngineStatus, type FileEntry,
    type MetadataEntry, type Profile, type Settings,
} from './types';

// --- DOM references -------------------------------------------------------

const $ = <T extends HTMLElement>(id: string) => document.getElementById(id) as T;

const btnAddFiles = $<HTMLButtonElement>('btn-add-files');
const btnAddFolder = $<HTMLButtonElement>('btn-add-folder');
const btnRemoveSelected = $<HTMLButtonElement>('btn-remove-selected');
const btnClear = $<HTMLButtonElement>('btn-clear');
const btnClean = $<HTMLButtonElement>('btn-clean');
const btnCancel = $<HTMLButtonElement>('btn-cancel');
const btnSettings = $<HTMLButtonElement>('btn-settings');
const btnOutputFolder = $<HTMLButtonElement>('btn-output-folder');
const btnOpenOutputFolder = $<HTMLButtonElement>('btn-open-output-folder');
const profileSelect = $<HTMLSelectElement>('profile-select');
const customCategoriesBar = $<HTMLDivElement>('custom-categories');

const queueBody = $<HTMLTableSectionElement>('queue-body');
const queueSummary = $<HTMLSpanElement>('queue-summary');
const selectAllBox = $<HTMLInputElement>('select-all');
const dropHint = $<HTMLDivElement>('drop-hint');

const inspectorTitle = $<HTMLSpanElement>('inspector-title');
const inspectorBody = $<HTMLDivElement>('inspector-body');
const metadataSearch = $<HTMLInputElement>('metadata-search');
const btnExportCsv = $<HTMLButtonElement>('btn-export-csv');
const btnExportJson = $<HTMLButtonElement>('btn-export-json');

const statusText = $<HTMLSpanElement>('status-text');
const engineStatusEl = $<HTMLSpanElement>('engine-status');
const progressFill = $<HTMLDivElement>('progress-fill');

const settingsModal = $<HTMLDivElement>('settings-modal');
const setDefaultProfile = $<HTMLSelectElement>('set-default-profile');
const setSuffix = $<HTMLInputElement>('set-suffix');
const setPreserveTimestamps = $<HTMLInputElement>('set-preserve-timestamps');
const setReplaceOriginal = $<HTMLInputElement>('set-replace-original');
const replaceWarning = $<HTMLParagraphElement>('replace-warning');
const btnSettingsCancel = $<HTMLButtonElement>('btn-settings-cancel');
const btnSettingsSave = $<HTMLButtonElement>('btn-settings-save');

const confirmModal = $<HTMLDivElement>('confirm-modal');
const btnConfirmCancel = $<HTMLButtonElement>('btn-confirm-cancel');
const btnConfirmProceed = $<HTMLButtonElement>('btn-confirm-proceed');

// --- State -----------------------------------------------------------------

let files: FileEntry[] = [];
let selected = new Set<string>();
let inspectedId: string | null = null;
let settings: Settings;
let outputFolder = '';
let cleaning = false;

// "other" is intentionally excluded: it has no removal args (no safe
// blanket mapping — see scanner.CategoryRemovalArgs), so offering it as a
// selectable Custom cleaning category would be a no-op checkbox.
const CUSTOM_CATEGORIES: Category[] = ['gps', 'author', 'camera', 'software', 'company', 'comments', 'timestamps'];

// --- Status bar helpers ------------------------------------------------

function setStatus(message: string): void {
    statusText.textContent = message;
    statusText.title = message;
}

function setProgress(fraction: number): void {
    progressFill.style.width = `${Math.round(fraction * 100)}%`;
}

// --- Queue rendering ---------------------------------------------------

function formatSize(bytes: number): string {
    if (bytes < 1024) return `${bytes} B`;
    const units = ['KB', 'MB', 'GB'];
    let value = bytes / 1024;
    let i = 0;
    while (value >= 1024 && i < units.length - 1) {
        value /= 1024;
        i++;
    }
    return `${value.toFixed(value < 10 ? 1 : 0)} ${units[i]}`;
}

function renderQueue(): void {
    queueBody.innerHTML = '';
    dropHint.classList.toggle('hidden', files.length > 0);

    for (const f of files) {
        const tr = document.createElement('tr');
        tr.dataset.id = f.id;
        if (selected.has(f.id)) tr.classList.add('selected');

        const checkTd = document.createElement('td');
        const checkbox = document.createElement('input');
        checkbox.type = 'checkbox';
        checkbox.checked = selected.has(f.id);
        checkbox.addEventListener('click', (e) => e.stopPropagation());
        checkbox.addEventListener('change', () => {
            if (checkbox.checked) selected.add(f.id); else selected.delete(f.id);
            tr.classList.toggle('selected', checkbox.checked);
            updateToolbarState();
        });
        checkTd.appendChild(checkbox);

        const nameTd = document.createElement('td');
        nameTd.className = 'name-cell';
        nameTd.textContent = f.name;
        nameTd.title = f.path;

        const typeTd = document.createElement('td');
        typeTd.textContent = f.ext.replace('.', '').toUpperCase();

        const sizeTd = document.createElement('td');
        sizeTd.textContent = formatSize(f.size);

        const metaTd = document.createElement('td');
        metaTd.textContent = f.status === 'unsupported' || f.status === 'error' ? '—' : String(f.metadata.length);

        const statusTd = document.createElement('td');
        const badge = document.createElement('span');
        badge.className = `status-badge status-${f.status}`;
        badge.textContent = STATUS_LABELS[f.status];
        if (f.cleanError) badge.textContent = `Cleaning failed · ${STATUS_LABELS[f.status]}`;
        if (f.error) badge.title = f.error;
        if (f.cleanError) badge.title = f.cleanError;
        statusTd.appendChild(badge);

        tr.append(checkTd, nameTd, typeTd, sizeTd, metaTd, statusTd);
        tr.addEventListener('click', () => {
            inspectedId = f.id;
            renderInspector();
            renderQueue();
        });
        if (f.id === inspectedId) tr.style.outline = `1px solid var(--accent)`;

        queueBody.appendChild(tr);
    }

    const supported = files.filter((f) => f.status !== 'unsupported' && f.status !== 'error').length;
    queueSummary.textContent = files.length ? `${files.length} file(s), ${supported} inspectable` : '';

    updateToolbarState();
}

function updateToolbarState(): void {
    btnRemoveSelected.disabled = selected.size === 0 || cleaning;
    btnClear.disabled = files.length === 0 || cleaning;
    btnClean.disabled = selected.size === 0 || cleaning;
    selectAllBox.checked = files.length > 0 && selected.size === files.length;
    selectAllBox.indeterminate = selected.size > 0 && selected.size < files.length;
}

// --- Inspector -----------------------------------------------------------

function renderInspector(): void {
    const file = files.find((f) => f.id === inspectedId);
    metadataSearch.disabled = !file;
    btnExportCsv.disabled = files.length === 0;
    btnExportJson.disabled = files.length === 0;

    if (!file) {
        inspectorTitle.textContent = 'Metadata Inspector';
        inspectorBody.innerHTML = '<div class="inspector-empty">Select a file to inspect its metadata.</div>';
        return;
    }

    inspectorTitle.textContent = file.name;
    inspectorTitle.title = file.path;

    if (file.status === 'unsupported') {
        inspectorBody.innerHTML = '<div class="inspector-empty">This file type is not supported for metadata inspection.</div>';
        return;
    }
    if (file.status === 'error') {
        inspectorBody.innerHTML = `<div class="inspector-empty">${escapeHtml(file.error || 'Could not read this file.')}</div>`;
        return;
    }
    if (file.metadata.length === 0) {
        inspectorBody.innerHTML = '<div class="inspector-empty">No metadata found in the current scan.</div>';
        if (file.cleanError) {
            const warning = document.createElement('p');
            warning.className = 'warning';
            warning.textContent = file.cleanError;
            inspectorBody.prepend(warning);
        }
        return;
    }

    const query = metadataSearch.value.trim().toLowerCase();
    const groups = new Map<Category, MetadataEntry[]>();
    for (const m of file.metadata) {
        if (query && !m.tag.toLowerCase().includes(query) && !m.value.toLowerCase().includes(query) && !m.label.toLowerCase().includes(query)) {
            continue;
        }
        if (!groups.has(m.category)) groups.set(m.category, []);
        groups.get(m.category)!.push(m);
    }

    inspectorBody.innerHTML = '';
    if (file.cleanError) {
        const warning = document.createElement('p');
        warning.className = 'warning';
        warning.textContent = file.cleanError;
        inspectorBody.appendChild(warning);
    }
    if (groups.size === 0) {
        inspectorBody.insertAdjacentHTML('beforeend', '<div class="inspector-empty">No metadata matches your filter.</div>');
        return;
    }

    const order: Category[] = ['gps', 'author', 'camera', 'software', 'company', 'comments', 'timestamps', 'other'];
    for (const cat of order) {
        const entries = groups.get(cat);
        if (!entries) continue;

        const groupEl = document.createElement('div');
        groupEl.className = 'meta-group';

        const title = document.createElement('div');
        title.className = 'meta-group-title' + (entries.some((e) => e.sensitive) ? ' sensitive' : '');
        title.textContent = `${CATEGORY_LABELS[cat]} (${entries.length})`;
        groupEl.appendChild(title);

        for (const m of entries) {
            const row = document.createElement('div');
            row.className = 'meta-row';

            const tagEl = document.createElement('div');
            tagEl.className = 'meta-tag';
            tagEl.textContent = `${m.group}:${m.tag}`;

            const valueEl = document.createElement('div');
            valueEl.className = 'meta-value' + (m.sensitive ? ' sensitive' : '');
            valueEl.textContent = m.value;

            const copyBtn = document.createElement('button');
            copyBtn.className = 'meta-copy';
            copyBtn.textContent = '⧉';
            copyBtn.title = 'Copy value';
            copyBtn.addEventListener('click', () => {
                void navigator.clipboard.writeText(m.value);
                setStatus(`Copied ${m.tag} value to clipboard.`);
            });

            row.append(tagEl, valueEl, copyBtn);
            groupEl.appendChild(row);
        }

        inspectorBody.appendChild(groupEl);
    }
}

function escapeHtml(s: string): string {
    const div = document.createElement('div');
    div.textContent = s;
    return div.innerHTML;
}

// --- Adding / removing files --------------------------------------------

function mergeNewFiles(added: unknown): void {
    const list = (added as FileEntry[] | null | undefined) ?? [];
    if (list.length === 0) return;
    files = files.concat(list);
    renderQueue();
}

btnAddFiles.addEventListener('click', async () => {
    try {
        const added = await AddFilesDialog();
        mergeNewFiles(added);
        if (added && added.length) setStatus(`Added ${added.length} file(s).`);
    } catch (err) {
        setStatus(`Could not add files: ${String(err)}`);
    }
});

btnAddFolder.addEventListener('click', async () => {
    try {
        const added = await AddFolderDialog();
        mergeNewFiles(added);
        if (added && added.length) setStatus(`Added ${added.length} file(s) from folder.`);
    } catch (err) {
        setStatus(`Could not add folder: ${String(err)}`);
    }
});

btnRemoveSelected.addEventListener('click', async () => {
    const ids = Array.from(selected);
    await RemoveFiles(ids);
    files = files.filter((f) => !selected.has(f.id));
    if (inspectedId && selected.has(inspectedId)) inspectedId = null;
    selected.clear();
    renderQueue();
    renderInspector();
});

btnClear.addEventListener('click', async () => {
    await ClearFiles();
    files = [];
    selected.clear();
    inspectedId = null;
    renderQueue();
    renderInspector();
    setStatus('Queue cleared.');
});

selectAllBox.addEventListener('change', () => {
    selected = selectAllBox.checked ? new Set(files.map((f) => f.id)) : new Set();
    renderQueue();
});

metadataSearch.addEventListener('input', renderInspector);

// --- Profile / custom categories -----------------------------------------

function syncCustomCategoriesVisibility(): void {
    customCategoriesBar.classList.toggle('hidden', profileSelect.value !== 'custom');
}

function buildCustomCategoriesBar(): void {
    customCategoriesBar.innerHTML = '';
    for (const cat of CUSTOM_CATEGORIES) {
        const label = document.createElement('label');
        const input = document.createElement('input');
        input.type = 'checkbox';
        input.dataset.category = cat;
        label.appendChild(input);
        label.append(` ${CATEGORY_LABELS[cat]}`);
        customCategoriesBar.appendChild(label);
    }
}

profileSelect.addEventListener('change', syncCustomCategoriesVisibility);

function selectedCustomCategories(): Category[] {
    const out: Category[] = [];
    customCategoriesBar.querySelectorAll('input[type=checkbox]').forEach((el) => {
        const input = el as HTMLInputElement;
        if (input.checked) out.push(input.dataset.category as Category);
    });
    return out;
}

// --- Output folder ---------------------------------------------------------

btnOutputFolder.addEventListener('click', async () => {
    const dir = await SelectOutputFolder();
    if (!dir) return;
    outputFolder = dir;
    settings.lastOutputFolder = dir;
    await SaveSettings(settings);
    setStatus(`Output folder set to ${dir}`);
    btnOutputFolder.title = dir;
    btnOpenOutputFolder.disabled = false;
});

btnOpenOutputFolder.addEventListener('click', async () => {
    if (!outputFolder) return;
    try {
        await OpenOutputFolder(outputFolder);
    } catch (err) {
        setStatus(`Could not open folder: ${String(err)}`);
    }
});

// --- Cleaning --------------------------------------------------------------

function setCleaning(active: boolean): void {
    cleaning = active;
    btnCancel.classList.toggle('hidden', !active);
    btnAddFiles.disabled = active;
    btnAddFolder.disabled = active;
    updateToolbarState();
}

async function runClean(replaceOriginal: boolean): Promise<void> {
    if (selected.size === 0) return;

    if (!replaceOriginal && !outputFolder) {
        const dir = await SelectOutputFolder();
        if (!dir) {
            setStatus('Clean cancelled: no output folder selected.');
            return;
        }
        outputFolder = dir;
    }

    const profile = profileSelect.value as Profile;
    setCleaning(true);
    setProgress(0);
    setStatus('Cleaning…');

    try {
        await StartClean({
            fileIds: Array.from(selected),
            profile,
            customCategories: profile === 'custom' ? selectedCustomCategories() : undefined,
            outputFolder,
            suffix: settings.suffix || '_clean',
            replaceOriginal,
            preserveTimestamps: settings.preserveTimestamps,
        });
    } catch (err) {
        setCleaning(false);
        setStatus(`Could not start cleaning: ${String(err)}`);
    }
}

btnClean.addEventListener('click', () => {
    if (settings.replaceOriginal) {
        confirmModal.classList.remove('hidden');
    } else {
        void runClean(false);
    }
});

btnConfirmCancel.addEventListener('click', () => confirmModal.classList.add('hidden'));
btnConfirmProceed.addEventListener('click', () => {
    confirmModal.classList.add('hidden');
    void runClean(true);
});

btnCancel.addEventListener('click', () => void CancelClean());

EventsOn('clean:progress', (progress: CleanProgress) => {
    setProgress((progress.index + 1) / progress.total);
    const file = files.find((f) => f.id === progress.result.fileId);
    files = applyCleanResult(files, progress.result);
    renderQueue();
    renderInspector();
    if (file) {
        setStatus(cleanSummary([progress.result], [file]));
    }
});

EventsOn('clean:done', (results: CleanFileResult[]) => {
    setCleaning(false);
    setProgress(0);
    for (const result of results) files = applyCleanResult(files, result);
    renderQueue();
    renderInspector();
    setStatus(cleanSummary(results, files));
});

EventsOn('files:added', (added: FileEntry[] | null, errMsg: string) => {
    mergeNewFiles(added);
    if (errMsg) setStatus(`Drop error: ${errMsg}`);
    else if (added && added.length) setStatus(`Added ${added.length} file(s) via drag & drop.`);
});

// --- Export ------------------------------------------------------------

btnExportCsv.addEventListener('click', async () => {
    const path = await ExportMetadataCSV();
    if (path) setStatus(`Metadata report saved to ${path}`);
});

btnExportJson.addEventListener('click', async () => {
    const path = await ExportPrivacyReportJSON();
    if (path) setStatus(`Privacy report saved to ${path}`);
});

// --- Settings modal ----------------------------------------------------

function openSettingsModal(): void {
    setDefaultProfile.value = settings.defaultProfile;
    setSuffix.value = settings.suffix;
    setPreserveTimestamps.checked = settings.preserveTimestamps;
    setReplaceOriginal.checked = settings.replaceOriginal;
    replaceWarning.classList.toggle('hidden', !setReplaceOriginal.checked);
    settingsModal.classList.remove('hidden');
}

btnSettings.addEventListener('click', openSettingsModal);
btnSettingsCancel.addEventListener('click', () => settingsModal.classList.add('hidden'));
setReplaceOriginal.addEventListener('change', () => {
    replaceWarning.classList.toggle('hidden', !setReplaceOriginal.checked);
});

btnSettingsSave.addEventListener('click', async () => {
    settings = {
        ...settings,
        defaultProfile: setDefaultProfile.value as Profile,
        suffix: setSuffix.value.trim() || '_clean',
        preserveTimestamps: setPreserveTimestamps.checked,
        replaceOriginal: setReplaceOriginal.checked,
    };
    await SaveSettings(settings);
    profileSelect.value = settings.defaultProfile;
    syncCustomCategoriesVisibility();
    settingsModal.classList.add('hidden');
    setStatus('Settings saved.');
});

// --- Keyboard shortcuts --------------------------------------------------

document.addEventListener('keydown', (e) => {
    if (e.key === 'Delete' && selected.size > 0 && !cleaning) {
        void btnRemoveSelected.click();
    }
});

// --- Startup ---------------------------------------------------------------

async function init(): Promise<void> {
    buildCustomCategoriesBar();

    settings = (await GetSettings()) as unknown as Settings;
    profileSelect.value = settings.defaultProfile;
    outputFolder = settings.lastOutputFolder || '';
    if (outputFolder) {
        btnOutputFolder.title = outputFolder;
        btnOpenOutputFolder.disabled = false;
    }
    syncCustomCategoriesVisibility();

    const status: EngineStatus = await GetEngineStatus();
    if (status.ready) {
        engineStatusEl.textContent = `ExifTool ${status.version}`;
        engineStatusEl.className = 'engine-status ok';
    } else {
        engineStatusEl.textContent = 'ExifTool unavailable';
        engineStatusEl.className = 'engine-status error';
        setStatus(`ExifTool engine not found: ${status.error}. Inspection and cleaning are disabled.`);
    }

    const existing = (await GetFiles()) as unknown as FileEntry[] | null;
    if (existing && existing.length) {
        files = existing;
        renderQueue();
    }

    renderQueue();
    renderInspector();
}

void init();
