export namespace models {
	
	export class DeleteRequest {
	    record_id: string;
	    address: string;
	    signature: string;
	
	    static createFrom(source: any = {}) {
	        return new DeleteRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.record_id = source["record_id"];
	        this.address = source["address"];
	        this.signature = source["signature"];
	    }
	}
	export class KeyGenRequest {
	    manager_addr: string;
	    room: string;
	    threshold: number;
	    parties: number;
	    party_index: number;
	    record_id: string;
	    filename: string;
	
	    static createFrom(source: any = {}) {
	        return new KeyGenRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.manager_addr = source["manager_addr"];
	        this.room = source["room"];
	        this.threshold = source["threshold"];
	        this.parties = source["parties"];
	        this.party_index = source["party_index"];
	        this.record_id = source["record_id"];
	        this.filename = source["filename"];
	    }
	}
	export class SignRequest {
	    manager_addr: string;
	    room: string;
	    parties: string;
	    signing_index: number;
	    message_hash: string;
	    filename: string;
	    encrypted_shard: string;
	    record_id: string;
	    address: string;
	    signature: string;
	
	    static createFrom(source: any = {}) {
	        return new SignRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.manager_addr = source["manager_addr"];
	        this.room = source["room"];
	        this.parties = source["parties"];
	        this.signing_index = source["signing_index"];
	        this.message_hash = source["message_hash"];
	        this.filename = source["filename"];
	        this.encrypted_shard = source["encrypted_shard"];
	        this.record_id = source["record_id"];
	        this.address = source["address"];
	        this.signature = source["signature"];
	    }
	}

}

