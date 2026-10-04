export namespace main {
	
	export class EngineStatus {
	    ready: boolean;
	    path?: string;
	    version?: string;
	    error?: string;
	
	    static createFrom(source: any = {}) {
	        return new EngineStatus(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.ready = source["ready"];
	        this.path = source["path"];
	        this.version = source["version"];
	        this.error = source["error"];
	    }
	}

}

export namespace model {
	
	export class CleanRequest {
	    fileIds: string[];
	    profile: string;
	    customCategories?: string[];
	    outputFolder: string;
	    suffix: string;
	    replaceOriginal: boolean;
	    preserveTimestamps: boolean;
	
	    static createFrom(source: any = {}) {
	        return new CleanRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.fileIds = source["fileIds"];
	        this.profile = source["profile"];
	        this.customCategories = source["customCategories"];
	        this.outputFolder = source["outputFolder"];
	        this.suffix = source["suffix"];
	        this.replaceOriginal = source["replaceOriginal"];
	        this.preserveTimestamps = source["preserveTimestamps"];
	    }
	}
	export class MetadataEntry {
	    group: string;
	    tag: string;
	    label: string;
	    value: string;
	    category: string;
	    sensitive: boolean;
	    removable: boolean;
	
	    static createFrom(source: any = {}) {
	        return new MetadataEntry(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.group = source["group"];
	        this.tag = source["tag"];
	        this.label = source["label"];
	        this.value = source["value"];
	        this.category = source["category"];
	        this.sensitive = source["sensitive"];
	        this.removable = source["removable"];
	    }
	}
	export class FileEntry {
	    id: string;
	    path: string;
	    name: string;
	    ext: string;
	    size: number;
	    status: string;
	    error?: string;
	    writable: boolean;
	    metadata: MetadataEntry[];
	    sensitiveCategories: string[];
	    cleanedPath?: string;
	
	    static createFrom(source: any = {}) {
	        return new FileEntry(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.path = source["path"];
	        this.name = source["name"];
	        this.ext = source["ext"];
	        this.size = source["size"];
	        this.status = source["status"];
	        this.error = source["error"];
	        this.writable = source["writable"];
	        this.metadata = this.convertValues(source["metadata"], MetadataEntry);
	        this.sensitiveCategories = source["sensitiveCategories"];
	        this.cleanedPath = source["cleanedPath"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	
	export class Settings {
	    defaultProfile: string;
	    outputFolder: string;
	    suffix: string;
	    replaceOriginal: boolean;
	    preserveTimestamps: boolean;
	    lastOutputFolder: string;
	    theme: string;
	
	    static createFrom(source: any = {}) {
	        return new Settings(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.defaultProfile = source["defaultProfile"];
	        this.outputFolder = source["outputFolder"];
	        this.suffix = source["suffix"];
	        this.replaceOriginal = source["replaceOriginal"];
	        this.preserveTimestamps = source["preserveTimestamps"];
	        this.lastOutputFolder = source["lastOutputFolder"];
	        this.theme = source["theme"];
	    }
	}

}

