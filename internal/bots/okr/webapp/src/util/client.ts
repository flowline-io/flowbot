import {AppClient} from "../client";

export function Client()  {
	let global = {base: "http://127.0.0.1:6060/service", flag: "dev"}
	// // @ts-ignore
	// if (Global != undefined) {// eslint-disable-line
	// 	// @ts-ignore
	// 	global = Global// eslint-disable-line
	// }
	return new AppClient({
		BASE: global.base,
		TOKEN: global.flag,
	})
}
