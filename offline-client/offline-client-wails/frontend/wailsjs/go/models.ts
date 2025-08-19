export namespace models {
	
	export class DeleteRequest {
	    username: string;
	    address: string;
	    signature: string;
	
	    static createFrom(source: any = {}) {
	        return new DeleteRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.username = source["username"];
	        this.address = source["address"];
	        this.signature = source["signature"];
	    }
	}
	export class KeyGenRequest {
	    threshold: number;
	    parties: number;
	    index: number;
	    filename: string;
	    username: string;
	
	    static createFrom(source: any = {}) {
	        return new KeyGenRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.threshold = source["threshold"];
	        this.parties = source["parties"];
	        this.index = source["index"];
	        this.filename = source["filename"];
	        this.username = source["username"];
	    }
	}
	export class SignRequest {
	    parties: string;
	    data: string;
	    filename: string;
	    encryptedKey: string;
	    userName: string;
	    address: string;
	    signature: string;
	
	    static createFrom(source: any = {}) {
	        return new SignRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.parties = source["parties"];
	        this.data = source["data"];
	        this.filename = source["filename"];
	        this.encryptedKey = source["encryptedKey"];
	        this.userName = source["userName"];
	        this.address = source["address"];
	        this.signature = source["signature"];
	    }
	}

}

