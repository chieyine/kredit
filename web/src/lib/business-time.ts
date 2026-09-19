// Kredit's business day is Lagos, wherever the operator happens to be.
export function localTime(value:string){
  const at = new Date(value);
  return Number.isFinite(at.getTime())
    ? new Intl.DateTimeFormat('en-NG',{dateStyle:'medium',timeStyle:'short',timeZone:'Africa/Lagos'}).format(at)
    : 'Time unavailable';
}
/** A `datetime-local` value the operator reads as Lagos wall time. */
export function localInput(value:string){
  const at = new Date(value);
  if(!Number.isFinite(at.getTime())) return '';
  const parts = new Intl.DateTimeFormat('en-CA',{timeZone:'Africa/Lagos',year:'numeric',month:'2-digit',day:'2-digit',hour:'2-digit',minute:'2-digit',hourCycle:'h23'}).formatToParts(at);
  const get = (type:string) => parts.find(part => part.type === type)?.value ?? '';
  return `${get('year')}-${get('month')}-${get('day')}T${get('hour')}:${get('minute')}`;
}
/** The Lagos wall time the operator typed, as an exact instant. */
export function lagosISO(value:string){
  if (!/^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}$/.test(value)) throw new Error('Enter a valid date and time.');
  const at = new Date(`${value}:00+01:00`);
  if(!Number.isFinite(at.getTime()) || new Date(at.getTime() + 3600000).toISOString().slice(0,16) !== value) throw new Error('Enter a valid date and time.');
  return at.toISOString();
}
