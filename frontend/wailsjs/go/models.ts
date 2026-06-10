export namespace profiles {
	
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
	    subFilter: string;
	    subQos: number;
	
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
	        this.subFilter = source["subFilter"];
	        this.subQos = source["subQos"];
	    }
	}

}

