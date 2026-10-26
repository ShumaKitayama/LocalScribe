export namespace main {
	
	export class SaveRecordingResult {
	    path: string;
	    size_bytes: number;
	
	    static createFrom(source: any = {}) {
	        return new SaveRecordingResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.path = source["path"];
	        this.size_bytes = source["size_bytes"];
	    }
	}

}

