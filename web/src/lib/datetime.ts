// datetime-local requires wall time, not an ISO timestamp with its zone removed.
export function localDateTime(value: string | Date): string {
 const date = new Date(value);
 const pad = (n: number) => String(n).padStart(2, '0');
 return `${date.getFullYear()}-${pad(date.getMonth()+1)}-${pad(date.getDate())}T${pad(date.getHours())}:${pad(date.getMinutes())}`;
}
