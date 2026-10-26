export namespace application {
	
	export class DoctorReport {
	    issues: string[];
	
	    static createFrom(source: any = {}) {
	        return new DoctorReport(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.issues = source["issues"];
	    }
	}

}

export namespace domain {
	
	export class AppSettings {
	    schema_version: number;
	    default_model_id: string;
	    language: string;
	    output_dir: string;
	    export_formats: string[];
	    initial_prompt: string;
	
	    static createFrom(source: any = {}) {
	        return new AppSettings(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.schema_version = source["schema_version"];
	        this.default_model_id = source["default_model_id"];
	        this.language = source["language"];
	        this.output_dir = source["output_dir"];
	        this.export_formats = source["export_formats"];
	        this.initial_prompt = source["initial_prompt"];
	    }
	}
	export class JobProgress {
	    job_id: string;
	    state: string;
	    elapsed_ms: number;
	    message: string;
	    percent?: number;
	
	    static createFrom(source: any = {}) {
	        return new JobProgress(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.job_id = source["job_id"];
	        this.state = source["state"];
	        this.elapsed_ms = source["elapsed_ms"];
	        this.message = source["message"];
	        this.percent = source["percent"];
	    }
	}
	export class ModelInfo {
	    id: string;
	    file_name: string;
	    size_bytes: number;
	    path?: string;
	    sha256?: string;
	    status: string;
	
	    static createFrom(source: any = {}) {
	        return new ModelInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.file_name = source["file_name"];
	        this.size_bytes = source["size_bytes"];
	        this.path = source["path"];
	        this.sha256 = source["sha256"];
	        this.status = source["status"];
	    }
	}
	export class Segment {
	    start_ms: number;
	    end_ms: number;
	    text: string;
	    edited_text?: string;
	
	    static createFrom(source: any = {}) {
	        return new Segment(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.start_ms = source["start_ms"];
	        this.end_ms = source["end_ms"];
	        this.text = source["text"];
	        this.edited_text = source["edited_text"];
	    }
	}
	export class SegmentEdit {
	    index: number;
	    edited_text: string;
	
	    static createFrom(source: any = {}) {
	        return new SegmentEdit(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.index = source["index"];
	        this.edited_text = source["edited_text"];
	    }
	}
	export class Transcript {
	    schema_version: number;
	    id: string;
	    job_id: string;
	    model_id: string;
	    language: string;
	    source_path: string;
	    duration_ms: number;
	    // Go type: time
	    created_at: any;
	    segments: Segment[];
	
	    static createFrom(source: any = {}) {
	        return new Transcript(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.schema_version = source["schema_version"];
	        this.id = source["id"];
	        this.job_id = source["job_id"];
	        this.model_id = source["model_id"];
	        this.language = source["language"];
	        this.source_path = source["source_path"];
	        this.duration_ms = source["duration_ms"];
	        this.created_at = this.convertValues(source["created_at"], null);
	        this.segments = this.convertValues(source["segments"], Segment);
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
	export class UserError {
	    message: string;
	    detail?: string;
	
	    static createFrom(source: any = {}) {
	        return new UserError(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.message = source["message"];
	        this.detail = source["detail"];
	    }
	}
	export class TranscriptionRequest {
	    source_path: string;
	    source_kind?: string;
	    model_id: string;
	    language: string;
	    output_dir: string;
	    initial_prompt: string;
	    export_formats: string[];
	
	    static createFrom(source: any = {}) {
	        return new TranscriptionRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.source_path = source["source_path"];
	        this.source_kind = source["source_kind"];
	        this.model_id = source["model_id"];
	        this.language = source["language"];
	        this.output_dir = source["output_dir"];
	        this.initial_prompt = source["initial_prompt"];
	        this.export_formats = source["export_formats"];
	    }
	}
	export class TranscriptionJob {
	    schema_version: number;
	    id: string;
	    state: string;
	    request: TranscriptionRequest;
	    // Go type: time
	    started_at: any;
	    // Go type: time
	    ended_at: any;
	    error?: UserError;
	    transcript_id?: string;
	    result_dir?: string;
	    progress: JobProgress;
	
	    static createFrom(source: any = {}) {
	        return new TranscriptionJob(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.schema_version = source["schema_version"];
	        this.id = source["id"];
	        this.state = source["state"];
	        this.request = this.convertValues(source["request"], TranscriptionRequest);
	        this.started_at = this.convertValues(source["started_at"], null);
	        this.ended_at = this.convertValues(source["ended_at"], null);
	        this.error = this.convertValues(source["error"], UserError);
	        this.transcript_id = source["transcript_id"];
	        this.result_dir = source["result_dir"];
	        this.progress = this.convertValues(source["progress"], JobProgress);
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
	

}

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

