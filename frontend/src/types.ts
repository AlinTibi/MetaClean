export type FileStatus =
    | 'pending'
    | 'clean'
    | 'metadata_found'
    | 'sensitive_metadata_found'
    | 'unsupported'
    | 'error';

export type Category =
    | 'gps'
    | 'author'
    | 'camera'
    | 'software'
    | 'company'
    | 'comments'
    | 'timestamps'
    | 'other';

export type Profile = 'privacy' | 'all' | 'custom';

export interface MetadataEntry {
    group: string;
    tag: string;
    label: string;
    value: string;
    category: Category;
    sensitive: boolean;
    removable: boolean;
}

export interface FileEntry {
    id: string;
    path: string;
    name: string;
    ext: string;
    size: number;
    status: FileStatus;
    error?: string;
    metadata: MetadataEntry[];
    sensitiveCategories: Category[];
    cleanedPath?: string;
}

export interface CleanRequest {
    fileIds: string[];
    profile: Profile;
    customCategories?: Category[];
    outputFolder: string;
    suffix: string;
    replaceOriginal: boolean;
    preserveTimestamps: boolean;
}

export interface CleanFileResult {
    fileId: string;
    success: boolean;
    error?: string;
    outputPath?: string;
    backupPath?: string;
    beforeCount: number;
    afterCount: number;
    remaining?: MetadataEntry[];
}

export interface CleanProgress {
    index: number;
    total: number;
    result: CleanFileResult;
}

export interface Settings {
    defaultProfile: Profile;
    outputFolder: string;
    suffix: string;
    replaceOriginal: boolean;
    preserveTimestamps: boolean;
    lastOutputFolder: string;
    theme: string;
}

export interface EngineStatus {
    ready: boolean;
    path?: string;
    version?: string;
    error?: string;
}

export const CATEGORY_LABELS: Record<Category, string> = {
    gps: 'GPS / Location',
    author: 'Author / Creator',
    camera: 'Camera / Device',
    software: 'Software / Application',
    company: 'Company / Manager',
    comments: 'Comments / Descriptions',
    timestamps: 'Timestamps',
    other: 'Other',
};

export const STATUS_LABELS: Record<FileStatus, string> = {
    pending: 'Pending',
    clean: 'Clean',
    metadata_found: 'Metadata Found',
    sensitive_metadata_found: 'Sensitive Metadata Found',
    unsupported: 'Unsupported',
    error: 'Error',
};
