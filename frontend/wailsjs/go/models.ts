export namespace api {
	
	export class Agency {
	    id: number;
	    name: string;
	    plan: string;
	    status: string;
	    maxRooms: number;
	    currentRooms: number;
	    // Go type: time
	    expiresAt: any;
	
	    static createFrom(source: any = {}) {
	        return new Agency(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.plan = source["plan"];
	        this.status = source["status"];
	        this.maxRooms = source["maxRooms"];
	        this.currentRooms = source["currentRooms"];
	        this.expiresAt = this.convertValues(source["expiresAt"], null);
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
	export class User {
	    id: number;
	    username: string;
	    email: string;
	    firstName?: string;
	    lastName?: string;
	    roleCode: string;
	
	    static createFrom(source: any = {}) {
	        return new User(source);
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
	    // Go type: time
	    expiresAt: any;
	    user: User;
	    agency: Agency;
	
	    static createFrom(source: any = {}) {
	        return new AuthResponse(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.accessToken = source["accessToken"];
	        this.refreshToken = source["refreshToken"];
	        this.tokenType = source["tokenType"];
	        this.expiresIn = source["expiresIn"];
	        this.expiresAt = this.convertValues(source["expiresAt"], null);
	        this.user = this.convertValues(source["user"], User);
	        this.agency = this.convertValues(source["agency"], Agency);
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
	export class Streamer {
	    streamerId: string;
	    bigoId: string;
	    bigoRoomId: string;
	    name: string;
	    avatar: string;
	    bindingGift: string;
	
	    static createFrom(source: any = {}) {
	        return new Streamer(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.streamerId = source["streamerId"];
	        this.bigoId = source["bigoId"];
	        this.bigoRoomId = source["bigoRoomId"];
	        this.name = source["name"];
	        this.avatar = source["avatar"];
	        this.bindingGift = source["bindingGift"];
	    }
	}
	export class Team {
	    teamId: string;
	    name: string;
	    bindingGift: string;
	    scoreMultipliers: Record<string, number>;
	    streamers: Streamer[];
	
	    static createFrom(source: any = {}) {
	        return new Team(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.teamId = source["teamId"];
	        this.name = source["name"];
	        this.bindingGift = source["bindingGift"];
	        this.scoreMultipliers = source["scoreMultipliers"];
	        this.streamers = this.convertValues(source["streamers"], Streamer);
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
	export class SessionInfo {
	    sessionId: string;
	    startTime: number;
	    status: string;
	
	    static createFrom(source: any = {}) {
	        return new SessionInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.sessionId = source["sessionId"];
	        this.startTime = source["startTime"];
	        this.status = source["status"];
	    }
	}
	export class Config {
	    roomId: string;
	    agencyId: number;
	    session: SessionInfo;
	    teams: Team[];
	
	    static createFrom(source: any = {}) {
	        return new Config(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.roomId = source["roomId"];
	        this.agencyId = source["agencyId"];
	        this.session = this.convertValues(source["session"], SessionInfo);
	        this.teams = this.convertValues(source["teams"], Team);
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
	    bigoId: string;
	    bigoRoomId: string;
	    status: string;
	    lastMessageAt?: number;
	    messagesReceived?: number;
	    error?: string;
	
	    static createFrom(source: any = {}) {
	        return new ConnectionStatus(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.bigoId = source["bigoId"];
	        this.bigoRoomId = source["bigoRoomId"];
	        this.status = source["status"];
	        this.lastMessageAt = source["lastMessageAt"];
	        this.messagesReceived = source["messagesReceived"];
	        this.error = source["error"];
	    }
	}
	
	
	
	
	export class ValidateTrialResponse {
	    allowed: boolean;
	    message: string;
	    blockedBigoIds: string[];
	    reason?: string;
	
	    static createFrom(source: any = {}) {
	        return new ValidateTrialResponse(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.allowed = source["allowed"];
	        this.message = source["message"];
	        this.blockedBigoIds = source["blockedBigoIds"];
	        this.reason = source["reason"];
	    }
	}
	export class ValidateTrialStreamer {
	    bigoId: string;
	    bigoRoomId: string;
	
	    static createFrom(source: any = {}) {
	        return new ValidateTrialStreamer(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.bigoId = source["bigoId"];
	        this.bigoRoomId = source["bigoRoomId"];
	    }
	}

}

export namespace profile {
	
	export class Profile {
	    id: string;
	    name: string;
	    roomId: string;
	    // Go type: time
	    createdAt: any;
	    // Go type: time
	    updatedAt: any;
	    // Go type: time
	    lastUsedAt?: any;
	    config: api.Config;
	
	    static createFrom(source: any = {}) {
	        return new Profile(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.roomId = source["roomId"];
	        this.createdAt = this.convertValues(source["createdAt"], null);
	        this.updatedAt = this.convertValues(source["updatedAt"], null);
	        this.lastUsedAt = this.convertValues(source["lastUsedAt"], null);
	        this.config = this.convertValues(source["config"], api.Config);
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

