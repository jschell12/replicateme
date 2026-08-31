export namespace config {
	
	export class RAGConfig {
	    enabled: boolean;
	    qdrantUrl?: string;
	    ollamaUrl?: string;
	    embedModel?: string;
	
	    static createFrom(source: any = {}) {
	        return new RAGConfig(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.enabled = source["enabled"];
	        this.qdrantUrl = source["qdrantUrl"];
	        this.ollamaUrl = source["ollamaUrl"];
	        this.embedModel = source["embedModel"];
	    }
	}
	export class QuirkToggles {
	    misspellings?: boolean;
	    grammarErrors?: boolean;
	    missingApostrophes?: boolean;
	    lowercaseI?: boolean;
	    skipPunctuation?: boolean;
	    doubleSpaces?: boolean;
	    fragments?: boolean;
	
	    static createFrom(source: any = {}) {
	        return new QuirkToggles(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.misspellings = source["misspellings"];
	        this.grammarErrors = source["grammarErrors"];
	        this.missingApostrophes = source["missingApostrophes"];
	        this.lowercaseI = source["lowercaseI"];
	        this.skipPunctuation = source["skipPunctuation"];
	        this.doubleSpaces = source["doubleSpaces"];
	        this.fragments = source["fragments"];
	    }
	}
	export class Config {
	    provider: string;
	    model?: string;
	    baseUrl?: string;
	    quirkLevel: number;
	    defaultPlatform: string;
	    persona?: string;
	    quirks?: QuirkToggles;
	    rag?: RAGConfig;
	
	    static createFrom(source: any = {}) {
	        return new Config(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.provider = source["provider"];
	        this.model = source["model"];
	        this.baseUrl = source["baseUrl"];
	        this.quirkLevel = source["quirkLevel"];
	        this.defaultPlatform = source["defaultPlatform"];
	        this.persona = source["persona"];
	        this.quirks = this.convertValues(source["quirks"], QuirkToggles);
	        this.rag = this.convertValues(source["rag"], RAGConfig);
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

export namespace corpus {
	
	export class ProfileSummary {
	    platform: string;
	    messageCount: number;
	    updatedAt: string;
	
	    static createFrom(source: any = {}) {
	        return new ProfileSummary(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.platform = source["platform"];
	        this.messageCount = source["messageCount"];
	        this.updatedAt = source["updatedAt"];
	    }
	}
	export class PlatformCount {
	    platform: string;
	    count: number;
	
	    static createFrom(source: any = {}) {
	        return new PlatformCount(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.platform = source["platform"];
	        this.count = source["count"];
	    }
	}
	export class CorpusStats {
	    totalMessages: number;
	    byPlatform: PlatformCount[];
	    profiles: ProfileSummary[];
	
	    static createFrom(source: any = {}) {
	        return new CorpusStats(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.totalMessages = source["totalMessages"];
	        this.byPlatform = this.convertValues(source["byPlatform"], PlatformCount);
	        this.profiles = this.convertValues(source["profiles"], ProfileSummary);
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
	
	export class TypicalError {
	    pattern: string;
	    frequency: number;
	    examples: string[];
	
	    static createFrom(source: any = {}) {
	        return new TypicalError(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.pattern = source["pattern"];
	        this.frequency = source["frequency"];
	        this.examples = source["examples"];
	    }
	}
	export class StyleProfile {
	    averageLength: number;
	    capitalizesFirstWord: number;
	    usesContractions: number;
	    usesPeriods: number;
	    usesCommas: number;
	    usesExclamation: number;
	    usesQuestionMark: number;
	    usesEmoji: number;
	    commonPhrases: string[];
	    typicalErrors: TypicalError[];
	    sentenceFragmentRatio: number;
	    lowercaseIRatio: number;
	
	    static createFrom(source: any = {}) {
	        return new StyleProfile(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.averageLength = source["averageLength"];
	        this.capitalizesFirstWord = source["capitalizesFirstWord"];
	        this.usesContractions = source["usesContractions"];
	        this.usesPeriods = source["usesPeriods"];
	        this.usesCommas = source["usesCommas"];
	        this.usesExclamation = source["usesExclamation"];
	        this.usesQuestionMark = source["usesQuestionMark"];
	        this.usesEmoji = source["usesEmoji"];
	        this.commonPhrases = source["commonPhrases"];
	        this.typicalErrors = this.convertValues(source["typicalErrors"], TypicalError);
	        this.sentenceFragmentRatio = source["sentenceFragmentRatio"];
	        this.lowercaseIRatio = source["lowercaseIRatio"];
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
	export class PlatformProfile {
	    Platform: string;
	    Profile: StyleProfile;
	    MessageCount: number;
	
	    static createFrom(source: any = {}) {
	        return new PlatformProfile(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.Platform = source["Platform"];
	        this.Profile = this.convertValues(source["Profile"], StyleProfile);
	        this.MessageCount = source["MessageCount"];
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
	export class ProfileResult {
	    Profile: StyleProfile;
	    MessageCount: number;
	
	    static createFrom(source: any = {}) {
	        return new ProfileResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.Profile = this.convertValues(source["Profile"], StyleProfile);
	        this.MessageCount = source["MessageCount"];
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
	
	export class IngestResult {
	    messageCount: number;
	    newCount: number;
	    profileNote: string;
	
	    static createFrom(source: any = {}) {
	        return new IngestResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.messageCount = source["messageCount"];
	        this.newCount = source["newCount"];
	        this.profileNote = source["profileNote"];
	    }
	}
	export class SourceInfo {
	    name: string;
	    description: string;
	    requiresFile: boolean;
	
	    static createFrom(source: any = {}) {
	        return new SourceInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.description = source["description"];
	        this.requiresFile = source["requiresFile"];
	    }
	}

}

