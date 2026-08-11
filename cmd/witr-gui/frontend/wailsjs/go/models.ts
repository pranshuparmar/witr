export namespace gui {
	
	export class GUIProcessItem {
	    PID: number;
	    PPID: number;
	    Command: string;
	    Cmdline: string;
	    Exe: string;
	    // Go type: time
	    StartedAt: any;
	    User: string;
	    CPUPercent: number;
	    MemoryRSS: number;
	    MemoryPercent: number;
	    WorkingDir: string;
	    GitRepo: string;
	    GitBranch: string;
	    Container: string;
	    Service: string;
	    Sockets: model.Socket[];
	    Health: string;
	    Forked: string;
	    Env: string[];
	    ExeDeleted: boolean;
	    cpuFormatted: string;
	    memFormatted: string;
	    memPercentStr: string;
	    startedStr: string;
	    isSystem: boolean;
	    hasSockets: boolean;
	
	    static createFrom(source: any = {}) {
	        return new GUIProcessItem(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.PID = source["PID"];
	        this.PPID = source["PPID"];
	        this.Command = source["Command"];
	        this.Cmdline = source["Cmdline"];
	        this.Exe = source["Exe"];
	        this.StartedAt = this.convertValues(source["StartedAt"], null);
	        this.User = source["User"];
	        this.CPUPercent = source["CPUPercent"];
	        this.MemoryRSS = source["MemoryRSS"];
	        this.MemoryPercent = source["MemoryPercent"];
	        this.WorkingDir = source["WorkingDir"];
	        this.GitRepo = source["GitRepo"];
	        this.GitBranch = source["GitBranch"];
	        this.Container = source["Container"];
	        this.Service = source["Service"];
	        this.Sockets = this.convertValues(source["Sockets"], model.Socket);
	        this.Health = source["Health"];
	        this.Forked = source["Forked"];
	        this.Env = source["Env"];
	        this.ExeDeleted = source["ExeDeleted"];
	        this.cpuFormatted = source["cpuFormatted"];
	        this.memFormatted = source["memFormatted"];
	        this.memPercentStr = source["memPercentStr"];
	        this.startedStr = source["startedStr"];
	        this.isSystem = source["isSystem"];
	        this.hasSockets = source["hasSockets"];
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
	export class GUIResult {
	    // Go type: model
	    Target: any;
	    ResolvedTarget: string;
	    // Go type: model
	    Process: any;
	    RestartCount: number;
	    Ancestry: model.Process[];
	    // Go type: model
	    Source: any;
	    Warnings: string[];
	    // Go type: model
	    SocketInfo?: any;
	    // Go type: model
	    ResourceContext?: any;
	    // Go type: model
	    FileContext?: any;
	    whyExplanation: string;
	    startedFormatted: string;
	    workingDir: string;
	    socketsList: string[];
	    cpuFormatted: string;
	    memoryVirtual: string;
	    memoryResident: string;
	    memoryPrivate: string;
	    ioRead: string;
	    ioWrite: string;
	    handlesCount: string;
	    threadCount: string;
	    envVars: Record<string, string>;
	
	    static createFrom(source: any = {}) {
	        return new GUIResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.Target = this.convertValues(source["Target"], null);
	        this.ResolvedTarget = source["ResolvedTarget"];
	        this.Process = this.convertValues(source["Process"], null);
	        this.RestartCount = source["RestartCount"];
	        this.Ancestry = this.convertValues(source["Ancestry"], model.Process);
	        this.Source = this.convertValues(source["Source"], null);
	        this.Warnings = source["Warnings"];
	        this.SocketInfo = this.convertValues(source["SocketInfo"], null);
	        this.ResourceContext = this.convertValues(source["ResourceContext"], null);
	        this.FileContext = this.convertValues(source["FileContext"], null);
	        this.whyExplanation = source["whyExplanation"];
	        this.startedFormatted = source["startedFormatted"];
	        this.workingDir = source["workingDir"];
	        this.socketsList = source["socketsList"];
	        this.cpuFormatted = source["cpuFormatted"];
	        this.memoryVirtual = source["memoryVirtual"];
	        this.memoryResident = source["memoryResident"];
	        this.memoryPrivate = source["memoryPrivate"];
	        this.ioRead = source["ioRead"];
	        this.ioWrite = source["ioWrite"];
	        this.handlesCount = source["handlesCount"];
	        this.threadCount = source["threadCount"];
	        this.envVars = source["envVars"];
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
	export class SystemAnalytics {
	    totalProcesses: number;
	    listeningPorts: number;
	    systemProcesses: number;
	    userProcesses: number;
	
	    static createFrom(source: any = {}) {
	        return new SystemAnalytics(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.totalProcesses = source["totalProcesses"];
	        this.listeningPorts = source["listeningPorts"];
	        this.systemProcesses = source["systemProcesses"];
	        this.userProcesses = source["userProcesses"];
	    }
	}

}

