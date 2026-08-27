export namespace katplugins {
	
	export class KatPluginsInfo {
	    state: string;
	    statusText: string;
	    statusColor: string;
	    buttonText: string;
	    buttonColor: string;
	    mainButtonVisible: boolean;
	    uninstallVisible: boolean;
	    currentVersion: string;
	    latestVersion: string;
	
	    static createFrom(source: any = {}) {
	        return new KatPluginsInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.state = source["state"];
	        this.statusText = source["statusText"];
	        this.statusColor = source["statusColor"];
	        this.buttonText = source["buttonText"];
	        this.buttonColor = source["buttonColor"];
	        this.mainButtonVisible = source["mainButtonVisible"];
	        this.uninstallVisible = source["uninstallVisible"];
	        this.currentVersion = source["currentVersion"];
	        this.latestVersion = source["latestVersion"];
	    }
	}

}

export namespace lolconfig {
	
	export class LolConfigInfo {
	    state: string;
	    statusText: string;
	    statusColor: string;
	    buttonText: string;
	    canToggle: boolean;
	
	    static createFrom(source: any = {}) {
	        return new LolConfigInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.state = source["state"];
	        this.statusText = source["statusText"];
	        this.statusColor = source["statusColor"];
	        this.buttonText = source["buttonText"];
	        this.canToggle = source["canToggle"];
	    }
	}

}

export namespace updater {
	
	export class UpdateInfo {
	    currentVersion: string;
	    latestVersion: string;
	    updateAvailable: boolean;
	    downloadUrl: string;
	    releaseUrl: string;
	
	    static createFrom(source: any = {}) {
	        return new UpdateInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.currentVersion = source["currentVersion"];
	        this.latestVersion = source["latestVersion"];
	        this.updateAvailable = source["updateAvailable"];
	        this.downloadUrl = source["downloadUrl"];
	        this.releaseUrl = source["releaseUrl"];
	    }
	}

}

