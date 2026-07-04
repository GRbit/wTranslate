export namespace libretranslate {
	
	export class DetectedLanguage {
	    language: string;
	    confidence: number;
	
	    static createFrom(source: any = {}) {
	        return new DetectedLanguage(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.language = source["language"];
	        this.confidence = source["confidence"];
	    }
	}
	export class Language {
	    code: string;
	    name: string;
	
	    static createFrom(source: any = {}) {
	        return new Language(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.code = source["code"];
	        this.name = source["name"];
	    }
	}
	export class TranslateRequest {
	    q: string;
	    source: string;
	    target: string;
	
	    static createFrom(source: any = {}) {
	        return new TranslateRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.q = source["q"];
	        this.source = source["source"];
	        this.target = source["target"];
	    }
	}
	export class TranslateResponse {
	    translatedText: string;
	    detectedLanguage?: DetectedLanguage;
	
	    static createFrom(source: any = {}) {
	        return new TranslateResponse(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.translatedText = source["translatedText"];
	        this.detectedLanguage = this.convertValues(source["detectedLanguage"], DetectedLanguage);
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

export namespace settings {
	
	export class Settings {
	    baseUrl: string;
	    apiKey: string;
	    liveTranslation: boolean;
	    shortcut: string;
	    defaultToAuto: boolean;
	    lastSourceLang: string;
	    lastTargetLang: string;
	    autoCopy: boolean;
	    debug: boolean;
	
	    static createFrom(source: any = {}) {
	        return new Settings(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.baseUrl = source["baseUrl"];
	        this.apiKey = source["apiKey"];
	        this.liveTranslation = source["liveTranslation"];
	        this.shortcut = source["shortcut"];
	        this.defaultToAuto = source["defaultToAuto"];
	        this.lastSourceLang = source["lastSourceLang"];
	        this.lastTargetLang = source["lastTargetLang"];
	        this.autoCopy = source["autoCopy"];
	        this.debug = source["debug"];
	    }
	}

}

