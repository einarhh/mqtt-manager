export namespace plugins {
	
	export class Plugin {
	    id: string;
	    name: string;
	    filename: string;
	    enabled: boolean;
	    order: number;
	    source: string;
	
	    static createFrom(source: any = {}) {
	        return new Plugin(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.filename = source["filename"];
	        this.enabled = source["enabled"];
	        this.order = source["order"];
	        this.source = source["source"];
	    }
	}

}

export namespace profiles {
	
	export class Subscription {
	    filter: string;
	    qos: number;
	
	    static createFrom(source: any = {}) {
	        return new Subscription(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.filter = source["filter"];
	        this.qos = source["qos"];
	    }
	}
	export class ConnectionProfile {
	    id: string;
	    name: string;
	    host: string;
	    port: number;
	    useTls: boolean;
	    tlsInsecure: boolean;
	    caCertPath: string;
	    clientId: string;
	    username: string;
	    password: string;
	    keepAlive: number;
	    subscriptions: Subscription[];
	    subFilter?: string;
	    subQos?: number;
	
	    static createFrom(source: any = {}) {
	        return new ConnectionProfile(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.host = source["host"];
	        this.port = source["port"];
	        this.useTls = source["useTls"];
	        this.tlsInsecure = source["tlsInsecure"];
	        this.caCertPath = source["caCertPath"];
	        this.clientId = source["clientId"];
	        this.username = source["username"];
	        this.password = source["password"];
	        this.keepAlive = source["keepAlive"];
	        this.subscriptions = this.convertValues(source["subscriptions"], Subscription);
	        this.subFilter = source["subFilter"];
	        this.subQos = source["subQos"];
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

