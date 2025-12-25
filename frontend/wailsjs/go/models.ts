export namespace api {
	
	export class UserInfo {
	    id: number;
	    username: string;
	    email: string;
	    firstName: string;
	    lastName: string;
	    roleCode: string;
	
	    static createFrom(source: any = {}) {
	        return new UserInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.username = source["username"];
	        this.email = source["email"];
	        this.firstName = source["firstName"];
	        this.lastName = source["lastName"];
	        this.roleCode = source["roleCode"];
	    }
	}
	export class AuthResponse {
	    accessToken: string;
	    refreshToken: string;
	    tokenType: string;
	    expiresIn: number;
	    expiresAt: string;
	    user: UserInfo;
	
	    static createFrom(source: any = {}) {
	        return new AuthResponse(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.accessToken = source["accessToken"];
	        this.refreshToken = source["refreshToken"];
	        this.tokenType = source["tokenType"];
	        this.expiresIn = source["expiresIn"];
	        this.expiresAt = source["expiresAt"];
	        this.user = this.convertValues(source["user"], UserInfo);
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
	export class ConnectionStatus {
	    bigoRoomId: string;
	    streamerId: string;
	    status: string;
	    messagesReceived: number;
	    lastMessageTime: number;
	    errorMessage?: string;
	
	    static createFrom(source: any = {}) {
	        return new ConnectionStatus(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.bigoRoomId = source["bigoRoomId"];
	        this.streamerId = source["streamerId"];
	        this.status = source["status"];
	        this.messagesReceived = source["messagesReceived"];
	        this.lastMessageTime = source["lastMessageTime"];
	        this.errorMessage = source["errorMessage"];
	    }
	}

}

export namespace session {
	
	export class Status {
	    roomId: string;
	    sessionId: string;
	    isActive: boolean;
	    connections: api.ConnectionStatus[];
	    startTime: number;
	    deviceHash: string;
	
	    static createFrom(source: any = {}) {
	        return new Status(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.roomId = source["roomId"];
	        this.sessionId = source["sessionId"];
	        this.isActive = source["isActive"];
	        this.connections = this.convertValues(source["connections"], api.ConnectionStatus);
	        this.startTime = source["startTime"];
	        this.deviceHash = source["deviceHash"];
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

