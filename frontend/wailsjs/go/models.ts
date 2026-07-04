export namespace color {
	
	export class ChannelSettings {
	    brightness: number;
	    contrast: number;
	    gamma: number;
	
	    static createFrom(source: any = {}) {
	        return new ChannelSettings(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.brightness = source["brightness"];
	        this.contrast = source["contrast"];
	        this.gamma = source["gamma"];
	    }
	}
	export class Settings {
	    temperature: number;
	    brightness: number;
	    contrast: number;
	    gamma: number;
	    saturation: number;
	    hue: number;
	    red: ChannelSettings;
	    green: ChannelSettings;
	    blue: ChannelSettings;
	    profile: string;
	
	    static createFrom(source: any = {}) {
	        return new Settings(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.temperature = source["temperature"];
	        this.brightness = source["brightness"];
	        this.contrast = source["contrast"];
	        this.gamma = source["gamma"];
	        this.saturation = source["saturation"];
	        this.hue = source["hue"];
	        this.red = this.convertValues(source["red"], ChannelSettings);
	        this.green = this.convertValues(source["green"], ChannelSettings);
	        this.blue = this.convertValues(source["blue"], ChannelSettings);
	        this.profile = source["profile"];
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

export namespace display {
	
	export class Display {
	    id: string;
	    name: string;
	
	    static createFrom(source: any = {}) {
	        return new Display(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	    }
	}

}

export namespace main {
	
	export class State {
	    settings: color.Settings;
	    displays: display.Display[];
	    selected: string;
	    gammaBackend: string;
	    vendorBackend: string;
	    saturationAvailable: boolean;
	    hueAvailable: boolean;
	    hueMin: number;
	    hueMax: number;
	    saturationDefault: number;
	    presets: string[];
	    userPresets: string[];
	    autostart: boolean;
	    autostartAvailable: boolean;
	    errors: string;
	    version: string;
	
	    static createFrom(source: any = {}) {
	        return new State(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.settings = this.convertValues(source["settings"], color.Settings);
	        this.displays = this.convertValues(source["displays"], display.Display);
	        this.selected = source["selected"];
	        this.gammaBackend = source["gammaBackend"];
	        this.vendorBackend = source["vendorBackend"];
	        this.saturationAvailable = source["saturationAvailable"];
	        this.hueAvailable = source["hueAvailable"];
	        this.hueMin = source["hueMin"];
	        this.hueMax = source["hueMax"];
	        this.saturationDefault = source["saturationDefault"];
	        this.presets = source["presets"];
	        this.userPresets = source["userPresets"];
	        this.autostart = source["autostart"];
	        this.autostartAvailable = source["autostartAvailable"];
	        this.errors = source["errors"];
	        this.version = source["version"];
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

