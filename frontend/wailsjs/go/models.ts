export namespace app {
	
	export class VersionInfo {
	    currentVersion: string;
	    latestVersion: string;
	    hasUpdate: boolean;
	    updateUrl: string;
	    releaseNotes: string;
	
	    static createFrom(source: any = {}) {
	        return new VersionInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.currentVersion = source["currentVersion"];
	        this.latestVersion = source["latestVersion"];
	        this.hasUpdate = source["hasUpdate"];
	        this.updateUrl = source["updateUrl"];
	        this.releaseNotes = source["releaseNotes"];
	    }
	}

}

export namespace frontend {
	
	export class FileFilter {
	    DisplayName: string;
	    Pattern: string;
	
	    static createFrom(source: any = {}) {
	        return new FileFilter(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.DisplayName = source["DisplayName"];
	        this.Pattern = source["Pattern"];
	    }
	}

}

export namespace models {
	
	export class Account {
	    name: string;
	    fakeid: string;
	    alias: string;
	    signature: string;
	    avatar: string;
	    qrCode: string;
	    serviceType: number;
	
	    static createFrom(source: any = {}) {
	        return new Account(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.fakeid = source["fakeid"];
	        this.alias = source["alias"];
	        this.signature = source["signature"];
	        this.avatar = source["avatar"];
	        this.qrCode = source["qrCode"];
	        this.serviceType = source["serviceType"];
	    }
	}
	export class AccountActivity {
	    accountName: string;
	    articleCount: number;
	    avgLength: number;
	    publishFreq: number;
	    lastPublishAt: string;
	    activityScore: number;
	
	    static createFrom(source: any = {}) {
	        return new AccountActivity(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.accountName = source["accountName"];
	        this.articleCount = source["articleCount"];
	        this.avgLength = source["avgLength"];
	        this.publishFreq = source["publishFreq"];
	        this.lastPublishAt = source["lastPublishAt"];
	        this.activityScore = source["activityScore"];
	    }
	}
	export class TimeDistribution {
	    date: string;
	    count: number;
	
	    static createFrom(source: any = {}) {
	        return new TimeDistribution(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.date = source["date"];
	        this.count = source["count"];
	    }
	}
	export class AccountTimeDistribution {
	    accountName: string;
	    data: TimeDistribution[];
	
	    static createFrom(source: any = {}) {
	        return new AccountTimeDistribution(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.accountName = source["accountName"];
	        this.data = this.convertValues(source["data"], TimeDistribution);
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
	export class LengthDistribution {
	    range: string;
	    count: number;
	
	    static createFrom(source: any = {}) {
	        return new LengthDistribution(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.range = source["range"];
	        this.count = source["count"];
	    }
	}
	export class KeywordFrequency {
	    word: string;
	    count: number;
	
	    static createFrom(source: any = {}) {
	        return new KeywordFrequency(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.word = source["word"];
	        this.count = source["count"];
	    }
	}
	export class AnalyticsData {
	    timeDistribution: AccountTimeDistribution[];
	    topKeywords: KeywordFrequency[];
	    lengthDistribution: LengthDistribution[];
	    accountRanking: AccountActivity[];
	    cachedAt: string;
	
	    static createFrom(source: any = {}) {
	        return new AnalyticsData(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.timeDistribution = this.convertValues(source["timeDistribution"], AccountTimeDistribution);
	        this.topKeywords = this.convertValues(source["topKeywords"], KeywordFrequency);
	        this.lengthDistribution = this.convertValues(source["lengthDistribution"], LengthDistribution);
	        this.accountRanking = this.convertValues(source["accountRanking"], AccountActivity);
	        this.cachedAt = source["cachedAt"];
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
	export class AppData {
	    totalArticles: number;
	    totalAccounts: number;
	    accountNames: string[];
	    lastUpdateTime: string;
	    totalImages: number;
	    lastScrapeTime: string;
	    totalExports: number;
	    todayArticles: number;
	    lastScrapeDate: string;
	
	    static createFrom(source: any = {}) {
	        return new AppData(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.totalArticles = source["totalArticles"];
	        this.totalAccounts = source["totalAccounts"];
	        this.accountNames = source["accountNames"];
	        this.lastUpdateTime = source["lastUpdateTime"];
	        this.totalImages = source["totalImages"];
	        this.lastScrapeTime = source["lastScrapeTime"];
	        this.totalExports = source["totalExports"];
	        this.todayArticles = source["todayArticles"];
	        this.lastScrapeDate = source["lastScrapeDate"];
	    }
	}
	export class Article {
	    id: string;
	    accountName: string;
	    accountFakeid: string;
	    title: string;
	    link: string;
	    digest: string;
	    content: string;
	    publishTime: string;
	    publishTimestamp: number;
	    // Go type: time
	    createdAt: any;
	
	    static createFrom(source: any = {}) {
	        return new Article(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.accountName = source["accountName"];
	        this.accountFakeid = source["accountFakeid"];
	        this.title = source["title"];
	        this.link = source["link"];
	        this.digest = source["digest"];
	        this.content = source["content"];
	        this.publishTime = source["publishTime"];
	        this.publishTimestamp = source["publishTimestamp"];
	        this.createdAt = this.convertValues(source["createdAt"], null);
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
	export class Config {
	    maxPages: number;
	    requestInterval: number;
	    maxWorkers: number;
	    includeContent: boolean;
	    cacheExpireHours: number;
	    outputDir: string;
	
	    static createFrom(source: any = {}) {
	        return new Config(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.maxPages = source["maxPages"];
	        this.requestInterval = source["requestInterval"];
	        this.maxWorkers = source["maxWorkers"];
	        this.includeContent = source["includeContent"];
	        this.cacheExpireHours = source["cacheExpireHours"];
	        this.outputDir = source["outputDir"];
	    }
	}
	export class CronValidationResult {
	    valid: boolean;
	    nextTime: string;
	    error: string;
	
	    static createFrom(source: any = {}) {
	        return new CronValidationResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.valid = source["valid"];
	        this.nextTime = source["nextTime"];
	        this.error = source["error"];
	    }
	}
	export class DataFileInfo {
	    filename: string;
	    filePath: string;
	    saveTime: string;
	    totalCount: number;
	    accounts: string[];
	    fileSize: number;
	
	    static createFrom(source: any = {}) {
	        return new DataFileInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.filename = source["filename"];
	        this.filePath = source["filePath"];
	        this.saveTime = source["saveTime"];
	        this.totalCount = source["totalCount"];
	        this.accounts = source["accounts"];
	        this.fileSize = source["fileSize"];
	    }
	}
	
	
	export class LoginStatus {
	    isLoggedIn: boolean;
	    // Go type: time
	    loginTime?: any;
	    // Go type: time
	    expireTime?: any;
	    hoursSinceLogin?: number;
	    hoursUntilExpire?: number;
	    token?: string;
	    message: string;
	
	    static createFrom(source: any = {}) {
	        return new LoginStatus(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.isLoggedIn = source["isLoggedIn"];
	        this.loginTime = this.convertValues(source["loginTime"], null);
	        this.expireTime = this.convertValues(source["expireTime"], null);
	        this.hoursSinceLogin = source["hoursSinceLogin"];
	        this.hoursUntilExpire = source["hoursUntilExpire"];
	        this.token = source["token"];
	        this.message = source["message"];
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
	export class ScheduledTask {
	    ID: number;
	    name: string;
	    description: string;
	    cronExpression: string;
	    enabled: boolean;
	    scrapeConfig: string;
	    // Go type: time
	    lastRunTime?: any;
	    // Go type: time
	    nextRunTime?: any;
	    lastRunStatus: string;
	    lastRunError: string;
	    totalRuns: number;
	    successRuns: number;
	    failedRuns: number;
	    // Go type: time
	    createdAt: any;
	    // Go type: time
	    updatedAt: any;
	
	    static createFrom(source: any = {}) {
	        return new ScheduledTask(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.ID = source["ID"];
	        this.name = source["name"];
	        this.description = source["description"];
	        this.cronExpression = source["cronExpression"];
	        this.enabled = source["enabled"];
	        this.scrapeConfig = source["scrapeConfig"];
	        this.lastRunTime = this.convertValues(source["lastRunTime"], null);
	        this.nextRunTime = this.convertValues(source["nextRunTime"], null);
	        this.lastRunStatus = source["lastRunStatus"];
	        this.lastRunError = source["lastRunError"];
	        this.totalRuns = source["totalRuns"];
	        this.successRuns = source["successRuns"];
	        this.failedRuns = source["failedRuns"];
	        this.createdAt = this.convertValues(source["createdAt"], null);
	        this.updatedAt = this.convertValues(source["updatedAt"], null);
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
	export class ScrapeConfig {
	    accounts: string[];
	    startDate: string;
	    endDate: string;
	    recentDays: number;
	    maxPages: number;
	    requestInterval: number;
	    includeContent: boolean;
	    keywordFilter: string;
	    maxWorkers: number;
	
	    static createFrom(source: any = {}) {
	        return new ScrapeConfig(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.accounts = source["accounts"];
	        this.startDate = source["startDate"];
	        this.endDate = source["endDate"];
	        this.recentDays = source["recentDays"];
	        this.maxPages = source["maxPages"];
	        this.requestInterval = source["requestInterval"];
	        this.includeContent = source["includeContent"];
	        this.keywordFilter = source["keywordFilter"];
	        this.maxWorkers = source["maxWorkers"];
	    }
	}
	export class TaskExecutionLog {
	    ID: number;
	    taskId: number;
	    taskName: string;
	    // Go type: time
	    startTime: any;
	    // Go type: time
	    endTime?: any;
	    duration: number;
	    status: string;
	    articlesCount: number;
	    errorMessage: string;
	    triggerType: string;
	    // Go type: time
	    createdAt: any;
	
	    static createFrom(source: any = {}) {
	        return new TaskExecutionLog(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.ID = source["ID"];
	        this.taskId = source["taskId"];
	        this.taskName = source["taskName"];
	        this.startTime = this.convertValues(source["startTime"], null);
	        this.endTime = this.convertValues(source["endTime"], null);
	        this.duration = source["duration"];
	        this.status = source["status"];
	        this.articlesCount = source["articlesCount"];
	        this.errorMessage = source["errorMessage"];
	        this.triggerType = source["triggerType"];
	        this.createdAt = this.convertValues(source["createdAt"], null);
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

export namespace spider {
	
	export class ImageInfo {
	    url: string;
	    index: number;
	    filename: string;
	    articleTitle: string;
	    accountName: string;
	
	    static createFrom(source: any = {}) {
	        return new ImageInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.url = source["url"];
	        this.index = source["index"];
	        this.filename = source["filename"];
	        this.articleTitle = source["articleTitle"];
	        this.accountName = source["accountName"];
	    }
	}

}

