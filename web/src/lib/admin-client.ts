import {csrfHeaders,idempotencyKey} from '$lib/api/client';
export async function adminGet(path:string){const r=await fetch(path,{credentials:'include',cache:'no-store'});const b=await r.json();if(!r.ok)throw new Error(b.detail||'Details could not be loaded');return b}
export async function adminPost(path:string,body:unknown){const r=await fetch(path,{method:'POST',credentials:'include',headers:{'Content-Type':'application/json','Idempotency-Key':idempotencyKey(),...csrfHeaders()},body:JSON.stringify(body)});const b=await r.json();if(!r.ok)throw new Error(b.detail||'Change could not be saved');return b}
export function localTime(value:string){return new Intl.DateTimeFormat('en-NG',{dateStyle:'medium',timeStyle:'short',timeZone:'Africa/Lagos'}).format(new Date(value))}
export function lagosISO(value:string){return new Date(`${value}:00+01:00`).toISOString()}
export function localInput(value:string){return new Date(new Date(value).getTime()+3600000).toISOString().slice(0,16)}
