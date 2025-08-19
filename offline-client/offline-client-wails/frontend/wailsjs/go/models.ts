export namespace main {
	
	export class DeleteMessageRequest {
	    user_name: string;
	    address: string;
	    signature: number[];
	
	    static createFrom(source: any = {}) {
	        return new DeleteMessageRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.user_name = source["user_name"];
	        this.address = source["address"];
	        this.signature = source["signature"];
	    }
	}
	export class KeyGenerationRequest {
	    threshold: number;
	    parties: number;
	    index: number;
	    user_name: string;
	
	    static createFrom(source: any = {}) {
	        return new KeyGenerationRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.threshold = source["threshold"];
	        this.parties = source["parties"];
	        this.index = source["index"];
	        this.user_name = source["user_name"];
	    }
	}
	export class SignMessageRequest {
	    message: string;
	    parties: string;
	    user_name: string;
	    address: string;
	    encrypted_key: number[];
	    signature: number[];
	
	    static createFrom(source: any = {}) {
	        return new SignMessageRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.message = source["message"];
	        this.parties = source["parties"];
	        this.user_name = source["user_name"];
	        this.address = source["address"];
	        this.encrypted_key = source["encrypted_key"];
	        this.signature = source["signature"];
	    }
	}

}

